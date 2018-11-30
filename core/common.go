package core

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

var (
	// ErrSkippedExecution pass this error to `Execution.Stop` if you wish to mark
	// it as skipped.
	ErrSkippedExecution = errors.New("skipped execution")
	ErrUnexpected       = errors.New("error unexpected, docker has returned exit code -1, maybe wrong user?")
	ErrMaxTimeRunning   = errors.New("the job has exceed the maximum allowed time running.")
)

type Job interface {
	GetName() string
	GetSchedule() string
	GetCommand() string
	Middlewares() []Middleware
	Use(...Middleware)
	Run(*Context) error
	Running() int32
	History() []*Execution
	AddHistory(...*Execution)
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
	c.Job.AddHistory(c.Execution)
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

func (c *Context) Log(msg string) {
	msg = fmt.Sprintf("[Job %q (%s)] %s", c.Job.GetName(), c.Execution.ID, msg)
	switch {
	case c.Execution.Failed:
		c.Logger.Errorf(msg)
	case c.Execution.Skipped:
		c.Logger.Warningf(msg)
	default:
		c.Logger.Noticef(msg)
	}
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

	OutputStream, ErrorStream io.ReadWriter `json:"-"`
}

// NewExecution returns a new Execution, with a random ID
func NewExecution() *Execution {
	return &Execution{
		ID:           randomID(),
		OutputStream: bytes.NewBuffer(nil),
		ErrorStream:  bytes.NewBuffer(nil),
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
	Criticalf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Noticef(format string, args ...interface{})
	Warningf(format string, args ...interface{})
}

func randomID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)
}

func buildPullOptions(image string) (docker.PullImageOptions, docker.AuthConfiguration) {
	tag := "latest"
	registry := ""

	parts := strings.Split(image, ":")
	if len(parts) == 2 {
		tag = parts[1]
	}

	name := parts[0]
	parts = strings.Split(name, "/")
	if len(parts) > 2 {
		registry = parts[0]
	}

	return docker.PullImageOptions{
		Repository: name,
		Registry:   registry,
		Tag:        tag,
	}, buildAuthConfiguration(registry)
}

func buildAuthConfiguration(registry string) docker.AuthConfiguration {
	var auth docker.AuthConfiguration
	if dockercfg == nil {
		return auth
	}

	auth, _ = dockercfg.Configs[registry]
	return auth
}
