package cli

import (
	"os"

	"github.com/mcuadros/ofelia/core"
	"github.com/mcuadros/ofelia/middlewares"
	logging "github.com/op/go-logging"

	defaults "github.com/mcuadros/go-defaults"
	gcfg "gopkg.in/gcfg.v1"
)

const (
	logFormat     = "%{color}%{shortfile} ▶ %{level}%{color:reset} %{message}"
	jobExec       = "job-exec"
	jobRun        = "job-run"
	jobServiceRun = "job-service-run"
	jobLocal      = "job-local"
)

// Config contains the configuration
type Config struct {
	Global struct {
		middlewares.SlackConfig `mapstructure:",squash"`
		middlewares.SaveConfig  `mapstructure:",squash"`
		middlewares.MailConfig  `mapstructure:",squash"`
	}
	ExecJobs      map[string]*ExecJobConfig    `gcfg:"job-exec" mapstructure:"job-exec,squash"`
	RunJobs       map[string]*RunJobConfig     `gcfg:"job-run" mapstructure:"job-run,squash"`
	ServiceJobs   map[string]*RunServiceConfig `gcfg:"job-service-run" mapstructure:"job-service-run,squash"`
	LocalJobs     map[string]*LocalJobConfig   `gcfg:"job-local" mapstructure:"job-local,squash"`
	sh            *core.Scheduler
	dockerHandler *DockerHandler
}

func NewConfig() *Config {
	// Initialize
	c := &Config{}
	c.ExecJobs = make(map[string]*ExecJobConfig)
	c.RunJobs = make(map[string]*RunJobConfig)
	c.ServiceJobs = make(map[string]*RunServiceConfig)
	c.LocalJobs = make(map[string]*LocalJobConfig)
	return c
}

// BuildFromDockerLabels builds a scheduler using the config from a docker labels
func BuildFromDockerLabels() (*core.Scheduler, error) {
	c := NewConfig()
	return c.build()
}

// BuildFromFile builds a scheduler using the config from a file
func BuildFromFile(filename string) (*core.Scheduler, error) {
	c := NewConfig()
	if err := gcfg.ReadFileInto(c, filename); err != nil {
		return nil, err
	}

	return c.build()
}

// BuildFromString builds a scheduler using the config from a string
func BuildFromString(config string) (*core.Scheduler, error) {
	c := &Config{}
	if err := gcfg.ReadStringInto(c, config); err != nil {
		return nil, err
	}

	return c.build()
}

func (c *Config) build() (*core.Scheduler, error) {
	defaults.SetDefaults(c)

	c.sh = core.NewScheduler(c.buildLogger())
	c.buildSchedulerMiddlewares(c.sh)

	var err error
	c.dockerHandler, err = NewDockerHandler(c)
	if err != nil {
		return nil, err
	}

	for name, j := range c.ExecJobs {
		defaults.SetDefaults(j)

		j.Client = c.dockerHandler.GetInternalDockerClient()
		j.Name = name
		j.buildMiddlewares()
		c.sh.AddJob(j)
	}

	for name, j := range c.RunJobs {
		defaults.SetDefaults(j)

		j.Client = c.dockerHandler.GetInternalDockerClient()
		j.Name = name
		j.buildMiddlewares()
		c.sh.AddJob(j)
	}

	for name, j := range c.LocalJobs {
		defaults.SetDefaults(j)

		j.Name = name
		j.buildMiddlewares()
		c.sh.AddJob(j)
	}

	for name, j := range c.ServiceJobs {
		defaults.SetDefaults(j)
		j.Name = name
		j.Client = c.dockerHandler.GetInternalDockerClient()
		j.buildMiddlewares()
		c.sh.AddJob(j)
	}

	return c.sh, nil
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

func (c *Config) dockerLabelsUpdate(labels map[string]map[string]string) {
	// Get the current labels
	var parsedLabelConfig Config
	parsedLabelConfig.buildFromDockerLabels(labels)

	// Calculate the delta
	for name, j := range c.ExecJobs {
		found := false
		for newJobsName, newJob := range parsedLabelConfig.ExecJobs {
			// Check if the schedule has changed
			if name == newJobsName {
				found = true
				// There is a slight race condition were a job can be canceled / restarted with a different schedule
				// so, lets take care of it by simply restarting
				if newJob.GetSchedule() != j.GetSchedule() {
					// Restart the job
					// Remove from the scheduler
					c.sh.RemoveJob(j)
					// Update the job config
					c.ExecJobs[name] = newJob
					c.sh.AddJob(j)
				}
				break
			}
		}
		if !found {
			// Remove the job
			c.sh.RemoveJob(j)
			delete(c.ExecJobs, name)
		}
	}

	// Check for aditions
	for newJobsName, newJob := range parsedLabelConfig.ExecJobs {
		found := false
		for name := range c.ExecJobs {
			if name == newJobsName {
				found = true
				break
			}
		}
		if !found {
			defaults.SetDefaults(newJob)
			newJob.Client = c.dockerHandler.GetInternalDockerClient()
			newJob.Name = newJobsName
			newJob.buildMiddlewares()
			c.sh.AddJob(newJob)
			c.ExecJobs[newJobsName] = newJob
		}
	}

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
