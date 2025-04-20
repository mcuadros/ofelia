package cli

import (
	"os"
	"regexp"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mcuadros/ofelia/core"
	"github.com/mcuadros/ofelia/middlewares"
	logging "github.com/op/go-logging"

	defaults "github.com/mcuadros/go-defaults"
	gcfg "gopkg.in/gcfg.v1"
)

const (
	logFormat     = "%{time} %{color} %{shortfile} ▶ %{level}%{color:reset} %{message}"
	jobExec       = "job-exec"
	jobRun        = "job-run"
	jobServiceRun = "job-service-run"
	jobLocal      = "job-local"
)

var IsDockerEnv bool

// Config contains the configuration
type Config struct {
	Global struct {
		middlewares.SlackConfig `mapstructure:",squash"`
		middlewares.SaveConfig  `mapstructure:",squash"`
		middlewares.MailConfig  `mapstructure:",squash"`
	}
	ExecJobs    map[string]*ExecJobConfig    `gcfg:"job-exec" mapstructure:"job-exec,squash"`
	RunJobs     map[string]*RunJobConfig     `gcfg:"job-run" mapstructure:"job-run,squash"`
	ServiceJobs map[string]*RunServiceConfig `gcfg:"job-service-run" mapstructure:"job-service-run,squash"`
	LocalJobs   map[string]*LocalJobConfig   `gcfg:"job-local" mapstructure:"job-local,squash"`
}

// BuildFromDockerLabels builds a scheduler using the config from a docker labels
func BuildFromDockerLabels(filterFlags ...string) (*core.Scheduler, error) {
	c := &Config{}

	d, err := c.buildDockerClient()
	if err != nil {
		return nil, err
	}

	labels, err := getLabels(d, filterFlags)
	if err != nil {
		return nil, err
	}

	if err := c.buildFromDockerLabels(labels); err != nil {
		return nil, err
	}

	return c.build()
}

var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func replaceEnvVariables(config string) string {
	return envVarPattern.ReplaceAllStringFunc(config, func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		if value, exists := os.LookupEnv(varName); exists {
			return value
		}
		return match
	})
}

// BuildFromFile builds a scheduler using the config from a file
func BuildFromFile(filename string) (*core.Scheduler, error) {
	c := &Config{}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	configStr := replaceEnvVariables(string(data))
	if err := gcfg.ReadStringInto(c, configStr); err != nil {
		return nil, err
	}

	return c.build()
}

// BuildFromString builds a scheduler using the config from a string
func BuildFromString(config string) (*core.Scheduler, error) {
	c := &Config{}
	config = replaceEnvVariables(config)
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

	for name, j := range c.LocalJobs {
		defaults.SetDefaults(j)

		j.Name = name
		j.buildMiddlewares()
		sh.AddJob(j)
	}

	for name, j := range c.ServiceJobs {
		defaults.SetDefaults(j)
		j.Name = name
		j.Client = d
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
	stdout := logging.NewLogBackend(os.Stdout, "", 0)
	// Set the backends to be used.
	logging.SetBackend(stdout)
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
	core.ExecJob              `mapstructure:",squash"`
	middlewares.OverlapConfig `mapstructure:",squash"`
	middlewares.SlackConfig   `mapstructure:",squash"`
	middlewares.SaveConfig    `mapstructure:",squash"`
	middlewares.MailConfig    `mapstructure:",squash"`
}

func (c *ExecJobConfig) buildMiddlewares() {
	c.ExecJob.Use(middlewares.NewOverlap(&c.OverlapConfig))
	c.ExecJob.Use(middlewares.NewSlack(&c.SlackConfig))
	c.ExecJob.Use(middlewares.NewSave(&c.SaveConfig))
	c.ExecJob.Use(middlewares.NewMail(&c.MailConfig))
}

// RunServiceConfig contains all configuration params needed to build a RunJob
type RunServiceConfig struct {
	core.RunServiceJob        `mapstructure:",squash"`
	middlewares.OverlapConfig `mapstructure:",squash"`
	middlewares.SlackConfig   `mapstructure:",squash"`
	middlewares.SaveConfig    `mapstructure:",squash"`
	middlewares.MailConfig    `mapstructure:",squash"`
}

type RunJobConfig struct {
	core.RunJob               `mapstructure:",squash"`
	middlewares.OverlapConfig `mapstructure:",squash"`
	middlewares.SlackConfig   `mapstructure:",squash"`
	middlewares.SaveConfig    `mapstructure:",squash"`
	middlewares.MailConfig    `mapstructure:",squash"`
}

func (c *RunJobConfig) buildMiddlewares() {
	c.RunJob.Use(middlewares.NewOverlap(&c.OverlapConfig))
	c.RunJob.Use(middlewares.NewSlack(&c.SlackConfig))
	c.RunJob.Use(middlewares.NewSave(&c.SaveConfig))
	c.RunJob.Use(middlewares.NewMail(&c.MailConfig))
}

// LocalJobConfig contains all configuration params needed to build a RunJob
type LocalJobConfig struct {
	core.LocalJob             `mapstructure:",squash"`
	middlewares.OverlapConfig `mapstructure:",squash"`
	middlewares.SlackConfig   `mapstructure:",squash"`
	middlewares.SaveConfig    `mapstructure:",squash"`
	middlewares.MailConfig    `mapstructure:",squash"`
}

func (c *LocalJobConfig) buildMiddlewares() {
	c.LocalJob.Use(middlewares.NewOverlap(&c.OverlapConfig))
	c.LocalJob.Use(middlewares.NewSlack(&c.SlackConfig))
	c.LocalJob.Use(middlewares.NewSave(&c.SaveConfig))
	c.LocalJob.Use(middlewares.NewMail(&c.MailConfig))
}

func (c *RunServiceConfig) buildMiddlewares() {
	c.RunServiceJob.Use(middlewares.NewOverlap(&c.OverlapConfig))
	c.RunServiceJob.Use(middlewares.NewSlack(&c.SlackConfig))
	c.RunServiceJob.Use(middlewares.NewSave(&c.SaveConfig))
	c.RunServiceJob.Use(middlewares.NewMail(&c.MailConfig))
}
