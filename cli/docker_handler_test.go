package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/mcuadros/ofelia/core"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&TestDockerSuit{})

type TestDockerSuit struct{}

// mockCLIDockerClient implements core.DockerClient for CLI tests.
type mockCLIDockerClient struct {
	containers []container.Summary
}

func (m *mockCLIDockerClient) Info(ctx context.Context, options client.InfoOptions) (client.SystemInfoResult, error) {
	return client.SystemInfoResult{}, nil
}

func (m *mockCLIDockerClient) ContainerList(ctx context.Context, options client.ContainerListOptions) (client.ContainerListResult, error) {
	var matched []container.Summary
	for _, cont := range m.containers {
		if matchesFilters(options.Filters, cont) {
			matched = append(matched, cont)
		}
	}
	return client.ContainerListResult{Items: matched}, nil
}

func matchesFilters(filters client.Filters, cont container.Summary) bool {
	if filters == nil {
		return true
	}
	for term, values := range filters {
		switch term {
		case "label":
			if !matchesLabelFilter(values, cont.Labels) {
				return false
			}
		case "name":
			if !matchesNameFilter(values, cont.Names) {
				return false
			}
		}
	}
	return true
}

func matchesLabelFilter(required map[string]bool, labels map[string]string) bool {
	for req := range required {
		parts := strings.SplitN(req, "=", 2)
		key := parts[0]
		val, exists := labels[key]
		if !exists {
			return false
		}
		if len(parts) == 2 && val != parts[1] {
			return false
		}
	}
	return true
}

func matchesNameFilter(names map[string]bool, containerNames []string) bool {
	for name := range names {
		for _, cn := range containerNames {
			trimmed := strings.TrimPrefix(cn, "/")
			if cn == name || trimmed == name {
				return true
			}
		}
	}
	return false
}

func (m *mockCLIDockerClient) ContainerInspect(ctx context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{}, nil
}

func (m *mockCLIDockerClient) ContainerCreate(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	return client.ContainerCreateResult{}, nil
}

func (m *mockCLIDockerClient) ContainerStart(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error) {
	return client.ContainerStartResult{}, nil
}

func (m *mockCLIDockerClient) ContainerStop(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error) {
	return client.ContainerStopResult{}, nil
}

func (m *mockCLIDockerClient) ContainerRemove(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	return client.ContainerRemoveResult{}, nil
}

func (m *mockCLIDockerClient) ContainerLogs(ctx context.Context, containerID string, options client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
	return nopReadCloser{}, nil
}

func (m *mockCLIDockerClient) ExecCreate(ctx context.Context, containerID string, options client.ExecCreateOptions) (client.ExecCreateResult, error) {
	return client.ExecCreateResult{}, nil
}

func (m *mockCLIDockerClient) ExecAttach(ctx context.Context, execID string, options client.ExecAttachOptions) (client.ExecAttachResult, error) {
	return client.ExecAttachResult{}, nil
}

func (m *mockCLIDockerClient) ExecInspect(ctx context.Context, execID string, options client.ExecInspectOptions) (client.ExecInspectResult, error) {
	return client.ExecInspectResult{}, nil
}

func (m *mockCLIDockerClient) ImageList(ctx context.Context, options client.ImageListOptions) (client.ImageListResult, error) {
	return client.ImageListResult{}, nil
}

func (m *mockCLIDockerClient) ImagePull(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
	return nil, nil
}

func (m *mockCLIDockerClient) NetworkList(ctx context.Context, options client.NetworkListOptions) (client.NetworkListResult, error) {
	return client.NetworkListResult{}, nil
}

func (m *mockCLIDockerClient) NetworkConnect(ctx context.Context, networkID string, options client.NetworkConnectOptions) (client.NetworkConnectResult, error) {
	return client.NetworkConnectResult{}, nil
}

func (m *mockCLIDockerClient) ServiceCreate(ctx context.Context, options client.ServiceCreateOptions) (client.ServiceCreateResult, error) {
	return client.ServiceCreateResult{}, nil
}

func (m *mockCLIDockerClient) ServiceInspect(ctx context.Context, serviceID string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
	return client.ServiceInspectResult{}, nil
}

func (m *mockCLIDockerClient) ServiceRemove(ctx context.Context, serviceID string, options client.ServiceRemoveOptions) (client.ServiceRemoveResult, error) {
	return client.ServiceRemoveResult{}, nil
}

func (m *mockCLIDockerClient) TaskList(ctx context.Context, options client.TaskListOptions) (client.TaskListResult, error) {
	return client.TaskListResult{}, nil
}

type nopReadCloser struct{}

func (nopReadCloser) Read([]byte) (int, error) { return 0, io.EOF }
func (nopReadCloser) Close() error             { return nil }

func newTestDockerHandler(mock core.DockerClient, filters []string) *DockerHandler {
	return &DockerHandler{
		dockerClient:      mock,
		configsFromLabels: true,
		filters:           filters,
		logger:            &TestLogger{},
	}
}

