package cli

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mcuadros/ofelia/core"
)

// DaemonCommand daemon process
type DaemonCommand struct {
	ConfigFile string `long:"config" description:"configuration file" default:"/etc/ofelia.conf"`

	config    *Config
	scheduler *core.Scheduler
	signals   chan os.Signal
	done      chan bool
}

// Execute runs the daemon
func (c *DaemonCommand) Execute(args []string) error {
	if err := c.boot(); err != nil {
		return err
	}

	if err := c.start(); err != nil {
		return err
	}

	if err := c.shutdown(); err != nil {
		return err
	}

	return nil
}

func (c *DaemonCommand) boot() error {
	sh, err := BuildFromFile(c.ConfigFile)
	if err != nil {
		return err
	}

	c.scheduler = sh
	return nil
}

func (c *DaemonCommand) start() error {
	c.setSignals()
	if err := c.scheduler.Start(); err != nil {
		return err
	}

	return nil
}

func (c *DaemonCommand) setSignals() {
	c.signals = make(chan os.Signal, 1)
	c.done = make(chan bool, 1)

	signal.Notify(c.signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-c.signals
		c.scheduler.Logger.Warning(
			"Signal recieved: %s, shuting down the process\n", sig,
		)

		c.done <- true
	}()
}

func (c *DaemonCommand) shutdown() error {
	<-c.done
	if !c.scheduler.IsRunning() {
		return nil
	}

	c.scheduler.Logger.Warning("Waiting running jobs.")
	return c.scheduler.Stop()
}
