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
		[job-exec "foo"]
		schedule = @every 10s
		container = test

		[job-exec "bar"]
		schedule = @every 10s
		container = test

		[job-run "qux"]
		schedule = @every 10s
		image = test
  `)

	c.Assert(err, IsNil)
	c.Assert(sh.Jobs, HasLen, 3)
	c.Assert(sh.Jobs[0].GetName(), Equals, "foo")
	c.Assert(sh.Jobs[1].GetName(), Equals, "bar")
	c.Assert(sh.Jobs[2].GetName(), Equals, "qux")
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
