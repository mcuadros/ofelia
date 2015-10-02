package middlewares

import (
	"fmt"

	"github.com/mcuadros/ofelia/core"
	"github.com/prometheus/log"

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

	log.Info(fmt.Sprintf(
		"%s - Job started %q - %s",
		j.GetName(), e.ID, j.GetCommand(),
	))

	err := ctx.Next()
	ctx.Stop(err)

	errText := "none"
	if ctx.Execution.Error != nil {
		errText = ctx.Execution.Error.Error()
	}

	log.Info(fmt.Sprintf(
		"%s - Job finished %q in %s, failed: %t, error: %s",
		j.GetName(), e.ID, e.Duration, e.Failed, errText,
	))

	return err
}
