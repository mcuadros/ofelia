package cli

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteConfig struct{}

var _ = Suite(&SuiteConfig{})

func (s *SuiteConfig) TestBuildFromString(c *C) {
	sh, err := BuildFromString(`
		[job "foo"]
		schedule = @every 10s
		container = test

		[job "bar"]
		schedule = @every 10s
		container = test
  `)

	c.Assert(err, IsNil)
	c.Assert(sh.Jobs, HasLen, 2)
	c.Assert(sh.Jobs[0].GetName(), Equals, "foo")
	c.Assert(sh.Jobs[1].GetName(), Equals, "bar")
}

func (s *SuiteConfig) TestExecJobBuildEmpty(c *C) {
	j := &ExecJobConfig{}
	j.buildMiddlewares()

	c.Assert(j.Middlewares(), HasLen, 0)
}

func (s *SuiteConfig) TestExecJobBuild(c *C) {
	j := &ExecJobConfig{}
	j.OverlapConfig.NoOverlap = true
	j.buildMiddlewares()

	c.Assert(j.Middlewares(), HasLen, 1)
}
