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

	sc := NewScheduler(&TestLogger{})
	err := sc.AddJob(job)
	c.Assert(err, IsNil)

	e := sc.cron.Entries()
	c.Assert(e, HasLen, 1)
	c.Assert(e[0].Job.(*jobWrapper).j, DeepEquals, job)
}

func (s *SuiteScheduler) TestStartStop(c *C) {
	job := &TestJob{}
	job.Schedule = "@every 1s"

	sc := NewScheduler(&TestLogger{})
	err := sc.AddJob(job)
	c.Assert(err, IsNil)

	sc.Start()
	c.Assert(sc.IsRunning(), Equals, true)

	time.Sleep(time.Second * 2)

	sc.Stop()
	c.Assert(sc.IsRunning(), Equals, false)
}

func (s *SuiteScheduler) TestMergeMiddlewaresSame(c *C) {
	mA, mB, mC := &TestMiddleware{}, &TestMiddleware{}, &TestMiddleware{}

	job := &TestJob{}
	job.Schedule = "@every 1s"
	job.Use(mB, mC)

	sc := NewScheduler(&TestLogger{})
	sc.Use(mA)
	sc.AddJob(job)

	m := job.Middlewares()
	c.Assert(m, HasLen, 1)
	c.Assert(m[0], Equals, mB)
}
