package core

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/distribution/reference"
	dockercliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/moby/moby/api/pkg/authconfig"
	"github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"
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
	Ctx       context.Context
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
		Ctx:         context.Background(),
		Scheduler:   s,
		Logger:      s.Logger,
		Job:         j,
		Execution:   e,
		middlewares: j.Middlewares(),
	}
}

func (c *Context) Context() context.Context {
	if c.Ctx != nil {
		return c.Ctx
	}
	return context.Background()
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

	OutputStream, ErrorStream *streamBuffer `json:"-"`
}

// NewExecution returns a new Execution, with a random ID.
func NewExecution() *Execution {
	return &Execution{
		ID:           randomID(),
		OutputStream: newStreamBuffer(maxStreamSize),
		ErrorStream:  newStreamBuffer(maxStreamSize),
	}
}

// streamBuffer is a lazy ring buffer that keeps only the last limit bytes
// written. The backing array grows on demand (doubling up to limit), so
// executions whose output stays small avoid reserving the full limit up front,
// which previously forced two 10 MiB allocations per execution.
type streamBuffer struct {
	limit int64

	written int64 // total bytes ever written
	head    int   // index in data where the next byte is written
	data    []byte
}

func newStreamBuffer(limit int64) *streamBuffer {
	return &streamBuffer{limit: limit}
}

// grow enlarges the backing array to hold at least n bytes, doubling its
// capacity up to limit. It is only called while the buffer is not full, when
// the retained bytes are data[:head] in write order.
func (b *streamBuffer) grow(n int) {
	capacity := max(len(b.data), min(256, int(b.limit)))
	for capacity < n {
		capacity *= 2
	}
	// n never exceeds limit, so capping here keeps capacity >= n while
	// avoiding an overshoot past the limit.
	capacity = min(capacity, int(b.limit))

	data := make([]byte, capacity)
	copy(data, b.data[:b.head])
	b.data = data
}

// Write appends buf to the buffer, keeping only the last limit bytes. It never
// fails and always reports len(buf), so it is safe as an io.Writer for job
// output streams.
func (b *streamBuffer) Write(buf []byte) (int, error) {
	n := len(buf)

	if b.limit <= 0 {
		// Nothing can be retained; just account for the bytes.
		b.written += int64(n)
		return n, nil
	}

	if b.written < b.limit {
		// Not full yet: append, growing the backing array on demand.
		m := min(n, int(b.limit-b.written))
		if len(b.data)-b.head < m {
			b.grow(b.head + m)
		}
		copy(b.data[b.head:], buf[:m])
		b.head += m
		b.written += int64(m)
		buf = buf[m:]
		if b.written < b.limit || len(buf) == 0 {
			if b.written == b.limit {
				b.head = 0
			}
			return n, nil
		}
		b.head = 0 // now exactly full: the oldest byte is at the start
	}

	// Full: overwrite the oldest bytes from head, wrapping around the ring.
	b.written += int64(len(buf))
	full := len(b.data)
	if len(buf) >= full {
		// The write covers the whole ring: only its tail survives.
		buf = buf[len(buf)-full:]
	}
	first := min(len(buf), full-b.head)
	copy(b.data[b.head:], buf[:first])
	copy(b.data, buf[first:])
	b.head = (b.head + len(buf)) % full
	return n, nil
}

// Size returns the maximum number of bytes the buffer keeps.
func (b *streamBuffer) Size() int64 {
	return b.limit
}

// TotalWritten returns the total number of bytes written so far.
func (b *streamBuffer) TotalWritten() int64 {
	return b.written
}

// Bytes returns a copy of the retained bytes, in write order.
func (b *streamBuffer) Bytes() []byte {
	if b.written < b.limit {
		return append([]byte(nil), b.data[:b.head]...)
	}
	out := make([]byte, len(b.data))
	n1 := copy(out, b.data[b.head:])
	copy(out[n1:], b.data[:b.head])
	return out
}

// String returns the retained bytes as a string.
func (b *streamBuffer) String() string {
	return string(b.Bytes())
}

var _ io.Writer = (*streamBuffer)(nil)

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
	dockerCfgMu     sync.Mutex
	dockerCfg       *configfile.ConfigFile
	dockerCfgLoaded bool
)

func loadDockerConfig() *configfile.ConfigFile {
	dockerCfgMu.Lock()
	defer dockerCfgMu.Unlock()
	if dockerCfgLoaded {
		return dockerCfg
	}
	cfg, err := dockercliconfig.Load(dockercliconfig.Dir())
	if err != nil {
		return nil
	}
	dockerCfg = cfg
	dockerCfgLoaded = true
	return dockerCfg
}

func buildPullOptions(img string) (string, string) {
	named, err := reference.ParseNormalizedNamed(img)
	if err != nil {
		return img + ":latest", ""
	}
	named = reference.TagNameOnly(named)
	ref := named.String()
	domain := reference.Domain(named)
	encodedAuth := buildEncodedAuth(domain)
	return ref, encodedAuth
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

	encoded, err := authconfig.Encode(regAuth)
	if err != nil {
		return ""
	}
	return encoded
}

func pullImage(dc DockerClient, image string, ctx context.Context) error {
	ref, encodedAuth := buildPullOptions(image)
	resp, err := dc.ImagePull(ctx, ref, client.ImagePullOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return fmt.Errorf("error pulling image %q: %s", image, err)
	}
	defer resp.Close()
	if err := resp.Wait(ctx); err != nil {
		return fmt.Errorf("error pulling image %q: %s", image, err)
	}
	return nil
}
