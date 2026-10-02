package core

import "testing"

func TestJobHashDetectsExecutionSettings(t *testing.T) {
	type hashJob interface{ Hash() uint64 }
	entrypoint := "/bin/sh"
	tests := []struct {
		name          string
		before, after hashJob
	}{
		{"exec container", &ExecJob{}, &ExecJob{Container: "replacement"}},
		{"exec user", &ExecJob{}, &ExecJob{User: "1000"}},
		{"exec tty", &ExecJob{}, &ExecJob{TTY: true}},
		{"exec environment", &ExecJob{}, &ExecJob{Environment: []string{"MODE=test"}}},
		{"run image", &RunJob{}, &RunJob{Image: "alpine"}},
		{"run user", &RunJob{}, &RunJob{User: "1000"}},
		{"run volume", &RunJob{}, &RunJob{Volume: []string{"/data:/data"}}},
		{"run environment", &RunJob{}, &RunJob{Environment: []string{"MODE=test"}}},
		{"run entrypoint", &RunJob{}, &RunJob{Entrypoint: &entrypoint}},
		{"service image", &RunServiceJob{}, &RunServiceJob{Image: "alpine"}},
		{"service network", &RunServiceJob{}, &RunServiceJob{Network: "replacement"}},
		{"local directory", &LocalJob{}, &LocalJob{Dir: "/tmp"}},
		{"local environment", &LocalJob{}, &LocalJob{Environment: []string{"MODE=test"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.before.Hash() == tt.after.Hash() {
				t.Fatal("execution setting change did not change the job hash")
			}
		})
	}
}

func TestJobHashIgnoresRuntimeState(t *testing.T) {
	bare := func() BareJob { return BareJob{Name: "test", Schedule: "@hourly", Command: "true"} }
	execJob := &ExecJob{BareJob: bare(), Container: "app", User: "1000"}
	runJob := &RunJob{BareJob: bare(), Image: "alpine", User: "1000"}
	serviceJob := &RunServiceJob{BareJob: bare(), Image: "alpine", Network: "jobs"}
	localJob := &LocalJob{BareJob: bare(), Dir: "/tmp"}
	client := &mockDockerClient{}

	for _, tt := range []struct {
		name   string
		hash   func() uint64
		change func()
	}{
		{"exec", execJob.Hash, func() {
			execJob.Client = client
			execJob.execID = "running-exec"
			execJob.NotifyStart()
			execJob.SetCronJobID(42)
			execJob.history = []*Execution{NewExecution()}
		}},
		{"run", runJob.Hash, func() {
			runJob.Client = client
			runJob.containerID = "running-container"
			runJob.NotifyStart()
			runJob.SetCronJobID(42)
		}},
		{"service", serviceJob.Hash, func() {
			serviceJob.Client = client
			serviceJob.NotifyStart()
			serviceJob.SetCronJobID(42)
		}},
		{"local", localJob.Hash, func() {
			localJob.NotifyStart()
			localJob.SetCronJobID(42)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := tt.hash()
			tt.change()
			if before != tt.hash() {
				t.Fatal("runtime state change changed the job hash")
			}
		})
	}
}
