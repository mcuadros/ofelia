package cli

import (
	"github.com/mcuadros/ofelia/core"
)

// ValidateCommand validates the config file
type ValidateCommand struct {
	ConfigFile string `long:"config" description:"configuration file" default:"/etc/ofelia.conf"`
	Logger     core.Logger
}

// Execute runs the validation command
func (c *ValidateCommand) Execute(args []string) error {
	c.Logger.Debugf("Validating %q ... ", c.ConfigFile)
	_, err := BuildFromFile(c.ConfigFile, c.Logger)
	if err != nil {
		c.Logger.Errorf("ERROR")
		return err
	}
	c.Logger.Debugf("OK")
	return nil
}
