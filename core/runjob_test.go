package core

import (
	"bytes"
	"context"
	"io"
	"sync/atomic"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	. "gopkg.in/check.v1"
)

const ImageFixture = "test-image"

type SuiteRunJob struct{}

var _ = Suite(&SuiteRunJob{})

func (s *SuiteRunJob) TestRun(c *C) {
	overridenEntrypoint := "/bin/bash -c"
	emptyEntrypoint := ""
	testCases := []struct {
		entrypoint         *string
		expectedEntrypoint strslice.StrSlice
	}{
		{nil, nil},
		{&overridenEntrypoint, strslice.StrSlice{"/bin/bash", "-c"}},
		{&emptyEntrypoint, strslice.StrSlice{}},
	}

	for _, tc := range testCases {
		var createdConfig *container.Config
		var inspectCount atomic.Int32
		var started bool
		var removed bool

		mock := &mockDockerClient{
			ImageListFn: func(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
				return []image.Summary{{}}, nil
			},
			ContainerCreateFn: func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
				createdConfig = config
				c.Assert(hostConfig.Binds, DeepEquals, []string{"/test/tmp:/test/tmp:ro", "/test/tmp:/test/tmp:rw"})
				return container.CreateResponse{ID: "cnt-123"}, nil
			},
			NetworkListFn: func(ctx context.Context, options network.ListOptions) ([]network.Inspect, error) {
				return []network.Inspect{{ID: "net-123"}}, nil
			},
			NetworkConnectFn: func(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error {
				c.Assert(networkID, Equals, "net-123")
				c.Assert(containerID, Equals, "cnt-123")
				return nil
			},
			ContainerStartFn: func(ctx context.Context, containerID string, options container.StartOptions) error {
				c.Assert(containerID, Equals, "cnt-123")
				started = true
				return nil
			},
			ContainerInspectFn: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
				c.Assert(containerID, Equals, "cnt-123")
				count := inspectCount.Add(1)
				if count <= 2 {
					return container.InspectResponse{
						ContainerJSONBase: &container.ContainerJSONBase{
							State: &container.State{Running: true},
						},
					}, nil
				}
				return container.InspectResponse{
					ContainerJSONBase: &container.ContainerJSONBase{
						State: &container.State{Running: false, ExitCode: 0},
					},
				}, nil
			},
			ContainerLogsFn: func(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
				c.Assert(containerID, Equals, "cnt-123")
				c.Assert(options.ShowStdout, Equals, true)
				c.Assert(options.ShowStderr, Equals, true)
				c.Assert(options.Since, Not(Equals), "")
				return io.NopCloser(bytes.NewReader([]byte("log output"))), nil
			},
			ContainerRemoveFn: func(ctx context.Context, containerID string, options container.RemoveOptions) error {
				c.Assert(containerID, Equals, "cnt-123")
				removed = true
				return nil
			},
		}

		job := &RunJob{Client: mock}
		job.Image = ImageFixture
		job.Command = `echo -a "foo bar"`
		job.User = "foo"
		job.TTY = true
		job.Delete = "true"
		job.Pull = "false"
		job.Network = "foo"
		job.Hostname = "test-host"
		job.Name = "test"
		job.Environment = []string{"test_Key1=value1", "test_Key2=value2"}
		job.Volume = []string{"/test/tmp:/test/tmp:ro", "/test/tmp:/test/tmp:rw"}
		job.Entrypoint = tc.entrypoint

		ctx := &Context{}
		ctx.Execution = NewExecution()
		ctx.Logger = NewSlogLogger(io.Discard)
		ctx.Job = job

		err := job.Run(ctx)
		c.Assert(err, IsNil)

		c.Assert(createdConfig.Entrypoint, DeepEquals, tc.expectedEntrypoint)
		c.Assert(createdConfig.Cmd, DeepEquals, strslice.StrSlice{"echo", "-a", "foo bar"})
		c.Assert(createdConfig.User, Equals, "foo")
		c.Assert(createdConfig.Image, Equals, ImageFixture)
		c.Assert([]string(createdConfig.Env), DeepEquals, job.Environment)
		c.Assert(started, Equals, true)
		c.Assert(removed, Equals, true)
	}
}

func (s *SuiteRunJob) TestRunExistingContainer(c *C) {
	var inspectCount atomic.Int32

	mock := &mockDockerClient{
		ContainerInspectFn: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
			count := inspectCount.Add(1)
			if count == 1 {
				return container.InspectResponse{
					ContainerJSONBase: &container.ContainerJSONBase{
						ID:    "existing-cnt",
						State: &container.State{Running: true},
					},
				}, nil
			}
			if count <= 3 {
				return container.InspectResponse{
					ContainerJSONBase: &container.ContainerJSONBase{
						State: &container.State{Running: true},
					},
				}, nil
			}
			return container.InspectResponse{
				ContainerJSONBase: &container.ContainerJSONBase{
					State: &container.State{Running: false, ExitCode: 0},
				},
			}, nil
		},
		ContainerLogsFn: func(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(nil)), nil
		},
	}

	job := &RunJob{Client: mock}
	job.Container = "my-container"
	job.Command = "echo hello"
	job.Name = "test"

	ctx := &Context{}
	ctx.Execution = NewExecution()
	ctx.Logger = NewSlogLogger(io.Discard)
	ctx.Job = job

	err := job.Run(ctx)
	c.Assert(err, IsNil)
}

func (s *SuiteRunJob) TestPullImageReturnsStreamError(c *C) {
	mock := &mockDockerClient{
		ImagePullFn: func(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error) {
			c.Assert(refStr, Equals, "private:latest")
			return io.NopCloser(bytes.NewReader([]byte(`{"errorDetail":{"message":"denied"}}`))), nil
		},
	}

	job := &RunJob{Client: mock}
	job.Image = "private"

	err := job.pullImage()
	c.Assert(err, ErrorMatches, `error pulling image "private": denied`)
}

func (s *SuiteRunJob) TestBuildPullImageOptionsBareImage(c *C) {
	ref, _ := buildPullOptions("foo")
	c.Assert(ref, Equals, "foo:latest")
}

func (s *SuiteRunJob) TestBuildPullImageOptionsVersion(c *C) {
	ref, _ := buildPullOptions("foo:qux")
	c.Assert(ref, Equals, "foo:qux")
}

func (s *SuiteRunJob) TestBuildPullImageOptionsRegistry(c *C) {
	ref, _ := buildPullOptions("quay.io/srcd/rest:qux")
	c.Assert(ref, Equals, "quay.io/srcd/rest:qux")
}
