package cli

import (
	"github.com/fsouza/go-dockerclient"
	"github.com/mcuadros/ofelia/core"
	"github.com/mcuadros/ofelia/middlewares"
	"github.com/op/go-logging"

	"github.com/mcuadros/go-defaults"
	"gopkg.in/gcfg.v1"
)

const logFormat = "%{color}%{shortfile} ▶ %{level}%{color:reset} %{message}"

// Config contains the configuration
type Config struct {
	Global struct {
		middlewares.SlackConfig
		middlewares.SaveConfig
		middlewares.MailConfig
	}
	ExecJobs map[string]*ExecJobConfig `gcfg:"job-exec"`
	RunJobs  map[string]*RunJobConfig  `gcfg:"job-run"`
}

// BuildFromFile buils a scheduler using the config from a file
func BuildFromFile(filename string) (*core.Scheduler, error) {
	c := &Config{}
	if err := gcfg.ReadFileInto(c, filename); err != nil {
		return nil, err
	}

	return c.build()
}

// BuildFromString buils a scheduler using the config from a string
func BuildFromString(config string) (*core.Scheduler, error) {
	c := &Config{}
	if err := gcfg.ReadStringInto(c, config); err != nil {
		return nil, err
	}

	return c.build()
}

func (c *Config) build() (*core.Scheduler, error) {
	defaults.SetDefaults(c)

	d, err := c.buildDockerClient()
	if err != nil {
		return nil, err
	}

	sh := core.NewScheduler(c.buildLogger())
	c.buildSchedulerMiddlewares(sh)

	for name, j := range c.ExecJobs {
		defaults.SetDefaults(j)

		j.Client = d
		j.Name = name
		j.buildMiddlewares()
		sh.AddJob(j)
	}

	for name, j := range c.RunJobs {
		defaults.SetDefaults(j)

		j.Client = d
		j.Name = name
		j.buildMiddlewares()
		sh.AddJob(j)
	}

	return sh, nil
}

func (c *Config) buildDockerClient() (*docker.Client, error) {
	d, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (c *Config) buildLogger() core.Logger {
	logging.SetFormatter(logging.MustStringFormatter(logFormat))

	return logging.MustGetLogger("ofelia")
}

func (c *Config) buildSchedulerMiddlewares(sh *core.Scheduler) {
	sh.Use(middlewares.NewSlack(&c.Global.SlackConfig))
	sh.Use(middlewares.NewSave(&c.Global.SaveConfig))
	sh.Use(middlewares.NewMail(&c.Global.MailConfig))
}

// ExecJobConfig contains all configuration params needed to build a ExecJob
type ExecJobConfig struct {
	core.ExecJob
	middlewares.OverlapConfig
	middlewares.SlackConfig
	middlewares.SaveConfig
	middlewares.MailConfig
}

func (c *ExecJobConfig) buildMiddlewares() {
	c.ExecJob.Use(middlewares.NewOverlap(&c.OverlapConfig))
	c.ExecJob.Use(middlewares.NewSlack(&c.SlackConfig))
	c.ExecJob.Use(middlewares.NewSave(&c.SaveConfig))
	c.ExecJob.Use(middlewares.NewMail(&c.MailConfig))
}

// RunJobConfig contains all configuration params needed to build a RunJob
type RunJobConfig struct {
	core.RunJob
	middlewares.OverlapConfig
	middlewares.SlackConfig
	middlewares.SaveConfig
	middlewares.MailConfig
}

func (c *RunJobConfig) buildMiddlewares() {
	c.RunJob.Use(middlewares.NewOverlap(&c.OverlapConfig))
	c.RunJob.Use(middlewares.NewSlack(&c.SlackConfig))
	c.RunJob.Use(middlewares.NewSave(&c.SaveConfig))
	c.RunJob.Use(middlewares.NewMail(&c.MailConfig))
}
