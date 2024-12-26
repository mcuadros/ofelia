package cli

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mcuadros/ofelia/core"
)

// DaemonCommand daemon process
type DaemonCommand struct {
	ConfigFile         string   `long:"config" description:"configuration file" default:"/etc/ofelia.conf"`
	DockerLabelsConfig bool     `short:"d" long:"docker" description:"read configurations from docker labels"`
	DockerFilters      []string `short:"f" long:"docker-filter" description:"filter to select docker containers"`
	LogLevel           string   `long:"log-level" description:"log level" default:"DEBUG" choice:"DEBUG" choice:"WARNING" choice:"ERROR"`

	config    *Config
	scheduler *core.Scheduler
	signals   chan os.Signal
	done      chan bool
}

// Execute runs the daemon
func (c *DaemonCommand) Execute(args []string) error {
	_, err := os.Stat("/.dockerenv")
	IsDockerEnv = !os.IsNotExist(err)

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

func (c *DaemonCommand) boot() (err error) {
	if c.DockerLabelsConfig {
		c.scheduler, err = BuildFromDockerLabels(c.LogLevel, c.DockerFilters...)
	} else {
		c.scheduler, err = BuildFromFile(c.LogLevel, c.ConfigFile)
	}

	return
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
		c.scheduler.Logger.Warningf(
			"Signal received: %s, shutting down the process\n", sig,
		)

		c.done <- true
	}()
}

func (c *DaemonCommand) shutdown() error {
	<-c.done
	if !c.scheduler.IsRunning() {
		return nil
	}

	c.scheduler.Logger.Warningf("Waiting running jobs.")
	return c.scheduler.Stop()
}
