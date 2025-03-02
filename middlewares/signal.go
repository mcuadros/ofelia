package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mcuadros/ofelia/core"
)

type SignalConfig struct {
	SignalURL         string   `gcfg:"signal-url" mapstructure:"signal-url" json:"-"`
	SignalNumber      string   `gcfg:"signal-number" mapstructure:"signal-number"`
	SignalRecipients  []string `gcfg:"signal-recipients" mapstructure:"signal-recipients"`
	SignalOnlyOnError bool     `gcfg:"signal-only-on-error" mapstructure:"signal-only-on-error"`
}

func NewSignal(c *SignalConfig) core.Middleware {
	var m core.Middleware
	if !IsEmpty(c) {
		m = &Signal{*c}
	}

	return m
}

type Signal struct {
	SignalConfig
}

func (m *Signal) ContinueOnStop() bool {
	return true
}

func (m *Signal) Run(ctx *core.Context) error {
	err := ctx.Next()
	ctx.Stop(err)

	if ctx.Execution.Failed || !m.SignalOnlyOnError {
		m.pushMessage(ctx)
	}

	return err
}

func (m *Signal) pushMessage(ctx *core.Context) {
	endpoint := m.SignalURL + "/v2/send"
	payload, _ := json.Marshal(m.buildMessage(ctx))
	ctx.Logger.Noticef("Sending Signal message. Sender: %s, Recipients: %v", m.SignalNumber, m.SignalRecipients)

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		ctx.Logger.Errorf("Error sending Signal message: %v", err)
		return
	}

	if resp.StatusCode != http.StatusCreated {
		ctx.Logger.Errorf("Failed to send Signal message. Non-201 status code: %d, URL: %q", resp.StatusCode, endpoint)
	}
}

func (m *Signal) buildMessage(ctx *core.Context) *signalMessage {
	msg := &signalMessage{
		Number:     m.SignalNumber,
		Recipients: m.SignalRecipients,
	}

	var errorDetails string
	if ctx.Execution.Failed && ctx.Execution.Error != nil {
		errorDetails = fmt.Sprintf(" Error: %s", ctx.Execution.Error.Error())
	}

	msg.Message = fmt.Sprintf(
		"[%s] Job *%q* finished in *%s*, command `%s`.%s",
		getExecutionStatus(ctx.Execution),
		ctx.Job.GetName(),
		ctx.Execution.Duration,
		ctx.Job.GetCommand(),
		errorDetails,
	)

	return msg
}

func getExecutionStatus(e *core.Execution) string {
	switch {
	case e.Failed:
		return "FAILED"
	case e.Skipped:
		return "SKIPPED"
	default:
		return "SUCCESS"
	}
}

type signalMessage struct {
	Message    string   `json:"message"`
	Number     string   `json:"number"`
	Recipients []string `json:"recipients"`
}
