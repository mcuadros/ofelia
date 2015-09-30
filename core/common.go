package core

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

var ErrSkippedExecution = errors.New("skipped execution")

type Job interface {
	GetName() string
	GetSchedule() string
	Run()
	Running() int32
	History() []*Execution
	SetAfterStart(h Hook)
	SetAfterStop(h Hook)
}

type BareJob struct {
	Schedule     string
	Name         string
	AllowOverlap bool `gcfg:"allow-overlap" default:"true"`

	running int32
	history []*Execution
	lock    sync.Mutex
	hooks   struct {
		afterStart Hook
		afterStop  Hook
	}
}

func (j *BareJob) GetName() string {
	return j.Name
}

func (j *BareJob) GetSchedule() string {
	return j.Schedule
}

func (j *BareJob) Running() int32 {
	return atomic.LoadInt32(&j.running)
}

func (j *BareJob) SetAfterStart(h Hook) {
	j.hooks.afterStart = h
}

func (j *BareJob) SetAfterStop(h Hook) {
	j.hooks.afterStop = h
}

func (j *BareJob) History() []*Execution {
	return j.history
}

func (j *BareJob) Start() *Execution {
	e := NewExecution()
	defer j.callAfterStart(e)

	j.lock.Lock()
	j.history = append(j.history, e)
	j.lock.Unlock()

	e.Start()
	if !j.AllowOverlap && j.Running() != 0 {
		e.Stop(ErrSkippedExecution)
		return nil
	}

	atomic.AddInt32(&j.running, 1)

	return e
}

func (j *BareJob) callAfterStart(e *Execution) {
	if j.hooks.afterStart == nil {
		return
	}

	j.hooks.afterStart(e)
}

func (j *BareJob) Stop(e *Execution, err error) {
	defer j.callAfterStop(e)

	e.Stop(err)
	atomic.AddInt32(&j.running, -1)
}

func (j *BareJob) callAfterStop(e *Execution) {
	if j.hooks.afterStop == nil {
		return
	}

	j.hooks.afterStop(e)
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

	if err != nil && err != ErrSkippedExecution {
		e.Error = err
		e.Failed = true
	} else if err == ErrSkippedExecution {
		e.Skipped = true
		e.Duration = 0
	}
}

type Hook func(*Execution)

func randomID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)
}
