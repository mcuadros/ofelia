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

		[job-exec "bar"]
		schedule = @every 10s

		[job-run "qux"]
		schedule = @every 10s

		[job-local "baz"]
		schedule = @every 10s
  `)

	c.Assert(err, IsNil)
	c.Assert(sh.Jobs, HasLen, 4)
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
