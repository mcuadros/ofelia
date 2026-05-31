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
}

func (s *SuiteRunServiceJob) TestBuildPullImageOptionsBareImage(c *C) {
	ref, _ := buildPullOptions("foo")
	c.Assert(ref, Equals, "foo:latest")
}

func (s *SuiteRunServiceJob) TestBuildPullImageOptionsVersion(c *C) {
	ref, _ := buildPullOptions("foo:qux")
	c.Assert(ref, Equals, "foo:qux")
}

func (s *SuiteRunServiceJob) TestBuildPullImageOptionsRegistry(c *C) {
	ref, _ := buildPullOptions("quay.io/srcd/rest:qux")
	c.Assert(ref, Equals, "quay.io/srcd/rest:qux")
}
