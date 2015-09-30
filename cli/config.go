package cli

import (
	"github.com/mcuadros/go-defaults"
	"github.com/mcuadros/ofelia/core"
	"gopkg.in/gcfg.v1"
)

// Config contains the configuration
type Config struct {
	Jobs map[string]*core.ExecJob `gcfg:"Job"`
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
