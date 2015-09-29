package core

import (
	"errors"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteCommon struct{}

var _ = Suite(&SuiteCommon{})

func (s *SuiteCommon) TestBasicJobGetters(c *C) {
	job := &BasicJob{
		name: "foo",
		spec: "bar",
	}

	c.Assert(job.Name(), Equals, "foo")
	c.Assert(job.Spec(), Equals, "bar")
	c.Assert(job.Running(), Equals, int32(0))
}

func (s *SuiteCommon) TestBasicJobStart(c *C) {
	job := &BasicJob{}
	job.Start()
	job.Start()

	c.Assert(job.Running(), Equals, int32(1))

	h := job.History()
	c.Assert(h, HasLen, 2)
	c.Assert(h[0].IsRunning, Equals, true)
	c.Assert(h[0].Skipped, Equals, false)
	c.Assert(h[1].IsRunning, Equals, false)
	c.Assert(h[1].Skipped, Equals, true)

}

func (s *SuiteCommon) TestBasicJobStartOverlap(c *C) {
	job := &BasicJob{}
	job.AllowOverlap = true
	job.Start()
	job.Start()

	c.Assert(job.Running(), Equals, int32(2))

	h := job.History()
	c.Assert(h, HasLen, 2)
	c.Assert(h[0].IsRunning, Equals, true)
	c.Assert(h[0].Skipped, Equals, false)
	c.Assert(h[1].IsRunning, Equals, true)
	c.Assert(h[1].Skipped, Equals, false)
}

func (s *SuiteCommon) TestBasicJobStop(c *C) {
	job := &BasicJob{}
	job.Stop(job.Start(), nil)

	c.Assert(job.Running(), Equals, int32(0))

	h := job.History()
	c.Assert(h, HasLen, 1)
	c.Assert(h[0].IsRunning, Equals, false)
}

func (s *SuiteCommon) TestExecutionStart(c *C) {
	exe := &Execution{}
	exe.Start()

	c.Assert(exe.IsRunning, Equals, true)
	c.Assert(exe.Date.IsZero(), Equals, false)
}

func (s *SuiteCommon) TestExecutionStop(c *C) {
	exe := &Execution{}
	exe.Start()
	exe.Stop(nil)

	c.Assert(exe.IsRunning, Equals, false)
	c.Assert(exe.Failed, Equals, false)
	c.Assert(exe.Skipped, Equals, false)
	c.Assert(exe.Error, Equals, nil)
	c.Assert(exe.Duration.Seconds() > .0, Equals, true)
}

func (s *SuiteCommon) TestExecutionStopSkipped(c *C) {
	exe := &Execution{}
	exe.Start()
	exe.Stop(ErrSkippedExecution)

	c.Assert(exe.IsRunning, Equals, false)
	c.Assert(exe.Failed, Equals, false)
	c.Assert(exe.Skipped, Equals, true)
	c.Assert(exe.Error, Equals, nil)
	c.Assert(exe.Duration.Seconds(), Equals, .0)
}

func (s *SuiteCommon) TestExecutionStopError(c *C) {
	err := errors.New("foo")

	exe := &Execution{}
	exe.Start()
	exe.Stop(err)

	c.Assert(exe.IsRunning, Equals, false)
	c.Assert(exe.Failed, Equals, true)
	c.Assert(exe.Skipped, Equals, false)
	c.Assert(exe.Error, Equals, err)
	c.Assert(exe.Duration.Seconds() > .0, Equals, true)
}
