package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsouza/go-dockerclient"
	"github.com/mcuadros/ofelia/core"
	"github.com/mcuadros/ofelia/middlewares"
	"github.com/op/go-logging"
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
	c.config = &Config{}
	if err := c.config.LoadFile(c.ConfigFile); err != nil {
		return err
	}

	fmt.Printf("Loaded %d job(s) from %s\n", len(c.config.Jobs), c.ConfigFile)

	d, err := docker.NewClientFromEnv()
	if err != nil {
		return err
	}

	logger := c.buildLogger()
	c.scheduler = core.NewScheduler()
	for _, j := range c.config.Jobs {
		j.Client = d
		j.Use(logger)
		c.scheduler.AddJob(j)
	}

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
		fmt.Printf("Signal recieved: %s, shuting down the process\n", sig)

		c.done <- true
	}()
}

func (c *DaemonCommand) shutdown() error {
	<-c.done
	if !c.scheduler.IsRunning() {
		return nil
	}

	fmt.Println("Waiting running jobs.")
	return c.scheduler.Stop()
}

const logFormat = "%{color}%{shortfile} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}"

func (c *DaemonCommand) buildLogger() *middlewares.Logger {
	logging.SetFormatter(logging.MustStringFormatter(logFormat))

	return middlewares.NewLogger(logging.MustGetLogger("ofelia"))

}
