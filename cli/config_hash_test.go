package cli

import (
	"testing"

	defaults "github.com/mcuadros/go-defaults"
	"github.com/mcuadros/ofelia/core"
)

func TestDockerLabelsUpdateExecutionSettings(t *testing.T) {
	for _, tt := range []struct {
		name, jobType, parameter, before, after string
	}{
		{"exec user", jobExec, "user", "nobody", "1000"},
		{"exec tty", jobExec, "tty", "false", "true"},
		{"exec environment", jobExec, "environment", `["MODE=old"]`, `["MODE=new"]`},
		{"exec container", jobExec, "container", "app-old", "app-new"},
		{"run user", jobRun, "user", "nobody", "1000"},
		{"run image", jobRun, "image", "alpine:3.23", "alpine:3.24"},
		{"run volume", jobRun, "volume", `["/old:/data"]`, `["/new:/data"]`},
		{"run environment", jobRun, "environment", `["MODE=old"]`, `["MODE=new"]`},
		{"run entrypoint", jobRun, "entrypoint", "/bin/sh", "/bin/ash"},
		{"run container", jobRun, "container", "app-old", "app-new"},
		{"run network", jobRun, "network", "old-network", "new-network"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prefix := labelPrefix + "." + tt.jobType + ".test."
			labels := map[string]map[string]string{
				"ofelia": {
					serviceLabel:          "true",
					prefix + "schedule":   "@hourly",
					prefix + "command":    "true",
					prefix + tt.parameter: tt.before,
				},
			}
			config := NewConfig(&TestLogger{})
			config.sh = core.NewScheduler(&TestLogger{})
			config.dockerHandler = &DockerHandler{dockerClient: &mockCLIDockerClient{}}
			config.dockerLabelsUpdate(labels)
			entries := config.sh.CronJobs()
			if len(entries) != 1 {
				t.Fatalf("expected one scheduled job, got %d", len(entries))
			}
			originalID := entries[0].ID

			labels["ofelia"][prefix+tt.parameter] = tt.after
			config.dockerLabelsUpdate(labels)
			entries = config.sh.CronJobs()
			if len(entries) != 1 || entries[0].ID == originalID {
				t.Fatal("changed execution setting did not replace the scheduled job")
			}
			replacementID := entries[0].ID

			var expected Config
			if err := expected.buildFromDockerLabels(labels); err != nil {
				t.Fatal(err)
			}
			if tt.jobType == jobExec {
				got, want := config.ExecJobs["test"], expected.ExecJobs["test"]
				defaults.SetDefaults(want)
				want.Name = "test"
				if toJSON(got) != toJSON(want) {
					t.Fatalf("updated exec settings differ: got %s, want %s", toJSON(got), toJSON(want))
				}
			} else {
				got, want := config.RunJobs["test"], expected.RunJobs["test"]
				defaults.SetDefaults(want)
				want.Name = "test"
				if toJSON(got) != toJSON(want) {
					t.Fatalf("updated run settings differ: got %s, want %s", toJSON(got), toJSON(want))
				}
			}

			config.dockerHandler.dockerClient = &mockCLIDockerClient{}
			config.dockerLabelsUpdate(labels)
			entries = config.sh.CronJobs()
			if len(entries) != 1 || entries[0].ID != replacementID {
				t.Fatal("unchanged labels replaced the scheduled job")
			}
		})
	}
}
