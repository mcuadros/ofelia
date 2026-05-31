package core

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/armon/circbuf"
	dockercliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/moby/moby/api/types/registry"
)

var (
	// ErrSkippedExecution pass this error to `Execution.Stop` if you wish to mark
	// it as skipped.
	ErrSkippedExecution   = errors.New("skipped execution")
	ErrUnexpected         = errors.New("error unexpected, docker has returned exit code -1, maybe wrong user?")
	ErrMaxTimeRunning     = errors.New("the job has exceed the maximum allowed time running.")
	ErrLocalImageNotFound = errors.New("couldn't find image on the host")
)

const (
	// maximum size of a stdout/stderr stream to be kept in memory and optional stored/sent via mail
	maxStreamSize = 10 * 1024 * 1024
)

type Job interface {
	GetName() string
	GetSchedule() string
	GetCommand() string
	GetCronJobID() int
	SetCronJobID(int)
	Middlewares() []Middleware
	Use(...Middleware)
	Run(*Context) error
	Running() int32
	NotifyStart()
	NotifyStop()
}

type Context struct {
	Scheduler *Scheduler
	Logger    Logger
	Job       Job
	Execution *Execution

	current     int
	executed    bool
	middlewares []Middleware
}

func NewContext(s *Scheduler, j Job, e *Execution) *Context {
	return &Context{
		Scheduler:   s,
		Logger:      s.Logger,
		Job:         j,
		Execution:   e,
		middlewares: j.Middlewares(),
	}
}

func (c *Context) Start() {
	c.Execution.Start()
	c.Job.NotifyStart()
}

func (c *Context) Next() error {
	if err := c.doNext(); err != nil || c.executed {
		c.Stop(err)
	}

	return nil
}

func (c *Context) doNext() error {
	for {
		m, end := c.getNext()
		if end {
			break
		}

		if !c.Execution.IsRunning && !m.ContinueOnStop() {
			continue
		}

		return m.Run(c)
	}

	if !c.Execution.IsRunning {
		return nil
	}

	c.executed = true
	return c.Job.Run(c)
}

func (c *Context) getNext() (Middleware, bool) {
	if c.current >= len(c.middlewares) {
		return nil, true
	}

	c.current++
	return c.middlewares[c.current-1], false
}

func (c *Context) Stop(err error) {
	if !c.Execution.IsRunning {
		return
	}

	c.Execution.Stop(err)
	c.Job.NotifyStop()
}

func (c *Context) Log(msg string, args ...any) {
	defaultArgs := []any{"job", c.Job.GetName(), "execution", c.Execution.ID}

	switch {
	case c.Execution.Failed:
		c.Logger.Error(msg, append(defaultArgs, args...)...)
	case c.Execution.Skipped:
		c.Logger.Warning(msg, append(defaultArgs, args...)...)
	default:
		c.Logger.Info(msg, append(defaultArgs, args...)...)
	}
}

func (c *Context) Warn(msg string, args ...any) {
	defaultArgs := []any{"job", c.Job.GetName(), "execution", c.Execution.ID}
	c.Logger.Warning(msg, append(defaultArgs, args...)...)
}

// Execution contains all the information relative to a Job execution.
type Execution struct {
	ID        string
	Date      time.Time
	Duration  time.Duration
	IsRunning bool
	Failed    bool
	Skipped   bool
	Error     error

	OutputStream, ErrorStream *circbuf.Buffer `json:"-"`
}

// NewExecution returns a new Execution, with a random ID
func NewExecution() *Execution {
	bufOut, _ := circbuf.NewBuffer(maxStreamSize)
	bufErr, _ := circbuf.NewBuffer(maxStreamSize)
	return &Execution{
		ID:           randomID(),
		OutputStream: bufOut,
		ErrorStream:  bufErr,
	}
}

// Start start the exection, initialize the running flags and the start date.
func (e *Execution) Start() {
	e.IsRunning = true
	e.Date = time.Now()
}

