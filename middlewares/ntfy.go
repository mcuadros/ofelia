package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/AnthonyHewins/gotfy"
	"github.com/mcuadros/ofelia/core"
)

type NtfyConfig struct {
	NtfyBaseURL     string `gcfg:"ntfy-base-url" mapstructure:"ntfy-base-url"`
	NtfyTopic       string `gcfg:"ntfy-topic" mapstructure:"ntfy-topic"`
	NtfyApiKey      string `gcfg:"ntfy-api-key" mapstructure:"ntfy-api-key" json:"-"`
	NtfyOnlyOnError bool   `gcfg:"ntfy-only-on-error" mapstructure:"ntfy-only-on-error"`
}

func NewNtfy(c *NtfyConfig) core.Middleware {
	var m core.Middleware

	if !IsEmpty(c) {
		m = &Ntfy{*c}
	}

	return m
}

type Ntfy struct {
	NtfyConfig
}

func (n *Ntfy) ContinueOnStop() bool {
	return true
}

func (n *Ntfy) Run(ctx *core.Context) error {
	err := ctx.Next()
	ctx.Stop(err)

	if ctx.Execution.Failed || !n.NtfyOnlyOnError {
		err := n.sendNtfy(ctx)
		if err != nil {
			ctx.Logger.Errorf("ntfy error: %q", err)
		}
	}

	return err
}

func (n *Ntfy) sendNtfy(ctx *core.Context) error {
	server, _ := url.Parse(n.NtfyBaseURL)
	httpClient := http.DefaultClient

	if n.NtfyApiKey == "" {
		return errors.New("missing credentials for ntfy service")
	}

	if n.NtfyTopic == "" {
		return errors.New("missing topic for ntfy service")
	}

	tp, err := gotfy.NewPublisher(server, httpClient)
	if err != nil {
		return err
	}
	tp.Headers.Add("Authorization", "Bearer "+n.NtfyApiKey)
	tp.Headers.Add("X-Markdown", "true")

	msg := &gotfy.Message{
		Topic: n.NtfyTopic,
	}
	msg.Message = fmt.Sprintf(
		"Job *%q* finished in *%s*, command `%s`",
		ctx.Job.GetName(), ctx.Execution.Duration, ctx.Job.GetCommand(),
	)

	if ctx.Execution.Failed {
		msg.Title = "Execution failed"
		msg.Message = ctx.Execution.Error.Error()
	} else if ctx.Execution.Skipped {
		msg.Title = "Execution skipped"
	} else {
		msg.Title = "Execution successful"
	}

	_, err = tp.SendMessage(context.Background(), msg)

	return err
}
