package cli

import (
	"encoding/json"
	"testing"

	defaults "github.com/mcuadros/go-defaults"
	"github.com/mcuadros/ofelia/core"
	"github.com/mcuadros/ofelia/middlewares"
	. "gopkg.in/check.v1"
	gcfg "gopkg.in/gcfg.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteConfig struct{}

var _ = Suite(&SuiteConfig{})

func (s *SuiteConfig) TestBuildFromString(c *C) {
	sh, err := BuildFromString("", `
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
  `)

	c.Assert(err, IsNil)
	c.Assert(sh.Jobs, HasLen, 5)
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

func (s *SuiteConfig) TestConfigIni(c *C) {
	testcases := []struct {
		Ini            string
		ExpectedConfig Config
		Comment        string
	}{
		{
			Ini: `
				[job-exec "foo"]
				schedule = @every 10s
				command = echo \"foo\"
				`,
			ExpectedConfig: Config{
				ExecJobs: map[string]*ExecJobConfig{
					"foo": {ExecJob: core.ExecJob{BareJob: core.BareJob{
						Schedule: "@every 10s",
						Command:  `echo "foo"`,
					}}},
				},
			},
			Comment: "Test job-exec",
		},
		{
			Ini: `
				[job-run "foo"]
				schedule = @every 10s
				environment = "KEY1=value1"
				Environment = "KEY2=value2"
				`,
			ExpectedConfig: Config{
				RunJobs: map[string]*RunJobConfig{
					"foo": {RunJob: core.RunJob{BareJob: core.BareJob{
						Schedule: "@every 10s",
					},
						Environment: []string{"KEY1=value1", "KEY2=value2"},
					}},
				},
			},
			Comment: "Test job-run with Env Variables",
		},
		{
			Ini: `
				[job-run "foo"]
				schedule = @every 10s
				volumes-from = "volume1"
				volumes-from = "volume2"
				`,
			ExpectedConfig: Config{
				RunJobs: map[string]*RunJobConfig{
					"foo": {RunJob: core.RunJob{BareJob: core.BareJob{
						Schedule: "@every 10s",
					},
						VolumesFrom: []string{"volume1", "volume2"},
					}},
				},
			},
			Comment: "Test job-run with Env Variables",
		},
	}

	for _, t := range testcases {
		conf := Config{}
		err := gcfg.ReadStringInto(&conf, t.Ini)
		c.Assert(err, IsNil)
		if !c.Check(conf, DeepEquals, t.ExpectedConfig) {
			c.Errorf("Test %q\nExpected %s, but got %s", t.Comment, toJSON(t.ExpectedConfig), toJSON(conf))
		}
	}
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
		{
			Labels: map[string]map[string]string{
				"some": {
					requiredLabel: "true",
					serviceLabel:  "true",
					labelPrefix + "." + jobRun + ".job1.schedule":    "schedule1",
					labelPrefix + "." + jobRun + ".job1.command":     "command1",
					labelPrefix + "." + jobRun + ".job1.environment": "KEY1=value1",
					labelPrefix + "." + jobRun + ".job2.schedule":    "schedule2",
					labelPrefix + "." + jobRun + ".job2.command":     "command2",
					labelPrefix + "." + jobRun + ".job2.environment": `["KEY1=value1", "KEY2=value2"]`,
				},
			},
			ExpectedConfig: Config{
				RunJobs: map[string]*RunJobConfig{
					"job1": {RunJob: core.RunJob{BareJob: core.BareJob{
						Schedule: "schedule1",
						Command:  "command1",
					},
						Environment: []string{"KEY1=value1"},
					},
					},
					"job2": {RunJob: core.RunJob{BareJob: core.BareJob{
						Schedule: "schedule2",
						Command:  "command2",
					},
						Environment: []string{"KEY1=value1", "KEY2=value2"},
					},
					},
				},
			},
			Comment: "Test run job with environment variables",
		},
		{
			Labels: map[string]map[string]string{
				"some": {
					requiredLabel: "true",
					serviceLabel:  "true",
					labelPrefix + "." + jobRun + ".job1.schedule":     "schedule1",
					labelPrefix + "." + jobRun + ".job1.command":      "command1",
					labelPrefix + "." + jobRun + ".job1.volumes-from": "test123",
					labelPrefix + "." + jobRun + ".job2.schedule":     "schedule2",
					labelPrefix + "." + jobRun + ".job2.command":      "command2",
					labelPrefix + "." + jobRun + ".job2.volumes-from": `["test321", "test456"]`,
				},
			},
			ExpectedConfig: Config{
				RunJobs: map[string]*RunJobConfig{
					"job1": {
						RunJob: core.RunJob{
							BareJob: core.BareJob{
								Schedule: "schedule1",
								Command:  "command1",
							},
							VolumesFrom: []string{"test123"},
						},
					},
					"job2": {
						RunJob: core.RunJob{
							BareJob: core.BareJob{
								Schedule: "schedule2",
								Command:  "command2",
							},
							VolumesFrom: []string{"test321", "test456"},
						},
					},
				},
			},
			Comment: "Test run job with volumes-from",
		},
	}

	for _, t := range testcases {
		var conf = Config{}
		err := conf.buildFromDockerLabels(t.Labels)
		c.Assert(err, IsNil)
		if !c.Check(conf, DeepEquals, t.ExpectedConfig) {
			c.Errorf("Test %q\nExpected %s, but got %s", t.Comment, toJSON(t.ExpectedConfig), toJSON(conf))
		}
	}
}

func toJSON(any interface{}) string {
	b, _ := json.MarshalIndent(any, "", "  ")
	return string(b)
}
