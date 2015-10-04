package core

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"reflect"
	"time"
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
	if !c.Execution.IsRunning {
		return nil
	}

	if err := c.doNext(); err != nil || c.executed {
		c.Stop(err)
	}

	return nil
}

func (c *Context) doNext() error {
	if c.current >= len(c.middlewares) {
		c.executed = true
		return c.Job.Run(c)
	}

	c.current++
	return c.middlewares[c.current-1].Run(c)
}

func (c *Context) Stop(err error) {
	if !c.Execution.IsRunning {
		return
	}

	c.Execution.Stop(err)
	c.Job.NotifyStop()
}

type Execution struct {
	ID        string
	Date      time.Time
	Duration  time.Duration
	IsRunning bool
	Failed    bool
	Skipped   bool
	Error     error

	OutputStream, ErrorStream io.ReadWriter
}

func NewExecution() *Execution {
	return &Execution{
		ID:           randomID(),
		OutputStream: bytes.NewBuffer(nil),
		ErrorStream:  bytes.NewBuffer(nil),
	}
}

func (e *Execution) Start() {
	e.IsRunning = true
	e.Date = time.Now()
}

func (e *Execution) Stop(err error) {
	e.IsRunning = false
	e.Duration = time.Since(e.Date)

	if err != nil {
		e.Error = err
		e.Failed = true
	}
}

type Middleware interface {
	Run(*Context) error
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
	Critical(format string, args ...interface{})
	Debug(format string, args ...interface{})
	Error(format string, args ...interface{})
	Notice(format string, args ...interface{})
	Warning(format string, args ...interface{})
}

func randomID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)
}