// Stop stops the executions, if a ErrSkippedExecution is given the exection
// is mark as skipped, if any other error is given the exection is mark as
// failed. Also mark the exection as IsRunning false and save the duration time
func (e *Execution) Stop(err error) {
	e.IsRunning = false
	e.Duration = time.Since(e.Date)

	if err != nil && err != ErrSkippedExecution {
		e.Error = err
		e.Failed = true
	} else if err == ErrSkippedExecution {
		e.Skipped = true
	}
}

// Middleware can wrap any job execution, allowing to execution code before
// or/and after of each `Job.Run`
type Middleware interface {
	// Run is called instead of the original `Job.Run`, you MUST call to `ctx.Run`
	// inside of the middleware `Run` function otherwise you will broken the
	// Job workflow.
	Run(*Context) error
	// ContinueOnStop,  If return true the Run function will be called even if
	// the execution is stopped
	ContinueOnStop() bool
}

type middlewareContainer struct {
	m     map[string]Middleware
	order []string
}

func (c *middlewareContainer) Use(ms ...Middleware) {
	if c.m == nil {
		c.m = make(map[string]Middleware, 0)
	}

	for _, m := range ms {
		if m == nil {
			continue
		}

		t := reflect.TypeOf(m).String()
		if _, ok := c.m[t]; ok {
			continue
		}

		c.order = append(c.order, t)
		c.m[t] = m
	}
}

func (c *middlewareContainer) Middlewares() []Middleware {
	var ms []Middleware
	for _, t := range c.order {
		ms = append(ms, c.m[t])
	}

	return ms
}

type Logger interface {
	Debug(str string, args ...any)
	Error(str string, args ...any)
	Info(str string, args ...any)
	Warning(str string, args ...any)
}

func randomID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)
}

// --- Docker auth and image helpers ---

var (
	dockerCfgOnce sync.Once
	dockerCfg     *configfile.ConfigFile
)

func loadDockerConfig() *configfile.ConfigFile {
	dockerCfgOnce.Do(func() {
		cfg, err := dockercliconfig.Load(dockercliconfig.Dir())
		if err != nil {
			return
		}
		dockerCfg = cfg
	})
	return dockerCfg
}

func parseRepositoryTag(repoTag string) (repository, tag string) {
	if n := strings.IndexRune(repoTag, '@'); n >= 0 {
		repoTag = repoTag[:n]
	}
	n := strings.LastIndexByte(repoTag, ':')
	if n < 0 {
		return repoTag, ""
	}
	if strings.Contains(repoTag[n+1:], "/") {
		return repoTag, ""
	}
	return repoTag[:n], repoTag[n+1:]
}

func buildPullOptions(img string) (string, string) {
	repository, tag := parseRepositoryTag(img)
	registry := parseRegistry(repository)

	if tag == "" {
		tag = "latest"
	}

	ref := repository + ":" + tag
	encodedAuth := buildEncodedAuth(registry)
	return ref, encodedAuth
}

func parseRegistry(repository string) string {
	parts := strings.Split(repository, "/")
	if len(parts) < 2 {
		return ""
	}

	if strings.ContainsAny(parts[0], ".:") || len(parts) > 2 {
		return parts[0]
	}

	return ""
}

func buildEncodedAuth(reg string) string {
	cfg := loadDockerConfig()
	if cfg == nil {
		return ""
	}

	hostname := reg
	if hostname == "" {
		hostname = "https://index.docker.io/v1/"
	}

	authCfg, err := cfg.GetAuthConfig(hostname)
	if err != nil {
		return ""
	}

	if authCfg.Username == "" && authCfg.Password == "" && authCfg.IdentityToken == "" {
		return ""
	}

	regAuth := registry.AuthConfig{
		Username:      authCfg.Username,
		Password:      authCfg.Password,
		ServerAddress: authCfg.ServerAddress,
		IdentityToken: authCfg.IdentityToken,
		RegistryToken: authCfg.RegistryToken,
	}

	encoded, err := encodeAuthConfig(regAuth)
	if err != nil {
		return ""
	}
	return encoded
}

func encodeAuthConfig(authConfig registry.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
