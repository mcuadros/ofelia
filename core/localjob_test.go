package core

import (
	"strings"

	"github.com/armon/circbuf"

	. "gopkg.in/check.v1"
)

type SuiteLocalJob struct{}

var _ = Suite(&SuiteLocalJob{})

func (s *SuiteLocalJob) TestRun(c *C) {
	job := &LocalJob{}
	job.Command = `echo "foo bar"`

	b, _ := circbuf.NewBuffer(1000)
	e := NewExecution()
	e.OutputStream = b

	err := job.Run(&Context{Execution: e})
	c.Assert(err, IsNil)
	c.Assert(b.String(), Equals, "foo bar\n")
}

func (s *SuiteLocalJob) TestEnvironment(c *C) {
	job := &LocalJob{}
	job.Command = `env`
	env := []string{"test_Key1=value1", "test_Key2=value2"}
	job.Environment = env

	b, _ := circbuf.NewBuffer(1000)
	e := NewExecution()
	e.OutputStream = b

	err := job.Run(&Context{Execution: e})
	c.Assert(err, IsNil)

	// check that expected keys are present in the system env
	for _, expectedEnv := range env {
		found := false
		for _, systemEnv := range strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n") {
			if expectedEnv == systemEnv {
				found = true
				break
			}
		}
		c.Assert(found, Equals, true)
	}
}
