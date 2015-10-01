package core

import (
	"errors"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteCommon struct{}

var _ = Suite(&SuiteCommon{})

func (s *SuiteCommon) TestNewContext(c *C) {
	h := &Scheduler{}
	j := &TestJob{}
	j.Use(&TestMiddleware{}, &TestMiddleware{})

	e := NewExecution()

	ctx := NewContext(h, j, e)
	c.Assert(ctx.Scheduler, DeepEquals, h)
	c.Assert(ctx.Job, DeepEquals, j)
	c.Assert(ctx.Execution, DeepEquals, e)
	c.Assert(ctx.middlewares, HasLen, 2)
}

func (s *SuiteCommon) TestContextNext(c *C) {
	mA := &TestMiddleware{}
	mB := &TestMiddleware{}
	mC := &TestMiddleware{}

	j := &TestJob{}
	j.Use(mA, mB, mC)

	e := NewExecution()

	ctx := NewContext(nil, j, e)

	err := ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mA.Called, Equals, true)
	c.Assert(mB.Called, Equals, false)
	c.Assert(mC.Called, Equals, false)
	c.Assert(j.Called, Equals, false)

	err = ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mB.Called, Equals, true)
	c.Assert(mC.Called, Equals, false)
	c.Assert(j.Called, Equals, false)

	err = ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mC.Called, Equals, true)
	c.Assert(j.Called, Equals, false)

	err = ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(j.Called, Equals, true)
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

type TestMiddleware struct {
	Called bool
}

func (m *TestMiddleware) Run(*Context) error {
	m.Called = true

	return nil
}

type TestJob struct {
	BareJob
	Called bool
}

func (j *TestJob) Run(ctx *Context) error {
	j.Called = true
	time.Sleep(time.Millisecond * 500)

	return nil
}
