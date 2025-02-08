package core

import (
	"sync"
	"sync/atomic"

	"github.com/gohugoio/hashstructure"
)

type BareJob struct {
	Schedule string
	Name     string
	Command  string

	middlewareContainer
	running int32
	lock    sync.Mutex
	history []*Execution
	cronID  int
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

func (j *BareJob) GetCronJobID() int {
	return j.cronID
}

func (j *BareJob) SetCronJobID(id int) {
	j.cronID = id
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

// Returns a hash of all the job attributes. Used to detect changes
// unexported struct fields are ignored - https://pkg.go.dev/github.com/gohugoio/hashstructure#Hash
func (j *BareJob) Hash() uint64 {
	hash, _ := hashstructure.Hash(j, nil)
	return hash
}