func (s *TestDockerSuit) TestBuildDockerClientReturnsMobyClient(c *check.C) {
	restore := withDockerEnv(map[string]string{
		"DOCKER_HOST":        "unix:///var/run/docker.sock",
		"DOCKER_TLS_VERIFY":  "",
		"DOCKER_CERT_PATH":   "",
		"DOCKER_API_VERSION": "",
	})
	defer restore()

	dc, err := buildDockerClient()
	c.Assert(err, check.IsNil)
	c.Assert(dc, check.NotNil)
}

func (s *TestDockerSuit) TestDockerHandlerAccessors(c *check.C) {
	mock := &mockCLIDockerClient{}
	handler := newTestDockerHandler(mock, nil)

	c.Assert(handler.ConfigFromLabelsEnabled(), check.Equals, true)
	c.Assert(handler.GetInternalDockerClient(), check.Equals, mock)
}

func (s *TestDockerSuit) TestNewDockerHandlerRejectsFiltersWithoutDockerLabels(c *check.C) {
	_, err := NewDockerHandler(NewConfig(&TestLogger{}), []string{"name=app"}, false, &TestLogger{})
	c.Assert(err, check.ErrorMatches, "docker filters can only be provided together with '--docker' flag")
}

func (s *TestDockerSuit) TestLabelsFilterJobsCount(c *check.C) {
	mock := &mockCLIDockerClient{
		containers: []container.Summary{
			{
				Names: []string{"/container1"},
				Labels: map[string]string{
					requiredLabel:       "true",
					"test_filter_label": "yesssss",
					labelPrefix + "." + jobExec + ".job2.schedule": "* * * * *",
					labelPrefix + "." + jobExec + ".job2.command":  "command2",
				},
			},
			{
				Names: []string{"/container2"},
				Labels: map[string]string{
					requiredLabel: "true",
					labelPrefix + "." + jobExec + ".job3.schedule": "* * * * *",
					labelPrefix + "." + jobExec + ".job3.command":  "command3",
				},
			},
		},
	}

	handler := newTestDockerHandler(mock, []string{"label=test_filter_label=yesssss"})

	mockLogger := &TestLogger{}
	conf := &Config{
		sh:            core.NewScheduler(mockLogger),
		dockerHandler: handler,
		logger:        mockLogger,
	}
	conf.ExecJobs = make(map[string]*ExecJobConfig)
	conf.RunJobs = make(map[string]*RunJobConfig)
	conf.ServiceJobs = make(map[string]*RunServiceConfig)
	conf.LocalJobs = make(map[string]*LocalJobConfig)

	dockerLabels, err := handler.GetDockerLabels()
	c.Assert(err, check.IsNil)

	err = conf.buildFromDockerLabels(dockerLabels)
	c.Assert(err, check.IsNil)
	c.Assert(len(conf.ExecJobs), check.Equals, 1)
	c.Assert(conf.ExecJobs["job2"].Command, check.Equals, "command2")
}

func (s *TestDockerSuit) TestGetDockerLabelsNoContainers(c *check.C) {
	mock := &mockCLIDockerClient{
		containers: []container.Summary{},
	}

	handler := newTestDockerHandler(mock, nil)
	_, err := handler.GetDockerLabels()
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*no containers matching filters.*")
}

func (s *TestDockerSuit) TestFilterErrors(c *check.C) {
	mock := &mockCLIDockerClient{
		containers: []container.Summary{},
	}

	handler := newTestDockerHandler(mock, []string{"bad-filter"})
	_, err := handler.GetDockerLabels()
	c.Assert(err, check.NotNil)
	c.Assert(errors.Is(err, errInvalidDockerFilter), check.Equals, true)
}

func (s *TestDockerSuit) TestFilterErrorsNoMatchingContainers(c *check.C) {
	mock := &mockCLIDockerClient{
		containers: []container.Summary{
			{
				Names: []string{"/container1"},
				Labels: map[string]string{
					requiredLabel: "true",
				},
			},
		},
	}

	handler := newTestDockerHandler(mock, []string{"label=test=123", "name=test-name"})
	_, err := handler.GetDockerLabels()
	c.Assert(errors.Is(err, errNoContainersMatchingFilters), check.Equals, true)
}

func (s *TestDockerSuit) TestGetContainerID(c *check.C) {
	tests := []struct {
		content string
		expect  string
	}{
		{
			content: `
206 205 0:29 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw
209 201 254:1 /docker/containers/test123/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/vda1 rw,discard
`,
			expect: "test123",
		},
		{
			content: `
206 205 0:29 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw
209 201 254:1 /var/lib/docker/containers/test123/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/vda1 rw,discard
`,
			expect: "test123",
		},
	}

	for _, tt := range tests {
		tmpFile, _ := os.CreateTemp(os.TempDir(), "mountinfo")
		tmpFile.WriteString(tt.content)
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		id, err := getContainerID(tmpFile.Name())
		c.Assert(err, check.IsNil)
		c.Assert(id, check.Equals, tt.expect)
	}
}

func withDockerEnv(envs map[string]string) func() {
	type envEntry struct {
		value string
		isSet bool
	}
	old := make(map[string]envEntry)
	for k, v := range envs {
		val, ok := os.LookupEnv(k)
		old[k] = envEntry{value: val, isSet: ok}
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
	return func() {
		for k, e := range old {
			if !e.isSet {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, e.value)
			}
		}
	}
}
