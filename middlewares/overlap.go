package middlewares

import (
	"errors"

	"github.com/mcuadros/ofelia/core"
)

var ErrSkippedExecution = errors.New("skipped execution")

type Overlap struct {
	AllowOverlap bool `gcfg:"allow-overlap" default:"true"`
}

func (m *Overlap) Run(ctx *core.Context) error {
	if !m.AllowOverlap && ctx.Job.Running() != 0 {
		return ErrSkippedExecution
	}

	return ctx.Next()
}
