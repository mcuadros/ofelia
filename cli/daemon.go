package cli

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mcuadros/ofelia/core"
)

// DaemonCommand daemon process
type DaemonCommand struct {
	ConfigFile    string   `long:"config" description:"configuration file" default:"/etc/ofelia.conf"`
	DockerFilters []string `short:"f" long:"docker-filter" description:"filter to select docker containers. https://docs.docker.com/reference/cli/docker/container/ls/#filter"`
	scheduler     *core.Scheduler
	signals       chan os.Signal
	done          chan bool
	Logger        core.Logger
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
		c.Logger.Debugf("Config file: %v not found", c.ConfigFile)
	}
	scheduler := core.NewScheduler(c.Logger)

	config.sh = scheduler
	config.buildSchedulerMiddlewares(scheduler)

	config.dockerHandler, err = NewDockerHandler(config, c.DockerFilters, c.Logger)
	if err != nil {
		return err
	}

	err = config.InitializeApp()
	if err != nil {
		c.Logger.Criticalf("Can't start the app: %v", err)
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

	c.Logger.Warningf("Waiting running jobs.")
	return c.scheduler.Stop()
}
