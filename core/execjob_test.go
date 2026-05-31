package core

import (
	"bufio"
	"context"
	"net"

	"github.com/moby/moby/client"
	. "gopkg.in/check.v1"
)

const ContainerFixture = "test-container"

type SuiteExecJob struct{}

var _ = Suite(&SuiteExecJob{})

func (s *SuiteExecJob) TestRun(c *C) {
	var createdOpts client.ExecCreateOptions

	mock := &mockDockerClient{
		ExecCreateFn: func(ctx context.Context, containerID string, options client.ExecCreateOptions) (client.ExecCreateResult, error) {
			createdOpts = options
			c.Assert(containerID, Equals, ContainerFixture)
			return client.ExecCreateResult{ID: "exec-123"}, nil
		},
		ExecAttachFn: func(ctx context.Context, execID string, options client.ExecAttachOptions) (client.ExecAttachResult, error) {
			c.Assert(execID, Equals, "exec-123")
			c.Assert(options.TTY, Equals, true)
			serverConn, clientConn := net.Pipe()
			go func() {
				serverConn.Write([]byte("output"))
				serverConn.Close()
			}()
			return client.ExecAttachResult{
				HijackedResponse: client.HijackedResponse{
					Conn:   clientConn,
					Reader: bufio.NewReader(clientConn),
				},
			}, nil
		},
		ExecInspectFn: func(ctx context.Context, execID string, options client.ExecInspectOptions) (client.ExecInspectResult, error) {
			c.Assert(execID, Equals, "exec-123")
			return client.ExecInspectResult{ExitCode: 0}, nil
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
	c.Assert(createdOpts.TTY, Equals, true)
	c.Assert(createdOpts.Env, DeepEquals, []string{"test_Key1=value1", "test_Key2=value2"})
	c.Assert(createdOpts.AttachStdout, Equals, true)
	c.Assert(createdOpts.AttachStderr, Equals, true)

	c.Assert(e.OutputStream.String(), Equals, "output")
}

func (s *SuiteExecJob) TestRunNonZeroExit(c *C) {
	mock := &mockDockerClient{
		ExecAttachFn: func(ctx context.Context, execID string, options client.ExecAttachOptions) (client.ExecAttachResult, error) {
			serverConn, clientConn := net.Pipe()
			serverConn.Close()
			return client.ExecAttachResult{
				HijackedResponse: client.HijackedResponse{
					Conn:   clientConn,
					Reader: bufio.NewReader(clientConn),
				},
			}, nil
		},
		ExecInspectFn: func(ctx context.Context, execID string, options client.ExecInspectOptions) (client.ExecInspectResult, error) {
			return client.ExecInspectResult{ExitCode: 1}, nil
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
		ExecAttachFn: func(ctx context.Context, execID string, options client.ExecAttachOptions) (client.ExecAttachResult, error) {
			serverConn, clientConn := net.Pipe()
			serverConn.Close()
			return client.ExecAttachResult{
				HijackedResponse: client.HijackedResponse{
					Conn:   clientConn,
					Reader: bufio.NewReader(clientConn),
				},
			}, nil
		},
		ExecInspectFn: func(ctx context.Context, execID string, options client.ExecInspectOptions) (client.ExecInspectResult, error) {
			return client.ExecInspectResult{ExitCode: -1}, nil
		},
	}

	job := &ExecJob{Client: mock}
	job.Container = ContainerFixture
	job.Command = "fail"

	e := NewExecution()
	err := job.Run(&Context{Execution: e})
	c.Assert(err, Equals, ErrUnexpected)
}
