package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mcuadros/ofelia/core"
)

// DaemonCommand daemon process
type DaemonCommand struct {
	ConfigFile        string   `long:"config" description:"configuration file" default:"/etc/ofelia.conf"`
	DockerLabelConfig bool     `short:"d" long:"docker" description:"continiously poll docker labels for configurations"`
	DockerFilters     []string `short:"f" long:"docker-filter" description:"filter to select docker containers. https://docs.docker.com/reference/cli/docker/container/ls/#filter"`
	scheduler         *core.Scheduler
	signals           chan os.Signal
	done              chan bool
	Logger            core.Logger

	dockerHandlerStop func()
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

func (c *DaemonCommand) boot() (err error) {
	// Always try to read the config file, as there are options such as globals or some tasks that can be specified there and not in docker
	config, err := BuildFromFile(c.ConfigFile, c.Logger)
	if err != nil {
		if !c.DockerLabelConfig {
			return fmt.Errorf("can't read the config file: %w", err)
		} else {
			c.Logger.Debugf("Config file %v not found. Proceeding to read docker labels...", c.ConfigFile)
		}
	} else {
		msg := "Found config file %v"
		if c.DockerLabelConfig {
			msg += ". Proceeding to read docker labels as well..."
		}
		c.Logger.Debugf(msg, c.ConfigFile)
	}

	scheduler := core.NewScheduler(c.Logger)

	config.sh = scheduler
	config.buildSchedulerMiddlewares(scheduler)

	config.dockerHandler, err = NewDockerHandler(config, c.DockerFilters, c.DockerLabelConfig, c.Logger)
	if err != nil {
		return fmt.Errorf("failed to create docker handler: %w", err)
	}

	c.dockerHandlerStop = config.dockerHandler.Stop

	err = config.InitializeApp()
	if err != nil {
		return fmt.Errorf("can't start the app: %w", err)
	}

	c.scheduler = config.sh

	return err
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
		c.Logger.Warningf(
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

	if c.dockerHandlerStop != nil {
		c.dockerHandlerStop()
	}

	c.Logger.Warningf("Waiting running jobs.")
	return c.scheduler.Stop()
}
