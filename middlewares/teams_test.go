package middlewares

import (
	"encoding/json"
	"errors"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

type SuiteTeams struct {
	BaseSuite
}

var _ = Suite(&SuiteTeams{})

func (s *SuiteTeams) TestNewTeamsEmpty(c *C) {
	c.Assert(NewTeams(&TeamsConfig{}), IsNil)
}

func (s *SuiteTeams) TestRunTeamsSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m teamsMessage
		b, _ := ioutil.ReadAll(r.Body)
		_ = json.Unmarshal(b, &m)
		c.Assert(m.Summary, Equals, "Execution successful")
	}))

	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(nil)

	m := NewTeams(&TeamsConfig{TeamsWebhook: ts.URL})
	c.Assert(m.Run(s.ctx), IsNil)
}

func (s *SuiteTeams) TestRunTeamsSuccessFailed(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m teamsMessage
		b, _ := ioutil.ReadAll(r.Body)
		_ = json.Unmarshal(b, &m)
		c.Assert(m.Summary, Equals, "Execution failed")
	}))

	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(errors.New("foo"))

	m := NewTeams(&TeamsConfig{TeamsWebhook: ts.URL})
	c.Assert(m.Run(s.ctx), IsNil)
}

func (s *SuiteTeams) TestRunTeamsSuccessOnError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(true, Equals, false)
	}))

	defer ts.Close()

	s.ctx.Start()
	s.ctx.Stop(nil)

	m := NewTeams(&TeamsConfig{TeamsWebhook: ts.URL, TeamsOnlyOnError: true})
	c.Assert(m.Run(s.ctx), IsNil)
}
