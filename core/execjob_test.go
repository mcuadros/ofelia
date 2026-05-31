package core

import (
	"bufio"
	"bytes"
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	. "gopkg.in/check.v1"
)

const ContainerFixture = "test-container"

type SuiteExecJob struct{}

var _ = Suite(&SuiteExecJob{})

func (s *SuiteExecJob) TestRun(c *C) {
	var createdOpts container.ExecOptions

	mock := &mockDockerClient{
		ContainerExecCreateFn: func(ctx context.Context, ctr string, config container.ExecOptions) (container.ExecCreateResponse, error) {
			createdOpts = config
			c.Assert(ctr, Equals, ContainerFixture)
			return container.ExecCreateResponse{ID: "exec-123"}, nil
		},
		ContainerExecAttachFn: func(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error) {
			c.Assert(execID, Equals, "exec-123")
			c.Assert(config.Tty, Equals, true)
			return types.HijackedResponse{
				Reader: bufio.NewReader(bytes.NewReader([]byte("output"))),
			}, nil
		},
		ContainerExecInspectFn: func(ctx context.Context, execID string) (container.ExecInspect, error) {
			c.Assert(execID, Equals, "exec-123")
			return container.ExecInspect{ExitCode: 0}, nil
		},
	}

	job := &ExecJob{Client: mock}
	job.Container = ContainerFixture
	job.Command = `echo -a "foo bar"`
	job.Environment = []string{"test_Key1=value1", "test_Key2=value2"}
	job.User = "foo"
	job.TTY = true

	e := NewExecution()

	err := job.Run(&Context{Execution: e})
	c.Assert(err, IsNil)

	c.Assert(createdOpts.Cmd, DeepEquals, []string{"echo", "-a", "foo bar"})
	c.Assert(createdOpts.User, Equals, "foo")
	c.Assert(createdOpts.Tty, Equals, true)
	c.Assert(createdOpts.Env, DeepEquals, []string{"test_Key1=value1", "test_Key2=value2"})
	c.Assert(createdOpts.AttachStdout, Equals, true)
	c.Assert(createdOpts.AttachStderr, Equals, true)

	c.Assert(e.OutputStream.String(), Equals, "output")
}

func (s *SuiteExecJob) TestRunNonZeroExit(c *C) {
	mock := &mockDockerClient{
		ContainerExecAttachFn: func(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error) {
			return types.HijackedResponse{Reader: bufio.NewReader(bytes.NewReader(nil))}, nil
		},
		ContainerExecInspectFn: func(ctx context.Context, execID string) (container.ExecInspect, error) {
			return container.ExecInspect{ExitCode: 1}, nil
		},
	}

	job := &ExecJob{Client: mock}
	job.Container = ContainerFixture
	job.Command = "fail"

	e := NewExecution()
	err := job.Run(&Context{Execution: e})
	c.Assert(err, ErrorMatches, "error non-zero exit code: 1")
}

func (s *SuiteExecJob) TestRunUnexpectedExit(c *C) {
	mock := &mockDockerClient{
		ContainerExecAttachFn: func(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error) {
			return types.HijackedResponse{Reader: bufio.NewReader(bytes.NewReader(nil))}, nil
		},
		ContainerExecInspectFn: func(ctx context.Context, execID string) (container.ExecInspect, error) {
			return container.ExecInspect{ExitCode: -1}, nil
		},
	}

	job := &ExecJob{Client: mock}
	job.Container = ContainerFixture
	job.Command = "fail"

	e := NewExecution()
	err := job.Run(&Context{Execution: e})
	c.Assert(err, Equals, ErrUnexpected)
}
