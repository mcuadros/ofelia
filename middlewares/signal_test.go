package middlewares

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	. "gopkg.in/check.v1"
)

type SuiteSignal struct {
	BaseSuite
}

var _ = Suite(&SuiteSignal{})

func (s *SuiteSignal) TestNewSignalEmpty(c *C) {
	c.Assert(NewSignal(&SignalConfig{}), IsNil)
}

func (s *SuiteSignal) TestRunSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		c.Assert(err, IsNil)

		var m signalMessage
		err = json.Unmarshal(body, &m)
		c.Assert(err, IsNil)
		c.Assert(strings.Contains(m.Message, "[SUCCESS]"), Equals, true)
	}))
	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(nil)

	m := NewSignal(&SignalConfig{SignalURL: ts.URL})
	c.Assert(m.Run(s.ctx), IsNil)
}

func (s *SuiteSignal) TestRunSuccessFailed(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		c.Assert(err, IsNil)

		var m signalMessage
		err = json.Unmarshal(body, &m)
		c.Assert(err, IsNil)
		c.Assert(strings.Contains(m.Message, "[FAILED]"), Equals, true)
	}))
	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(errors.New("foo"))

	m := NewSignal(&SignalConfig{SignalURL: ts.URL})
	c.Assert(m.Run(s.ctx), IsNil)
}

func (s *SuiteSignal) TestRunSuccessOnError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(true, Equals, false)
	}))

	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(nil)

	m := NewSignal(&SignalConfig{SignalURL: ts.URL, SignalOnlyOnError: true})
	c.Assert(m.Run(s.ctx), IsNil)
}

func (s *SuiteSignal) TestConfig(c *C) {
	number := "223344"
	recipients := []string{"reciepient1", "r2"}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		c.Assert(err, IsNil)

		var m signalMessage
		err = json.Unmarshal(body, &m)
		c.Assert(err, IsNil)
		c.Assert(m.Number, Equals, number)
		c.Assert(m.Recipients, DeepEquals, recipients)
	}))

	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(nil)

	m := NewSignal(&SignalConfig{SignalURL: ts.URL, SignalNumber: number, SignalRecipients: recipients})
	c.Assert(m.Run(s.ctx), IsNil)
}
