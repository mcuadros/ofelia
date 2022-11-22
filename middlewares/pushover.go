package middlewares

import (
	"fmt"

	"github.com/azmodan2k/ofelia/core"
	"github.com/gregdel/pushover"
)

// PushoverConfig configuration for the Pushover middleware
type PushoverConfig struct {
	PushoverUserKey     string `gcfg:"pushover-userkey" mapstructure:"pushover-userkey"`
	PushoverAppKey      string `gcfg:"pushover-appkey" mapstructure:"pushover-appkey"`
	PushoverOnlyOnError bool   `gcfg:"pushover-only-on-error" mapstructure:"pushover-only-on-error"`
}

// NewPushover returns a Pushover middleware if the given configuration is not empty
func NewPushover(c *PushoverConfig) core.Middleware {
	var m core.Middleware
	if !IsEmpty(c) {
		m = &Pushover{*c}
	}

	return m
}

// Pushover middleware calls to a Pushover input-hook after every execution of a job
type Pushover struct {
	PushoverConfig
}

// ContinueOnStop return allways true, we want alloways report the final status
func (m *Pushover) ContinueOnStop() bool {
	return true
}

// Run sends a message to the pushover channel, its close stop the exection to
// collect the metrics
func (m *Pushover) Run(ctx *core.Context) error {
	err := ctx.Next()
	ctx.Stop(err)

	if ctx.Execution.Failed || !m.PushoverOnlyOnError {
		m.pushMessage(ctx)
	}

	return err
}

func (m *Pushover) pushMessage(ctx *core.Context) {
	app := pushover.New(m.PushoverAppKey)
	recipient := pushover.NewRecipient(m.PushoverUserKey)

	message := &pushover.Message{
		Message: fmt.Sprintf(
			"Job *%q* finished in *%s*, command `%s`",
			ctx.Job.GetName(), ctx.Execution.Duration, ctx.Job.GetCommand(),
		),
	}

	if ctx.Execution.Failed {
		message.Title = fmt.Sprintf("Execution *%q* failed", ctx.Job.GetName())
		message.Priority = pushover.PriorityEmergency
	} else if ctx.Execution.Skipped {
		message.Title = fmt.Sprintf("Execution *%q* skipped", ctx.Job.GetName())
		message.Priority = pushover.PriorityNormal
	} else {
		message.Title = fmt.Sprintf("Execution *%q* successful", ctx.Job.GetName())
		message.Priority = pushover.PriorityNormal
	}

	_, err := app.SendMessage(message, recipient)
	if err != nil {
		ctx.Logger.Errorf("Pushover error: %q", err)
	}
}
