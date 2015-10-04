package middlewares

import (
	"errors"

	"github.com/mcuadros/ofelia/core"
)

var ErrSkippedExecution = errors.New("skipped execution")

type OverlapConfig struct {
	NoOverlap bool `gcfg:"no-overlap"`
}

func NewOverlap(c *OverlapConfig) core.Middleware {
	var m core.Middleware
	if !IsEmpty(c) {
		m = &Overlap{*c}
	}

	return m
}

type Overlap struct {
	OverlapConfig
}

func (m *Overlap) Run(ctx *core.Context) error {
	if m.NoOverlap && ctx.Job.Running() > 1 {
		return ErrSkippedExecution
	}

	return ctx.Next()
}
