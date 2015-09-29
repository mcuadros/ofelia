package core

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var ErrSkippedExecution = errors.New("skipped execution")

type Job interface {
	Name() string
	Spec() string
	Run()
	Running() int32
	History() []*Execution
}

type BasicJob struct {
	AllowOverlap bool

	name    string
	spec    string
	running int32
	history []*Execution
	l       sync.Mutex
}

func (j *BasicJob) Name() string {
	return j.name
}

func (j *BasicJob) Spec() string {
	return j.spec
}

func (j *BasicJob) Running() int32 {
	return atomic.LoadInt32(&j.running)
}

func (j *BasicJob) SetName(name string) {
	j.name = name
}

func (j *BasicJob) SetSpec(spec string) {
	j.spec = spec
}

func (j *BasicJob) History() []*Execution {
	return j.history
}

func (j *BasicJob) Start() *Execution {
	e := &Execution{}

	j.l.Lock()
	j.history = append(j.history, e)
	j.l.Unlock()

	e.Start()
	if !j.AllowOverlap && j.Running() != 0 {
		e.Stop(ErrSkippedExecution)
		return nil
	}

	atomic.AddInt32(&j.running, 1)

	return e
}

func (j *BasicJob) Stop(e *Execution, err error) {
	e.Stop(err)
	atomic.AddInt32(&j.running, -1)
}

type Execution struct {
	Date      time.Time
	Duration  time.Duration
	IsRunning bool
	Failed    bool
	Skipped   bool
	Error     error
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
