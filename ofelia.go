package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/mcuadros/ofelia/cli"
)

var version string
var build string

func main() {
	parser := flags.NewNamedParser("ofelia", flags.Default)
	parser.AddCommand("daemon", "daemon process", "", &cli.DaemonCommand{})
	parser.AddCommand("validate", "validates the config file", "", &cli.ValidateCommand{})

	if _, err := parser.Parse(); err != nil {
		if _, ok := err.(*flags.Error); ok {
			parser.WriteHelp(os.Stdout)
			fmt.Printf("\nBuild information\n  commit: %s\n  date:%s\n", version, build)
		}

		os.Exit(1)
	}
}
