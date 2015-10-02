package middlewares

import (
	"testing"

	"github.com/mcuadros/ofelia/core"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteOverlap struct{}

var _ = Suite(&SuiteOverlap{})

func (s *SuiteOverlap) TestRun(c *C) {
	ctx := core.NewContext(nil, &TestJob{}, nil)

	m := &Overlap{}
	c.Assert(m.Run(ctx), IsNil)
}

func (s *SuiteOverlap) TestRunOverlap(c *C) {
	job := &TestJob{}
	ctx := core.NewContext(nil, job, nil)

	job.NotifyStart()
	m := &Overlap{}
	c.Assert(m.Run(ctx), Equals, ErrSkippedExecution)
}

func (s *SuiteOverlap) TestRunAllowOverlap(c *C) {
	job := &TestJob{}
	ctx := core.NewContext(nil, job, nil)

	job.NotifyStart()
	m := &Overlap{AllowOverlap: true}
	c.Assert(m.Run(ctx), IsNil)
}

type TestJob struct {
	core.BareJob
}

func (j *TestJob) Run(ctx *core.Context) error {
	return nil
}
