package middlewares

import (
	"testing"

	"github.com/netresearch/ofelia/core"

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
	job *TestJob
}

func (s *BaseSuite) SetUpTest(c *C) {
	s.job = &TestJob{}
	sh := core.NewScheduler(&TestLogger{})
	e := core.NewExecution()

	s.ctx = core.NewContext(sh, s.job, e)
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

func (*TestLogger) Criticalf(format string, args ...interface{}) {}
func (*TestLogger) Debugf(format string, args ...interface{})    {}
func (*TestLogger) Errorf(format string, args ...interface{})    {}
func (*TestLogger) Noticef(format string, args ...interface{})   {}
func (*TestLogger) Warningf(format string, args ...interface{})  {}
