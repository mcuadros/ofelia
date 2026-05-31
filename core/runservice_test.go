package core

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/swarm"
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
	var createdSpec swarm.ServiceSpec

	mock := &mockDockerClient{
		ImagePullFn: func(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error) {
			return io.NopCloser(io.LimitReader(nil, 0)), nil
		},
		ServiceCreateFn: func(ctx context.Context, service swarm.ServiceSpec, options swarm.ServiceCreateOptions) (swarm.ServiceCreateResponse, error) {
			createdSpec = service
			return swarm.ServiceCreateResponse{ID: "svc-123"}, nil
		},
		ServiceInspectWithRawFn: func(ctx context.Context, serviceID string, options swarm.ServiceInspectOptions) (swarm.Service, []byte, error) {
			return swarm.Service{
				ID: serviceID,
				Meta: swarm.Meta{
					CreatedAt: time.Now(),
				},
			}, nil, nil
		},
		TaskListFn: func(ctx context.Context, options swarm.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{
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
			}, nil
		},
		ServiceRemoveFn: func(ctx context.Context, serviceID string) error {
			c.Assert(serviceID, Equals, "svc-123")
			return nil
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

	c.Assert(createdSpec.TaskTemplate.ContainerSpec.Command, DeepEquals, []string{"echo", "-a", "foo", "bar"})
	c.Assert(createdSpec.TaskTemplate.ContainerSpec.Image, Equals, ServiceImageFixture)
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
