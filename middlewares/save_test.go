package middlewares

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	. "gopkg.in/check.v1"
)

type SuiteSave struct {
	BaseSuite
}

var _ = Suite(&SuiteSave{})

func (s *SuiteSave) TestNewSlackEmpty(c *C) {
	c.Assert(NewSave(&SaveConfig{}), IsNil)
}

func (s *SuiteSave) TestRunSuccess(c *C) {
	dir, err := ioutil.TempDir("/tmp", "save")
	c.Assert(err, IsNil)

	s.ctx.Start()
	s.ctx.Stop(nil)

	s.job.Name = "foo"
	s.ctx.Execution.Date = time.Time{}

	m := NewSave(&SaveConfig{SaveFolder: dir})
	c.Assert(m.Run(s.ctx), IsNil)

	_, err = os.Stat(filepath.Join(dir, "00010101_000000_foo.json"))
	c.Assert(err, IsNil)

	_, err = os.Stat(filepath.Join(dir, "00010101_000000_foo.stdout.log"))
	c.Assert(err, IsNil)

	_, err = os.Stat(filepath.Join(dir, "00010101_000000_foo.stderr.log"))
	c.Assert(err, IsNil)
}

func (s *SuiteSave) TestRunSuccessOnError(c *C) {
	dir, err := ioutil.TempDir("/tmp", "save")
	c.Assert(err, IsNil)

	s.ctx.Start()
	s.ctx.Stop(nil)

	s.job.Name = "foo"
	s.ctx.Execution.Date = time.Time{}

	m := NewSave(&SaveConfig{SaveFolder: dir, SaveOnlyOnError: true})
	c.Assert(m.Run(s.ctx), IsNil)

	_, err = os.Stat(filepath.Join(dir, "00010101_000000_foo.json"))
	c.Assert(err, Not(IsNil))
}
