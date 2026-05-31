package core

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/armon/circbuf"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
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
	dockerAuthOnce sync.Once
	dockerAuth     map[string]registry.AuthConfig
)

func loadDockerAuth() map[string]registry.AuthConfig {
	dockerAuthOnce.Do(func() {
		dockerAuth = make(map[string]registry.AuthConfig)

		for _, path := range dockerConfigPaths() {
			cfgs, err := readDockerAuthFile(path)
			if err != nil {
				continue
			}
			for registry, auth := range cfgs {
				if _, exists := dockerAuth[registry]; exists {
					continue
				}
				dockerAuth[registry] = auth
			}
		}
	})
	return dockerAuth
}

func dockerConfigPaths() []string {
	if dockerConfig := os.Getenv("DOCKER_CONFIG"); dockerConfig != "" {
		return []string{
			filepath.Join(dockerConfig, "plaintext-passwords.json"),
			filepath.Join(dockerConfig, "config.json"),
		}
	}

	home := os.Getenv("HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil
		}
	}

	return []string{
		filepath.Join(home, ".docker", "plaintext-passwords.json"),
		filepath.Join(home, ".docker", "config.json"),
		filepath.Join(home, ".dockercfg"),
	}
}

func readDockerAuthFile(path string) (map[string]registry.AuthConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg struct {
		Auths map[string]registry.AuthConfig `json:"auths"`
	}
	if err := json.Unmarshal(data, &cfg); err == nil && len(cfg.Auths) > 0 {
		return normalizeDockerAuthConfigs(cfg.Auths), nil
	}

	var legacy map[string]registry.AuthConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}
	return normalizeDockerAuthConfigs(legacy), nil
}

func normalizeDockerAuthConfigs(cfgs map[string]registry.AuthConfig) map[string]registry.AuthConfig {
	normalized := make(map[string]registry.AuthConfig, len(cfgs))
	for server, auth := range cfgs {
		if auth.Auth != "" && (auth.Username == "" || auth.Password == "") {
			username, password, err := decodeDockerAuth(auth.Auth)
			if err != nil {
				continue
			}
			auth.Username = username
			auth.Password = password
			auth.Auth = ""
		}

		if isAuthConfigEmpty(auth) {
			continue
		}

		auth.ServerAddress = server
		normalized[server] = auth
	}
	return normalized
}

func decodeDockerAuth(auth string) (username, password string, err error) {
	data, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		data, err = base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(auth)
	}
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(string(data), ":", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid docker auth entry")
	}
	return parts[0], parts[1], nil
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

func buildFindLocalImageOptions(img string) image.ListOptions {
	return image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", img)),
	}
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
	auth := buildAuthConfig(reg)
	if isAuthConfigEmpty(auth) {
		return ""
	}

	encoded, err := registry.EncodeAuthConfig(auth)
	if err != nil {
		return ""
	}
	return encoded
}

func isAuthConfigEmpty(auth registry.AuthConfig) bool {
	return auth.Username == "" &&
		auth.Password == "" &&
		auth.Auth == "" &&
		auth.IdentityToken == "" &&
		auth.RegistryToken == ""
}

func buildAuthConfig(reg string) registry.AuthConfig {
	cfgs := loadDockerAuth()
	if cfgs == nil {
		return registry.AuthConfig{}
	}

	if v, ok := cfgs[reg]; ok {
		return v
	}

	if reg == "" {
		if v, ok := cfgs["https://index.docker.io/v2/"]; ok {
			return v
		}
		if v, ok := cfgs["https://index.docker.io/v1/"]; ok {
			return v
		}
	}

	return registry.AuthConfig{}
}

func consumePullResponse(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	for {
		var msg struct {
			ErrorDetail *struct {
				Message string `json:"message"`
			} `json:"errorDetail,omitempty"`
			ErrorMessage string `json:"error,omitempty"`
		}
		if err := decoder.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if msg.ErrorDetail != nil && msg.ErrorDetail.Message != "" {
			return errors.New(msg.ErrorDetail.Message)
		}
		if msg.ErrorMessage != "" {
			return errors.New(msg.ErrorMessage)
		}
	}
}
