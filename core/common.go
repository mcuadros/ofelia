package core

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
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
	Job       Job
	Execution *Execution

	current     int
	executed    bool
	middlewares []Middleware
}

func NewContext(s *Scheduler, j Job, e *Execution) *Context {
	return &Context{
		Scheduler:   s,
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

type Middleware interface {
	Run(*Context) error
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

func randomID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)
}
