package middlewares

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/mcuadros/ofelia/core"
)

// SaveConfig configuration for the Save middleware
type SaveConfig struct {
	SaveFolder      string `gcfg:"save-folder" mapstructure:"save-folder"`
	SaveOnlyOnError bool   `gcfg:"save-only-on-error" mapstructure:"save-only-on-error"`
}

// NewSave returns a Save middleware if the given configuration is not empty
func NewSave(c *SaveConfig) core.Middleware {
	var m core.Middleware
	if !IsEmpty(c) {
		m = &Save{*c}
	}

	return m
}

// Save the save middleware saves to disk a dump of the stdout and stderr after
// every execution of the process
type Save struct {
	SaveConfig
}

// ContinueOnStop return allways true, we want always report the final status
func (m *Save) ContinueOnStop() bool {
	return true
}

// Run save the result of the execution to disk
func (m *Save) Run(ctx *core.Context) error {
	err := ctx.Next()
	ctx.Stop(err)

	if ctx.Execution.Failed || !m.SaveOnlyOnError {
		err := m.saveToDisk(ctx)
		if err != nil {
			ctx.Logger.Errorf("Save error: %q", err)
		}
	}

	return err
}

func (m *Save) saveToDisk(ctx *core.Context) error {
	root := filepath.Join(m.SaveFolder, fmt.Sprintf(
		"%s_%s",
		ctx.Execution.Date.Format("20060102_150405"), ctx.Job.GetName(),
	))

	e := ctx.Execution
	err := m.writeFile(e.ErrorStream.Bytes(), fmt.Sprintf("%s.stderr.log", root))
	if err != nil {
		return err
	}

	err = m.writeFile(e.OutputStream.Bytes(), fmt.Sprintf("%s.stdout.log", root))
	if err != nil {
		return err
	}

	err = m.saveContextToDisk(ctx, fmt.Sprintf("%s.json", root))
	if err != nil {
		return err
	}

	return nil
}

func (m *Save) saveContextToDisk(ctx *core.Context, filename string) error {
	js, _ := json.MarshalIndent(map[string]interface{}{
		"Job":       ctx.Job,
		"Execution": ctx.Execution,
	}, "", "  ")

	return m.writeFile(js, filename)
}

func (m *Save) writeFile(data []byte, filename string) error {
	return ioutil.WriteFile(filename, data, 0644)
}
