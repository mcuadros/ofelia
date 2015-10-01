package core

import (
	"errors"
	"sync"

	"github.com/robfig/cron"
)

var ErrEmptyScheduler error = errors.New("unable to start a empty scheduler.")
var ErrEmptySchedule error = errors.New("unable to add a job with a empty schedule.")

type Scheduler struct {
	Jobs []Job

	cron      *cron.Cron
	wg        sync.WaitGroup
	isRunning bool
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		cron: cron.New(),
	}
}

func (s *Scheduler) AddJob(j Job) error {
	if j.GetSchedule() == "" {
		return ErrEmptySchedule
	}

	err := s.cron.AddJob(j.GetSchedule(), &jobWrapper{s, j})
	if err != nil {
		return err
	}

	s.Jobs = append(s.Jobs, j)
	return nil
}

func (s *Scheduler) Start() error {
	if len(s.Jobs) == 0 {
		return ErrEmptyScheduler
	}

	s.isRunning = true
	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() error {
	s.wg.Wait()
	s.cron.Stop()
	s.isRunning = false

	return nil
}

func (s *Scheduler) IsRunning() bool {
	return s.isRunning
}

type jobWrapper struct {
	s *Scheduler
	j Job
}

func (w *jobWrapper) Run() {
	w.s.wg.Add(1)
	defer w.s.wg.Done()

	e := NewExecution()
	w.start(e)

	ctx := NewContext(w.s, w.j, e)
	err := ctx.Next()
	w.stop(e, err)
}

func (w *jobWrapper) start(e *Execution) {
	e.Start()
	w.j.AddHistory(e)
	w.j.NotifyStart()
}

func (w *jobWrapper) stop(e *Execution, err error) {
	e.Stop(err)
	w.j.NotifyStop()
}
