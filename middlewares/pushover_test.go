package middlewares

import (
	"time"
	. "gopkg.in/check.v1"
)

type SuitePushover struct {
	BaseSuite

	pushoverUserKey     string
	pushoverAppKey      string
}

var _ = Suite(&SuitePushover{})

func (s *SuitePushover) SetUpTest(c *C) {
	s.BaseSuite.SetUpTest(c)

	s.pushoverUserKey = "gznej3rKEVAvPUxu9vvNnqpmZpokzF"
	s.pushoverAppKey = "uQiRzpo4DXghDmr9QzzfQu27cmVRsG"
}

func (s *SuitePushover) TestNewPushoverEmpty(c *C) {
	c.Assert(NewPushover(&PushoverConfig{}), IsNil)
}

func (s *SuitePushover) TestRunSuccess(c *C) {
	s.ctx.Start()
	s.ctx.Stop(nil)

	s.job.Name = "foo"
	s.ctx.Execution.Date = time.Time{}
	s.ctx.Execution.Failed = false

	m := NewPushover(&PushoverConfig{ 
			PushoverUserKey: s.pushoverUserKey, 
			PushoverAppKey: s.pushoverAppKey, 
			PushoverOnlyOnError: false,
		})

	c.Assert(m.Run(s.ctx), IsNil)
}

func (s *SuitePushover) TestRunFailure(c *C) {
	s.ctx.Start()
	s.ctx.Stop(nil)

	s.job.Name = "foo"
	s.ctx.Execution.Date = time.Time{}
	s.ctx.Execution.Failed = true

	m := NewPushover(&PushoverConfig{ 
		PushoverUserKey: s.pushoverUserKey, 
		PushoverAppKey: s.pushoverAppKey, 
		PushoverOnlyOnError: false,
	})
	
	c.Assert(m.Run(s.ctx), IsNil)
}