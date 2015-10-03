package middlewares

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteCommon struct{}

var _ = Suite(&SuiteCommon{})

func (s *SuiteCommon) TestIsEmpty(c *C) {
	config := &TestConfig{}
	c.Assert(IsEmpty(config), Equals, true)

	config = &TestConfig{Foo: "foo"}
	c.Assert(IsEmpty(config), Equals, false)

	config = &TestConfig{Qux: 42}
	c.Assert(IsEmpty(config), Equals, false)
}

type TestConfig struct {
	Foo string
	Qux int
	Bar bool
}
