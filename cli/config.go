package cli

import (
	"github.com/mcuadros/go-defaults"
	"github.com/mcuadros/ofelia/core"
	"github.com/mcuadros/ofelia/middlewares"
	"gopkg.in/gcfg.v1"
)

// Config contains the configuration
type Config struct {
	Jobs map[string]*ExecJobConfig `gcfg:"Job"`
}

// LoadFile loads the content into the Config struct
func (c *Config) LoadFile(filename string) error {
	err := gcfg.ReadFileInto(c, filename)
	if err != nil {
		return err
	}

	c.loadDefaults()
	return nil
}

func (c *Config) loadDefaults() {
	defaults.SetDefaults(c)
	for name, j := range c.Jobs {
		j.Name = name
		defaults.SetDefaults(j)
	}
}

// ExecJobConfig contains all configuration params needed to build a ExecJob
type ExecJobConfig struct {
	core.ExecJob
	middlewares.OverlapConfig
	middlewares.SlackConfig
}

// Build instanciates all the middlewares configured
func (c *ExecJobConfig) Build() {
	var ms []core.Middleware
	ms = append(ms, middlewares.NewOverlap(&c.OverlapConfig))
	ms = append(ms, middlewares.NewSlack(&c.SlackConfig))

	for _, m := range ms {
		if m == nil {
			continue
		}

		c.ExecJob.Use(m)
	}
}
