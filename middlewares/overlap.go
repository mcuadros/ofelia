package middlewares

import "github.com/mcuadros/ofelia/core"

// OverlapConfig configuration for the Overlap middleware
type OverlapConfig struct {
	NoOverlap bool `gcfg:"no-overlap" mapstructure:"no-overlap"`
}

// NewOverlap returns a Overlap middleware if the given configuration is not empty
func NewOverlap(c *OverlapConfig) core.Middleware {
	var m core.Middleware
	if !IsEmpty(c) {
		m = &Overlap{*c}
	}

	return m
}

// Overlap when this middleware is enabled avoid to overlap executions from a
// specific job
type Overlap struct {
	OverlapConfig
}

// ContinueOnStop Overlap is only called if the process is still running
func (m *Overlap) ContinueOnStop() bool {
	return false
}

// Run stops the execution if the another execution is already running
func (m *Overlap) Run(ctx *core.Context) error {
	if m.NoOverlap && ctx.Job.Running() > 1 {
		ctx.Stop(core.ErrSkippedExecution)
	}

	return ctx.Next()
}
