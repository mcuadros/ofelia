package core

import (
	"time"

	. "gopkg.in/check.v1"
)

type SuiteScheduler struct{}

var _ = Suite(&SuiteScheduler{})

func (s *SuiteScheduler) TestAddJob(c *C) {
	job := &TestJob{}
	job.Schedule = "@hourly"

	sc := NewScheduler()
	err := sc.AddJob(job)
	c.Assert(err, IsNil)
	c.Assert(sc.Jobs, HasLen, 1)

	e := sc.cron.Entries()
	c.Assert(e, HasLen, 1)
	c.Assert(e[0].Job.(*cronJob).Job, DeepEquals, job)
}

func (s *SuiteScheduler) TestStartStop(c *C) {
	job := &TestJob{}
	job.Schedule = "@every 1s"

	sc := NewScheduler()
	err := sc.AddJob(job)
	c.Assert(err, IsNil)

	sc.Start()
	c.Assert(sc.IsRunning(), Equals, true)

	time.Sleep(time.Second * 2)

	sc.Stop()
	c.Assert(sc.IsRunning(), Equals, false)

	h := job.History()
	c.Assert(h, HasLen, 2)
	c.Assert(h[0].IsRunning, Equals, false)
	c.Assert(h[0].Date.IsZero(), Equals, false)
	c.Assert(h[1].IsRunning, Equals, false)
	c.Assert(h[1].Date.IsZero(), Equals, false)
}

type TestJob struct {
	BareJob
}

func (j *TestJob) Run() {
	e := j.Start()
	defer j.Stop(e, nil)

	time.Sleep(time.Millisecond * 500)
}
