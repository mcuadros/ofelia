package middlewares

import . "gopkg.in/check.v1"

type SuiteOverlap struct {
	BaseSuite
}

var _ = Suite(&SuiteOverlap{})

func (s *SuiteOverlap) TestNewOverlapEmpty(c *C) {
	c.Assert(NewOverlap(&OverlapConfig{}), IsNil)
}

func (s *SuiteOverlap) TestRun(c *C) {
	m := &Overlap{}
	c.Assert(m.Run(s.ctx), IsNil)
}

func (s *SuiteOverlap) TestRunOverlap(c *C) {
	s.ctx.Job.NotifyStart()
	s.ctx.Job.NotifyStart()

	m := NewOverlap(&OverlapConfig{NoOverlap: true})
	c.Assert(m.Run(s.ctx), Equals, ErrSkippedExecution)
}

func (s *SuiteOverlap) TestRunAllowOverlap(c *C) {
	s.ctx.Job.NotifyStart()

	m := NewOverlap(&OverlapConfig{NoOverlap: true})
	c.Assert(m.Run(s.ctx), IsNil)
}
