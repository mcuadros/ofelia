package core

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	. "gopkg.in/check.v1"
)

const ServiceImageFixture = "test-image"

type SuiteRunServiceJob struct{}

var _ = Suite(&SuiteRunServiceJob{})

var logger Logger

func (s *SuiteRunServiceJob) SetUpTest(c *C) {
	logger = NewSlogLogger(io.Discard)
}

func (s *SuiteRunServiceJob) TestRun(c *C) {
	var createdOpts client.ServiceCreateOptions

	mock := &mockDockerClient{
		ImagePullFn: func(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
			return &mockPullResponse{}, nil
		},
		ServiceCreateFn: func(ctx context.Context, options client.ServiceCreateOptions) (client.ServiceCreateResult, error) {
			createdOpts = options
			return client.ServiceCreateResult{ID: "svc-123"}, nil
		},
		ServiceInspectFn: func(ctx context.Context, serviceID string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
			return client.ServiceInspectResult{
				Service: swarm.Service{
					ID: serviceID,
					Meta: swarm.Meta{
						CreatedAt: time.Now(),
					},
				},
			}, nil
		},
		TaskListFn: func(ctx context.Context, options client.TaskListOptions) (client.TaskListResult, error) {
			return client.TaskListResult{
				Items: []swarm.Task{
					{
						Status: swarm.TaskStatus{
							State: swarm.TaskStateComplete,
							ContainerStatus: &swarm.ContainerStatus{
								ExitCode: 0,
							},
						},
						Spec: swarm.TaskSpec{
							ContainerSpec: &swarm.ContainerSpec{
								Command: strings.Split("echo -a foo bar", " "),
							},
						},
						ServiceID: "svc-123",
					},
				},
			}, nil
		},
		ServiceRemoveFn: func(ctx context.Context, serviceID string, options client.ServiceRemoveOptions) (client.ServiceRemoveResult, error) {
			c.Assert(serviceID, Equals, "svc-123")
			return client.ServiceRemoveResult{}, nil
		},
	}

	job := &RunServiceJob{Client: mock}
	job.Image = ServiceImageFixture
	job.Command = `echo -a foo bar`
	job.User = "foo"
	job.TTY = true
	job.Delete = "true"
	job.Network = "foo"

	e := NewExecution()
	err := job.Run(&Context{Execution: e, Logger: logger})
	c.Assert(err, IsNil)

	c.Assert(createdOpts.Spec.TaskTemplate.ContainerSpec.Command, DeepEquals, []string{"echo", "-a", "foo", "bar"})
	c.Assert(createdOpts.Spec.TaskTemplate.ContainerSpec.Image, Equals, ServiceImageFixture)
	c.Assert(createdOpts.Spec.TaskTemplate.Networks, DeepEquals, []swarm.NetworkAttachmentConfig{{Target: "foo"}})
	c.Assert(createdOpts.Spec.TaskTemplate.RestartPolicy.Condition, Equals, swarm.RestartPolicyConditionNone)
}

func (s *SuiteRunServiceJob) TestQuotedCommandPreserved(c *C) {
	var createdOpts client.ServiceCreateOptions

	mock := &mockDockerClient{
		ImagePullFn: func(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
			return &mockPullResponse{}, nil
		},
		ServiceCreateFn: func(ctx context.Context, options client.ServiceCreateOptions) (client.ServiceCreateResult, error) {
			createdOpts = options
			return client.ServiceCreateResult{ID: "svc-123"}, nil
		},
		ServiceInspectFn: func(ctx context.Context, serviceID string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
			return client.ServiceInspectResult{
				Service: swarm.Service{
					ID: serviceID,
					Meta: swarm.Meta{
						CreatedAt: time.Now(),
					},
				},
			}, nil
		},
		TaskListFn: func(ctx context.Context, options client.TaskListOptions) (client.TaskListResult, error) {
			return client.TaskListResult{
				Items: []swarm.Task{
					{
						Status: swarm.TaskStatus{
							State: swarm.TaskStateComplete,
							ContainerStatus: &swarm.ContainerStatus{
								ExitCode: 0,
							},
						},
						Spec: swarm.TaskSpec{
							ContainerSpec: &swarm.ContainerSpec{
								Command: []string{"/bin/sh", "-c", "echo \"hello world\""},
							},
						},
						ServiceID: "svc-123",
					},
				},
			}, nil
		},
		ServiceRemoveFn: func(ctx context.Context, serviceID string, options client.ServiceRemoveOptions) (client.ServiceRemoveResult, error) {
			return client.ServiceRemoveResult{}, nil
		},
	}

	job := &RunServiceJob{Client: mock}
	job.Image = ServiceImageFixture
	job.Command = `/bin/sh -c "echo \"hello world\""`

	e := NewExecution()
	err := job.Run(&Context{Execution: e, Logger: logger})
	c.Assert(err, IsNil)

	// Verify that quoted arguments are preserved as single arguments
	c.Assert(createdOpts.Spec.TaskTemplate.ContainerSpec.Command, DeepEquals,
		[]string{"/bin/sh", "-c", "echo \"hello world\""})
}

func (s *SuiteRunServiceJob) TestPullImageError(c *C) {
	mock := &mockDockerClient{
		ImagePullFn: func(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
			c.Assert(refStr, Equals, "docker.io/library/private:latest")
			return &mockPullResponseWithError{err: "denied"}, nil
		},
	}

	err := pullImage(mock, "private", context.Background())
	c.Assert(err, ErrorMatches, `error pulling image "private": denied`)
}

func (s *SuiteRunServiceJob) TestFindTaskStatusWaitsWhenNoTasksExist(c *C) {
	mock := &mockDockerClient{
		TaskListFn: func(ctx context.Context, options client.TaskListOptions) (client.TaskListResult, error) {
