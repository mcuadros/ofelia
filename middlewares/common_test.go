package middlewares

import (
	"testing"

	"github.com/mcuadros/ofelia/core"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteCommon struct {
	BaseSuite
}

var _ = Suite(&SuiteCommon{})

func (s *SuiteCommon) TestIsEmpty(c *C) {
	config := &TestConfig{}
	c.Assert(IsEmpty(config), Equals, true)

	config = &TestConfig{Foo: "foo"}
	c.Assert(IsEmpty(config), Equals, false)

	config = &TestConfig{Qux: 42}
	c.Assert(IsEmpty(config), Equals, false)
}

type BaseSuite struct {
	ctx *core.Context
}

func (s *BaseSuite) SetUpTest(c *C) {
	job := &TestJob{}
	sh := core.NewScheduler(&TestLogger{})
	e := core.NewExecution()

	s.ctx = core.NewContext(sh, job, e)
}

type TestConfig struct {
	Foo string
	Qux int
	Bar bool
}

type TestJob struct {
	core.BareJob
}

func (j *TestJob) Run(ctx *core.Context) error {
	return nil
}

type TestLogger struct{}

func (*TestLogger) Critical(format string, args ...interface{}) {}
func (*TestLogger) Debug(format string, args ...interface{})    {}
func (*TestLogger) Error(format string, args ...interface{})    {}
func (*TestLogger) Notice(format string, args ...interface{})   {}
func (*TestLogger) Warning(format string, args ...interface{})  {}
