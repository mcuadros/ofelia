package core

import (
	"sync"
	"sync/atomic"
)

type BareJob struct {
	Schedule     string
	Name         string
	Command      string
	RunOnStartup string `default:"false" gcfg:"run-on-startup" mapstructure:"run-on-startup"`

	middlewareContainer
	running int32
	lock    sync.Mutex
	history []*Execution
}

func (j *BareJob) GetName() string {
	return j.Name
}

func (j *BareJob) GetSchedule() string {
	return j.Schedule
}

func (j *BareJob) GetCommand() string {
	return j.Command
}

func (j *BareJob) GetRunOnStartup() string {
	return j.RunOnStartup
}

func (j *BareJob) Running() int32 {
	return atomic.LoadInt32(&j.running)
}

func (j *BareJob) NotifyStart() {
	atomic.AddInt32(&j.running, 1)
}

func (j *BareJob) NotifyStop() {
	atomic.AddInt32(&j.running, -1)
}
