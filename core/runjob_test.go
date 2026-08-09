package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"
	"sync/atomic"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
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
		expectedEntrypoint []string
	}{
		{nil, nil},
		{&overridenEntrypoint, []string{"/bin/bash", "-c"}},
		{&emptyEntrypoint, []string{}},
	}

	for _, tc := range testCases {
		var createdOpts client.ContainerCreateOptions
		var inspectCount atomic.Int32
		var started bool
		var removed bool

		mock := &mockDockerClient{
			ImageListFn: func(ctx context.Context, options client.ImageListOptions) (client.ImageListResult, error) {
				return client.ImageListResult{Items: []image.Summary{{}}}, nil
			},
			ContainerCreateFn: func(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
				createdOpts = options
				c.Assert(options.HostConfig.Binds, DeepEquals, []string{"/test/tmp:/test/tmp:ro", "/test/tmp:/test/tmp:rw"})
				return client.ContainerCreateResult{ID: "cnt-123"}, nil
			},
			NetworkListFn: func(ctx context.Context, options client.NetworkListOptions) (client.NetworkListResult, error) {
				return client.NetworkListResult{Items: []network.Summary{{Network: network.Network{ID: "net-123", Name: "foo"}}}}, nil
			},
			NetworkConnectFn: func(ctx context.Context, networkID string, options client.NetworkConnectOptions) (client.NetworkConnectResult, error) {
				c.Assert(networkID, Equals, "net-123")
				c.Assert(options.Container, Equals, "cnt-123")
				return client.NetworkConnectResult{}, nil
			},
			ContainerStartFn: func(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error) {
				c.Assert(containerID, Equals, "cnt-123")
				started = true
				return client.ContainerStartResult{}, nil
			},
			ContainerInspectFn: func(ctx context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
				c.Assert(containerID, Equals, "cnt-123")
				count := inspectCount.Add(1)
				if count <= 2 {
					return client.ContainerInspectResult{
						Container: container.InspectResponse{
							State: &container.State{Running: true},
						},
					}, nil
				}
				return client.ContainerInspectResult{
					Container: container.InspectResponse{
						State: &container.State{Running: false, ExitCode: 0},
					},
				}, nil
			},
			ContainerLogsFn: func(ctx context.Context, containerID string, options client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
				c.Assert(containerID, Equals, "cnt-123")
				c.Assert(options.ShowStdout, Equals, true)
				c.Assert(options.ShowStderr, Equals, true)
				c.Assert(options.Since, Not(Equals), "")
				return io.NopCloser(bytes.NewReader([]byte("log output"))), nil
			},
			ContainerRemoveFn: func(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
				c.Assert(containerID, Equals, "cnt-123")
				removed = true
				return client.ContainerRemoveResult{}, nil
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

		c.Assert(createdOpts.Config.Entrypoint, DeepEquals, tc.expectedEntrypoint)
		c.Assert(createdOpts.Config.Cmd, DeepEquals, []string{"echo", "-a", "foo bar"})
		c.Assert(createdOpts.Config.User, Equals, "foo")
		c.Assert(createdOpts.Config.Image, Equals, ImageFixture)
		c.Assert([]string(createdOpts.Config.Env), DeepEquals, job.Environment)
		c.Assert(started, Equals, true)
		c.Assert(removed, Equals, true)
	}
}

func (s *SuiteRunJob) TestRunExistingContainer(c *C) {
	var inspectCount atomic.Int32

	mock := &mockDockerClient{
		ContainerInspectFn: func(ctx context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
			count := inspectCount.Add(1)
			if count == 1 {
				return client.ContainerInspectResult{
					Container: container.InspectResponse{
						ID:    "existing-cnt",
						State: &container.State{Running: true},
					},
				}, nil
			}
			if count <= 3 {
				return client.ContainerInspectResult{
					Container: container.InspectResponse{
						State: &container.State{Running: true},
					},
				}, nil
			}
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					State: &container.State{Running: false, ExitCode: 0},
				},
			}, nil
		},
		ContainerLogsFn: func(ctx context.Context, containerID string, options client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
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

func (s *SuiteRunJob) TestRunNamedContainer(c *C) {
	testCases := []struct {
		description    string
		containerFound bool
		delete         string
		expectCreated  bool
		expectRemoved  bool
		expectStartID  string
	}{
		{
			description:    "first run: container not found, creates with name, keeps it (delete=false)",
			containerFound: false,
			delete:         "false",
			expectCreated:  true,
			expectRemoved:  false,
			expectStartID:  "new-cnt",
		},
		{
			description:    "subsequent run: container exists, reuses it (delete=false)",
			containerFound: true,
			delete:         "false",
			expectCreated:  false,
			expectRemoved:  false,
			expectStartID:  "existing-cnt",
		},
		{
			description:    "container not found, creates with name, removes after run (delete=true)",
			containerFound: false,
			delete:         "true",
			expectCreated:  true,
			expectRemoved:  true,
			expectStartID:  "new-cnt",
		},
	}

	for _, tc := range testCases {
		var created bool
		var removed bool
		var startedID string
		var inspectCount atomic.Int32

		mock := &mockDockerClient{
			ContainerInspectFn: func(ctx context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
				count := inspectCount.Add(1)
				if count == 1 {
					if !tc.containerFound {
						return client.ContainerInspectResult{}, errdefs.ErrNotFound
					}
					return client.ContainerInspectResult{
						Container: container.InspectResponse{
							ID:    "existing-cnt",
							State: &container.State{Running: false, ExitCode: 0},
						},
					}, nil
				}
				if count <= 3 {
					return client.ContainerInspectResult{
						Container: container.InspectResponse{
							State: &container.State{Running: true},
						},
					}, nil
				}
				return client.ContainerInspectResult{
					Container: container.InspectResponse{
						State: &container.State{Running: false, ExitCode: 0},
					},
				}, nil
			},
			ImageListFn: func(ctx context.Context, options client.ImageListOptions) (client.ImageListResult, error) {
				return client.ImageListResult{Items: []image.Summary{{}}}, nil
			},
			ContainerCreateFn: func(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
				created = true
				c.Assert(options.Name, Equals, "my-named-container")
				return client.ContainerCreateResult{ID: "new-cnt"}, nil
			},
			ContainerStartFn: func(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error) {
				startedID = containerID
				return client.ContainerStartResult{}, nil
			},
			ContainerLogsFn: func(ctx context.Context, containerID string, options client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
				return io.NopCloser(bytes.NewReader(nil)), nil
			},
			ContainerRemoveFn: func(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
				removed = true
				return client.ContainerRemoveResult{}, nil
			},
		}

		job := &RunJob{Client: mock}
		job.Image = "alpine:latest"
		job.Container = "my-named-container"
		job.Command = "echo hello"
		job.Delete = tc.delete
		job.Pull = "false"
		job.Name = "test"

		ctx := &Context{}
		ctx.Execution = NewExecution()
		ctx.Logger = NewSlogLogger(io.Discard)
		ctx.Job = job

		err := job.Run(ctx)
		c.Assert(err, IsNil, Commentf(tc.description))
		c.Assert(created, Equals, tc.expectCreated, Commentf(tc.description))
		c.Assert(removed, Equals, tc.expectRemoved, Commentf(tc.description))
		c.Assert(startedID, Equals, tc.expectStartID, Commentf(tc.description))
	}
}

func (s *SuiteRunJob) TestPullImageError(c *C) {
	mock := &mockDockerClient{
		ImagePullFn: func(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
			c.Assert(refStr, Equals, "docker.io/library/private:latest")
			return &mockPullResponseWithError{err: "denied"}, nil
		},
	}

	err := pullImage(mock, "private", context.Background())
	c.Assert(err, ErrorMatches, `error pulling image "private": denied`)
}

func (s *SuiteRunJob) TestBuildPullImageOptionsBareImage(c *C) {
	ref, _ := buildPullOptions("foo")
	c.Assert(ref, Equals, "docker.io/library/foo:latest")
}

func (s *SuiteRunJob) TestBuildPullImageOptionsVersion(c *C) {
	ref, _ := buildPullOptions("foo:qux")
	c.Assert(ref, Equals, "docker.io/library/foo:qux")
}

func (s *SuiteRunJob) TestBuildPullImageOptionsRegistry(c *C) {
	ref, _ := buildPullOptions("quay.io/srcd/rest:qux")
	c.Assert(ref, Equals, "quay.io/srcd/rest:qux")
}

// mockPullResponseWithError simulates a pull that returns an error via Wait.
type mockPullResponseWithError struct {
	err string
}

func (m *mockPullResponseWithError) Read(p []byte) (int, error) { return 0, io.EOF }
func (m *mockPullResponseWithError) Close() error               { return nil }
func (m *mockPullResponseWithError) Wait(_ context.Context) error {
	return fmt.Errorf("%s", m.err)
}
func (m *mockPullResponseWithError) JSONMessages(_ context.Context) iter.Seq2[jsonstream.Message, error] {
	return func(yield func(jsonstream.Message, error) bool) {}
}
