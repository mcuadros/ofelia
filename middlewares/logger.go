package middlewares

import (
	"fmt"

	"github.com/mcuadros/ofelia/core"

	"github.com/op/go-logging"
)

type Logger struct {
	logger *logging.Logger
}

func NewLogger(logger *logging.Logger) *Logger {
	return &Logger{logger}
}

func (m *Logger) Run(ctx *core.Context) error {
	e := ctx.Execution
	j := ctx.Job

	m.logger.Debug(fmt.Sprintf(
		"%s - Job started %q - %s",
		j.GetName(), e.ID, j.GetCommand(),
	))

	err := ctx.Next()
	ctx.Stop(err)

	errText := "none"
	if ctx.Execution.Error != nil {
		errText = ctx.Execution.Error.Error()
	}

	msg := fmt.Sprintf(
		"%s - Job finished %q in %s, failed: %t, error: %s",
		j.GetName(), e.ID, e.Duration, e.Failed, errText,
	)

	if ctx.Execution.Error != nil {
		m.logger.Warning(msg)
	} else {
		m.logger.Notice(msg)
	}

	return err
}
