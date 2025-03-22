package middlewares

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/AnthonyHewins/gotfy"
	. "gopkg.in/check.v1"
)

type SuiteNtfy struct {
	BaseSuite
}

var _ = Suite(&SuiteNtfy{})

func (s *SuiteNtfy) TestNewNtfyEmpty(c *C) {
	c.Assert(NewNtfy(&NtfyConfig{}), IsNil)
}

func (s *SuiteNtfy) TestRunSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m gotfy.Message
		// read json from request body
		json.NewDecoder(r.Body).Decode(&m)
		c.Assert(m.Title, Equals, "Execution successful")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gotfy.PublishResp{
			ID:      "foo",
			Time:    1,
			Expires: 2147483647,
			Topic:   "bar",
			Event:   "message",
			Message: "triggered",
		})
	}))

	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(nil)

	m := NewNtfy(&NtfyConfig{
		NtfyBaseURL:     ts.URL,
		NtfyApiKey:      "foo",
		NtfyTopic:       "bar",
		NtfyOnlyOnError: false,
	})
	c.Assert(m.Run(s.ctx), IsNil)
}

func (s *SuiteNtfy) TestRunSuccessFailed(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m gotfy.Message

		// read json from request body
		json.NewDecoder(r.Body).Decode(&m)
		c.Assert(m.Title, Equals, "Execution failed")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gotfy.PublishResp{
			ID:      "foo",
			Time:    1,
			Expires: 2147483647,
			Topic:   "bar",
			Event:   "message",
			Message: "triggered",
		})
	}))

	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(errors.New("foo"))

	m := NewNtfy(&NtfyConfig{
		NtfyBaseURL:     ts.URL,
		NtfyApiKey:      "foo",
		NtfyTopic:       "bar",
		NtfyOnlyOnError: false,
	})
	c.Assert(m.Run(s.ctx), IsNil)
}

func (s *SuiteNtfy) TestRunSuccessOnError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(true, Equals, false)
	}))

	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(nil)

	m := NewNtfy(&NtfyConfig{
		NtfyBaseURL:     ts.URL,
		NtfyApiKey:      "foo",
		NtfyTopic:       "bar",
		NtfyOnlyOnError: true,
	})
	c.Assert(m.Run(s.ctx), IsNil)
}
