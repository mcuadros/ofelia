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

	cron *cron.Cron
	wg   sync.WaitGroup
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

	s.registerHooks(j)
	err := s.cron.AddJob(j.GetSchedule(), &cronJob{j, &s.wg})
	if err != nil {
		return err
	}

	s.Jobs = append(s.Jobs, j)
	return nil
}

func (s *Scheduler) registerHooks(j Job) {
	j.SetAfterStart(func(e *Execution) {
		AfterStartHook(s, j, e)
	})

	j.SetAfterStop(func(e *Execution) {
		AfterStopHook(s, j, e)
	})
}

func (s *Scheduler) Start() error {
	if len(s.Jobs) == 0 {
		return ErrEmptyScheduler
	}

	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() error {
	s.wg.Wait()
	s.cron.Stop()

	return nil
}

type cronJob struct {
	Job Job
	wg  *sync.WaitGroup
}

func (c *cronJob) Run() {
	c.wg.Add(1)
	defer c.wg.Done()

	c.Job.Run()
}
