package core

import (
	"time"

	. "gopkg.in/check.v1"
)

type SuiteScheduler struct{}

var _ = Suite(&SuiteScheduler{})

func (s *SuiteScheduler) TestAddJob(c *C) {
	job := &TestJob{}
	job.SetSpec("@hourly")

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
	job.SetSpec("@every 1s")

	sc := NewScheduler()
	err := sc.AddJob(job)
	c.Assert(err, IsNil)

	sc.Start()
	time.Sleep(time.Second)
	sc.Stop()

	c.Assert(job.Executions, Equals, 1)
	c.Assert(job.LastExecution().IsZero(), Equals, false)

}

type TestJob struct {
	BasicJob
	Executions int
}

func (j *TestJob) Run() {
	j.MarkStart()
	defer j.MarkStop(nil)

	time.Sleep(time.Millisecond * 500)
	j.Executions++
}
