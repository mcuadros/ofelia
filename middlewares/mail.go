package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"

	"crypto/tls"
	"gopkg.in/gomail.v2"

	"github.com/mcuadros/ofelia/core"
)

// MailConfig configuration for the Mail middleware
type MailConfig struct {
	SMTPHost          string `gcfg:"smtp-host" mapstructure:"smtp-host"`
	SMTPPort          int    `gcfg:"smtp-port" mapstructure:"smtp-port"`
	SMTPUser          string `gcfg:"smtp-user" mapstructure:"smtp-user"`
	SMTPPassword      string `gcfg:"smtp-password" mapstructure:"smtp-password"`
	SMTPTLSSkipVerify bool   `gcfg:"smtp-tls-skip-verify" mapstructure:"smtp-tls-skip-verify"`
	EmailTo           string `gcfg:"email-to" mapstructure:"email-to"`
	EmailFrom         string `gcfg:"email-from" mapstructure:"email-from"`
	MailOnlyOnError   bool   `gcfg:"mail-only-on-error" mapstructure:"mail-only-on-error"`
}

// NewMail returns a Mail middleware if the given configuration is not empty
func NewMail(c *MailConfig) core.Middleware {
	var m core.Middleware

	if !IsEmpty(c) {
		m = &Mail{*c}
	}

	return m
}

// Mail middleware delivers a email just after an execution finishes
type Mail struct {
	MailConfig
}

// ContinueOnStop return allways true, we want always report the final status
func (m *Mail) ContinueOnStop() bool {
	return true
}

// Run sents a email with the result of the execution
func (m *Mail) Run(ctx *core.Context) error {
	err := ctx.Next()
	ctx.Stop(err)

	if ctx.Execution.Failed || !m.MailOnlyOnError {
		err := m.sendMail(ctx)
		if err != nil {
			ctx.Logger.Errorf("Mail error: %q", err)
		}
	}

	return err
}

func (m *Mail) sendMail(ctx *core.Context) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from())
	msg.SetHeader("To", strings.Split(m.EmailTo, ",")...)
	msg.SetHeader("Subject", m.subject(ctx))
	msg.SetBody("text/html", m.body(ctx))

	base := fmt.Sprintf("%s_%s", ctx.Job.GetName(), ctx.Execution.ID)
	msg.Attach(base+".stdout.log", gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := w.Write(ctx.Execution.OutputStream.Bytes())
		return err
	}))

	msg.Attach(base+".stderr.log", gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := w.Write(ctx.Execution.ErrorStream.Bytes())
		return err
	}))

	msg.Attach(base+".stderr.json", gomail.SetCopyFunc(func(w io.Writer) error {
		js, _ := json.MarshalIndent(map[string]interface{}{
			"Job":       ctx.Job,
			"Execution": ctx.Execution,
		}, "", "  ")

		_, err := w.Write(js)
		return err
	}))

	d := gomail.NewPlainDialer(m.SMTPHost, m.SMTPPort, m.SMTPUser, m.SMTPPassword)
	// When TLSConfig.InsecureSkipVerify is true, mail server certificate authority is not validated
	if m.SMTPTLSSkipVerify {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if err := d.DialAndSend(msg); err != nil {
		return err
	}

	return nil
}

func (m *Mail) from() string {
	if strings.Index(m.EmailFrom, "%") == -1 {
		return m.EmailFrom
	}

	hostname, _ := os.Hostname()
	return fmt.Sprintf(m.EmailFrom, hostname)
}

func (m *Mail) subject(ctx *core.Context) string {
	buf := bytes.NewBuffer(nil)
	mailSubjectTemplate.Execute(buf, ctx)

	return buf.String()
}

func (m *Mail) body(ctx *core.Context) string {
	buf := bytes.NewBuffer(nil)
	mailBodyTemplate.Execute(buf, ctx)

	return buf.String()
}

var mailBodyTemplate, mailSubjectTemplate *template.Template

func init() {
	f := map[string]interface{}{
		"status": executionLabel,
	}

	mailBodyTemplate = template.New("mail-body")
	mailSubjectTemplate = template.New("mail-subject")
	mailBodyTemplate.Funcs(f)
	mailSubjectTemplate.Funcs(f)

	template.Must(mailBodyTemplate.Parse(`
		<p>
			Job ​<b>{{.Job.GetName}}</b>,
			Execution <b>{{status .Execution}}</b> in ​<b>{{.Execution.Duration}}</b>​,
			command: ​<pre>{{.Job.GetCommand}}</pre>​
		</p>
  `))

	template.Must(mailSubjectTemplate.Parse(
		"[Execution {{status .Execution}}] Job {{.Job.GetName}} finished in {{.Execution.Duration}}",
	))
}

func executionLabel(e *core.Execution) string {
	status := "successful"
	if e.Skipped {
		status = "skipped"
	} else if e.Failed {
		status = "failed"
	}

	return status
}
