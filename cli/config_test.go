package cli

import (
	"testing"

	defaults "github.com/mcuadros/go-defaults"
	"github.com/mcuadros/ofelia/core"
	"github.com/mcuadros/ofelia/middlewares"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteConfig struct{}

var _ = Suite(&SuiteConfig{})

type TestLogger struct{}

func (*TestLogger) Criticalf(format string, args ...interface{}) {}
func (*TestLogger) Debugf(format string, args ...interface{})    {}
func (*TestLogger) Errorf(format string, args ...interface{})    {}
func (*TestLogger) Noticef(format string, args ...interface{})   {}
func (*TestLogger) Warningf(format string, args ...interface{})  {}

func (s *SuiteConfig) TestBuildFromString(c *C) {
	mockLogger := TestLogger{}
	_, err := BuildFromString(`
		[job-exec "foo"]
		schedule = @every 10s

		[job-exec "bar"]
		schedule = @every 10s

		[job-run "qux"]
		schedule = @every 10s

		[job-local "baz"]
		schedule = @every 10s

		[job-service-run "bob"]
		schedule = @every 10s
  `, &mockLogger)

	c.Assert(err, IsNil)
}

func (s *SuiteConfig) TestJobDefaultsSet(c *C) {
	j := &RunJobConfig{}
	j.Pull = "false"

	defaults.SetDefaults(j)

	c.Assert(j.Pull, Equals, "false")
}

func (s *SuiteConfig) TestJobDefaultsNotSet(c *C) {
	j := &RunJobConfig{}

	defaults.SetDefaults(j)

	c.Assert(j.Pull, Equals, "true")
}

func (s *SuiteConfig) TestExecJobBuildEmpty(c *C) {
	j := &ExecJobConfig{}

	c.Assert(j.Middlewares(), HasLen, 0)
}

func (s *SuiteConfig) TestExecJobBuild(c *C) {
	j := &ExecJobConfig{}
	j.OverlapConfig.NoOverlap = true
	j.buildMiddlewares()

	c.Assert(j.Middlewares(), HasLen, 1)
}

func (s *SuiteConfig) TestLabelsConfig(c *C) {
	testcases := []struct {
		Labels         map[string]map[string]string
		ExpectedConfig Config
		Comment        string
	}{
		{
			Labels:         map[string]map[string]string{},
			ExpectedConfig: Config{},
			Comment:        "No labels, no config",
		},
		{
			Labels: map[string]map[string]string{
				"some": map[string]string{
					"label1": "1",
					"label2": "2",
				},
			},
			ExpectedConfig: Config{},
			Comment:        "No required label, no config",
		},
		{
			Labels: map[string]map[string]string{
				"some": map[string]string{
					requiredLabel: "true",
					"label2":      "2",
				},
			},
			ExpectedConfig: Config{},
			Comment:        "No prefixed labels, no config",
		},
		{
			Labels: map[string]map[string]string{
				"some": map[string]string{
					requiredLabel: "false",
					labelPrefix + "." + jobLocal + ".job1.schedule": "everyday! yey!",
				},
			},
			ExpectedConfig: Config{},
			Comment:        "With prefixed labels, but without required label still no config",
		},
		{
			Labels: map[string]map[string]string{
				"some": map[string]string{
					requiredLabel: "true",
					labelPrefix + "." + jobLocal + ".job1.schedule": "everyday! yey!",
					labelPrefix + "." + jobLocal + ".job1.command":  "rm -rf *test*",
					labelPrefix + "." + jobLocal + ".job2.schedule": "everynanosecond! yey!",
					labelPrefix + "." + jobLocal + ".job2.command":  "ls -al *test*",
				},
			},
			ExpectedConfig: Config{},
			Comment:        "No service label, no 'local' jobs",
		},

		{
			Labels: map[string]map[string]string{
				"some": map[string]string{
					requiredLabel: "true",
					serviceLabel:  "true",
					labelPrefix + "." + jobLocal + ".job1.schedule":      "schedule1",
					labelPrefix + "." + jobLocal + ".job1.command":       "command1",
					labelPrefix + "." + jobRun + ".job2.schedule":        "schedule2",
					labelPrefix + "." + jobRun + ".job2.command":         "command2",
					labelPrefix + "." + jobServiceRun + ".job3.schedule": "schedule3",
					labelPrefix + "." + jobServiceRun + ".job3.command":  "command3",
				},
				"other": map[string]string{
					requiredLabel: "true",
					labelPrefix + "." + jobLocal + ".job4.schedule":      "schedule4",
					labelPrefix + "." + jobLocal + ".job4.command":       "command4",
					labelPrefix + "." + jobRun + ".job5.schedule":        "schedule5",
					labelPrefix + "." + jobRun + ".job5.command":         "command5",
					labelPrefix + "." + jobServiceRun + ".job6.schedule": "schedule6",
					labelPrefix + "." + jobServiceRun + ".job6.command":  "command6",
				},
			},
			ExpectedConfig: Config{
				LocalJobs: map[string]*LocalJobConfig{
					"job1": &LocalJobConfig{LocalJob: core.LocalJob{BareJob: core.BareJob{
						Schedule: "schedule1",
						Command:  "command1",
					}}},
				},
				RunJobs: map[string]*RunJobConfig{
					"job2": &RunJobConfig{RunJob: core.RunJob{BareJob: core.BareJob{
						Schedule: "schedule2",
						Command:  "command2",
					}}},
					"job5": &RunJobConfig{RunJob: core.RunJob{BareJob: core.BareJob{
						Schedule: "schedule5",
						Command:  "command5",
					}}},
				},
				ServiceJobs: map[string]*RunServiceConfig{
					"job3": &RunServiceConfig{RunServiceJob: core.RunServiceJob{BareJob: core.BareJob{
						Schedule: "schedule3",
						Command:  "command3",
					}}},
				},
			},
			Comment: "Local/Run/Service jobs from non-service container ignored",
		},

		{
			Labels: map[string]map[string]string{
				"some": map[string]string{
					requiredLabel: "true",
					serviceLabel:  "true",
					labelPrefix + "." + jobExec + ".job1.schedule": "schedule1",
					labelPrefix + "." + jobExec + ".job1.command":  "command1",
				},
				"other": map[string]string{
					requiredLabel: "true",
					labelPrefix + "." + jobExec + ".job2.schedule": "schedule2",
					labelPrefix + "." + jobExec + ".job2.command":  "command2",
				},
			},
			ExpectedConfig: Config{
				ExecJobs: map[string]*ExecJobConfig{
					"job1": &ExecJobConfig{ExecJob: core.ExecJob{BareJob: core.BareJob{
						Schedule: "schedule1",
						Command:  "command1",
					}}},
					"job2": &ExecJobConfig{ExecJob: core.ExecJob{
						BareJob: core.BareJob{
							Schedule: "schedule2",
							Command:  "command2",
						},
						Container: "other",
					}},
				},
			},
			Comment: "Exec jobs from non-service container, saves container name to be able to exect to",
		},
		{
			Labels: map[string]map[string]string{
				"some": map[string]string{
					requiredLabel: "true",
					serviceLabel:  "true",
					labelPrefix + "." + jobExec + ".job1.schedule":   "schedule1",
					labelPrefix + "." + jobExec + ".job1.command":    "command1",
					labelPrefix + "." + jobExec + ".job1.no-overlap": "true",
				},
			},
			ExpectedConfig: Config{
				ExecJobs: map[string]*ExecJobConfig{
					"job1": &ExecJobConfig{ExecJob: core.ExecJob{BareJob: core.BareJob{
						Schedule: "schedule1",
						Command:  "command1",
					}},
						OverlapConfig: middlewares.OverlapConfig{NoOverlap: true},
					},
				},
			},
			Comment: "Test job with 'no-overlap' set",
		},
		{
			Labels: map[string]map[string]string{
				"some": {
					requiredLabel: "true",
					serviceLabel:  "true",
					labelPrefix + "." + jobRun + ".job1.schedule": "schedule1",
					labelPrefix + "." + jobRun + ".job1.command":  "command1",
					labelPrefix + "." + jobRun + ".job1.volume":   "/test/tmp:/test/tmp:ro",
					labelPrefix + "." + jobRun + ".job2.schedule": "schedule2",
					labelPrefix + "." + jobRun + ".job2.command":  "command2",
					labelPrefix + "." + jobRun + ".job2.volume":   `["/test/tmp:/test/tmp:ro", "/test/tmp:/test/tmp:rw"]`,
				},
			},
			ExpectedConfig: Config{
				RunJobs: map[string]*RunJobConfig{
					"job1": {RunJob: core.RunJob{BareJob: core.BareJob{
						Schedule: "schedule1",
						Command:  "command1",
					},
						Volume: []string{"/test/tmp:/test/tmp:ro"},
					},
					},
					"job2": {RunJob: core.RunJob{BareJob: core.BareJob{
						Schedule: "schedule2",
						Command:  "command2",
					},
						Volume: []string{"/test/tmp:/test/tmp:ro", "/test/tmp:/test/tmp:rw"},
					},
					},
				},
			},
			Comment: "Test run job with volumes",
		},
	}

	for _, t := range testcases {
		var conf = Config{}
		err := conf.buildFromDockerLabels(t.Labels)
		c.Assert(err, IsNil)
		c.Assert(conf, DeepEquals, t.ExpectedConfig)
	}
}
