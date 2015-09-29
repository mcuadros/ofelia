package core

import "time"

type Job interface {
	Name() string
	Spec() string
	IsRunning() bool
	LastError() error
	LastExecution() time.Time
	LastDuration() time.Duration
	Run()
}

type BasicJob struct {
	name          string
	spec          string
	isRunning     bool
	lastError     error
	lastExecution time.Time
	lastDuration  time.Duration
}

func (j *BasicJob) Name() string {
	return j.name
}

func (j *BasicJob) Spec() string {
	return j.spec
}

func (j *BasicJob) SetName(name string) {
	j.name = name
}

func (j *BasicJob) SetSpec(spec string) {
	j.spec = spec
}

func (j *BasicJob) IsRunning() bool {
	return j.isRunning
}

func (j *BasicJob) LastError() error {
	return j.lastError
}

func (j *BasicJob) LastExecution() time.Time {
	return j.lastExecution
}

func (j *BasicJob) LastDuration() time.Duration {
	return j.lastDuration
}

func (j *BasicJob) MarkStart() {
	j.isRunning = true
	j.lastExecution = time.Now()
}

func (j *BasicJob) MarkStop(err error) {
	j.isRunning = false
	j.lastError = err
	j.lastDuration = time.Since(j.lastExecution)
}
