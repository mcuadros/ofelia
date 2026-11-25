package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mcuadros/ofelia/core"
)

var (
	teamsAvatarURL = "https://raw.githubusercontent.com/mcuadros/ofelia/master/static/avatar.png"
)

// TeamsConfig configuration for the Teams middleware
type TeamsConfig struct {
	TeamsWebhook     string `gcfg:"teams-webhook" mapstructure:"teams-webhook"`
	TeamsOnlyOnError bool   `gcfg:"teams-only-on-error" mapstructure:"teams-only-on-error"`
}

func NewTeams(c *TeamsConfig) core.Middleware {
	var m core.Middleware
	if !IsEmpty(c) {
		m = &Teams{*c}
	}

	return m
}

// Teams middleware calls to a Teams input-hook after every execution of a job
type Teams struct {
	TeamsConfig
}

// ContinueOnStop returns always true
func (m *Teams) ContinueOnStop() bool {
	return true
}

// Run sends a message to the Teams channel, its close stop the execution to
// collect the metrics
func (m *Teams) Run(ctx *core.Context) error {
	err := ctx.Next()
	ctx.Stop(err)

	if ctx.Execution.Failed || !m.TeamsOnlyOnError {
		m.pushMessage(ctx)
	}

	return err
}

func (m *Teams) pushMessage(ctx *core.Context) {
	content, _ := json.Marshal(m.buildMessage(ctx))
	reader := bytes.NewReader(content)

	r, err := http.Post(m.TeamsWebhook, "application/json", reader)
	if err != nil {
		ctx.Logger.Errorf("Teams error calling %q error: %q", m.TeamsWebhook, err)
	} else if r.StatusCode != 200 {
		body, _ := ioutil.ReadAll(r.Body)
		ctx.Logger.Errorf("Teams error non-200 status code calling %q: %v", m.TeamsWebhook, string(body))
	}
}

func (m *Teams) buildMessage(ctx *core.Context) *teamsMessage {
	msg := newTeamsMessage()

	title := fmt.Sprintf(
		"Job *%q* finished in *%s*, command `%s`",
		ctx.Job.GetName(), ctx.Execution.Duration, ctx.Job.GetCommand(),
	)

	s1 := teamsMessageSections{
		ActivityTitle:    title,
		ActivitySubtitle: "Execution successful",
		ActivityImage:    teamsAvatarURL,
		Facts:            make([]teamsMessageSectionFact, 0),
		Markdown:         true,
	}

	if ctx.Execution.Failed {
		msg.ThemeColor = "F35A00"
		msg.Summary = "Execution failed"
		s1.ActivitySubtitle = fmt.Sprintf("Execution failed: %v", ctx.Execution.Error.Error())
	} else if ctx.Execution.Skipped {
		msg.ThemeColor = "FFA500"
		msg.Summary = "Execution skipped"
		s1.ActivitySubtitle = fmt.Sprintf("Execution skipped")
	}

	msg.Sections = append(msg.Sections, s1)

	if isSuccess(ctx.Execution) {
		s2 := teamsMessageSections{
			ActivityTitle: "Execution results",
			ActivityText:  strings.ReplaceAll(ctx.Execution.OutputStream.String(), "\n", "<br>"),
			ActivityImage: "",
			Facts:         nil,
			Markdown:      true,
		}
		msg.Sections = append(msg.Sections, s2)
	}

	return msg
}

func isSuccess(e *core.Execution) bool {
	if e.Failed || e.Skipped {
		return false
	}
	return true
}

func newTeamsMessage() *teamsMessage {
	return &teamsMessage{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: "0076D7",
		Summary:    "Execution successful",
		Sections:   make([]teamsMessageSections, 0),
	}
}

type teamsMessage struct {
	Type       string                 `json:"@type"`
	Context    string                 `json:"@context"`
	ThemeColor string                 `json:"themeColor"`
	Summary    string                 `json:"summary"`
	Sections   []teamsMessageSections `json:"sections"`
}

type teamsMessageSections struct {
	ActivityTitle    string                    `json:"activityTitle"`
	ActivitySubtitle string                    `json:"activitySubtitle"`
	ActivityImage    string                    `json:"activityImage"`
	ActivityText     string                    `json:"activityText"`
	Facts            []teamsMessageSectionFact `json:"facts"`
	Markdown         bool                      `json:"markdown"`
}

type teamsMessageSectionFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
