package cli

import "fmt"

// ValidateCommand validates the config file
type ValidateCommand struct {
	ConfigFile string `long:"config" description:"configuration file" default:"/etc/ofelia.conf"`
}

// Execute runs the validation command
func (c *ValidateCommand) Execute(args []string) error {
	fmt.Printf("Validating %q ... ", c.ConfigFile)
	config, err := BuildFromFile("", c.ConfigFile)
	if err != nil {
		fmt.Println("ERROR")
		return err
	}

	fmt.Println("OK")
	fmt.Printf("Found %d jobs:\n", len(config.Jobs))

	for _, j := range config.Jobs {
		fmt.Printf(
			"- name: %s schedule: %q command: %q\n",
			j.GetName(), j.GetSchedule(), j.GetCommand(),
		)
	}

	return nil
}
