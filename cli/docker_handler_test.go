package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	dockerclient "github.com/docker/docker/client"
	"github.com/mcuadros/ofelia/core"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&TestDockerSuit{})

type TestDockerSuit struct{}

// mockCLIDockerClient implements core.DockerClient for CLI tests.
type mockCLIDockerClient struct {
	containers []container.Summary
}

func (m *mockCLIDockerClient) Info(ctx context.Context) (system.Info, error) {
	return system.Info{}, nil
}

func (m *mockCLIDockerClient) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	containers := make([]container.Summary, 0, len(m.containers))
	for _, cont := range m.containers {
		if !options.Filters.MatchKVList("label", cont.Labels) {
			continue
		}
		if !matchesContainerNames(options.Filters, cont.Names) {
			continue
		}
		containers = append(containers, cont)
	}
	return containers, nil
}

func matchesContainerNames(filterArgs filters.Args, names []string) bool {
	if !filterArgs.Contains("name") {
		return true
	}

	for _, name := range names {
		trimmed := strings.TrimPrefix(name, "/")
		if filterArgs.Match("name", name) || filterArgs.Match("name", trimmed) {
			return true
		}
	}
	return false
}

func (m *mockCLIDockerClient) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return container.InspectResponse{}, nil
}

func (m *mockCLIDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	return container.CreateResponse{}, nil
}

func (m *mockCLIDockerClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return nil
}

func (m *mockCLIDockerClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	return nil
}

func (m *mockCLIDockerClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	return nil
}

func (m *mockCLIDockerClient) ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	return io.NopCloser(io.LimitReader(nil, 0)), nil
}

func (m *mockCLIDockerClient) ContainerExecCreate(ctx context.Context, ctr string, config container.ExecOptions) (container.ExecCreateResponse, error) {
	return container.ExecCreateResponse{}, nil
}

func (m *mockCLIDockerClient) ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, nil
}

func (m *mockCLIDockerClient) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return container.ExecInspect{}, nil
}

func (m *mockCLIDockerClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	return nil, nil
}

func (m *mockCLIDockerClient) ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error) {
	return io.NopCloser(io.LimitReader(nil, 0)), nil
}

func (m *mockCLIDockerClient) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Inspect, error) {
	return nil, nil
}

func (m *mockCLIDockerClient) NetworkConnect(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error {
	return nil
}

func (m *mockCLIDockerClient) ServiceCreate(ctx context.Context, service swarm.ServiceSpec, options swarm.ServiceCreateOptions) (swarm.ServiceCreateResponse, error) {
	return swarm.ServiceCreateResponse{}, nil
}

func (m *mockCLIDockerClient) ServiceInspectWithRaw(ctx context.Context, serviceID string, options swarm.ServiceInspectOptions) (swarm.Service, []byte, error) {
	return swarm.Service{}, nil, nil
}

func (m *mockCLIDockerClient) ServiceRemove(ctx context.Context, serviceID string) error {
	return nil
}

func (m *mockCLIDockerClient) TaskList(ctx context.Context, options swarm.TaskListOptions) ([]swarm.Task, error) {
	return nil, nil
}

func newTestDockerHandler(mock core.DockerClient, filters []string) *DockerHandler {
	return &DockerHandler{
		dockerClient:      mock,
		configsFromLabels: true,
		filters:           filters,
		logger:            &TestLogger{},
	}
}

func (s *TestDockerSuit) TestBuildDockerClientReturnsOfficialClient(c *check.C) {
	restore := withDockerEnv(map[string]string{
		"DOCKER_HOST":        "unix:///var/run/docker.sock",
		"DOCKER_TLS_VERIFY":  "",
		"DOCKER_CERT_PATH":   "",
		"DOCKER_API_VERSION": "",
	})
	defer restore()

	dc, err := buildDockerClient()
	c.Assert(err, check.IsNil)
	cli, ok := dc.(*dockerclient.Client)
	c.Assert(ok, check.Equals, true)
	c.Assert(cli.Close(), check.IsNil)
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
	c.Assert(err.Error(), check.Matches, ".*invalid docker filter.*")
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
	c.Assert(err.Error(), check.Matches, ".*test=123.*")
	c.Assert(err.Error(), check.Matches, ".*test-name.*")
}

func (s *TestDockerSuit) TestGetContainerID(c *check.C) {
	tests := []struct {
		content string
		expect  string
	}{
		{
			content: `
206 205 0:29 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw
207 203 0:67 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
208 203 0:72 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k
209 201 254:1 /docker/containers/test123/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/vda1 rw,discard
210 201 254:1 /docker/containers/test123/hostname /etc/hostname rw,relatime - ext4 /dev/vda1 rw,discard
211 201 254:1 /docker/containers/test123/hosts /etc/hosts rw,relatime - ext4 /dev/vda1 rw,discard
85 203 0:70 /0 /dev/console rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
86 202 0:68 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
87 202 0:68 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
88 202 0:68 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
`,
			expect: "test123",
		},
		{
			content: `
206 205 0:29 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw
207 203 0:67 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
208 203 0:72 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k
209 201 254:1 /var/lib/docker/containers/test123/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/vda1 rw,discard
210 201 254:1 /var/lib/docker/containers/test123/hostname /etc/hostname rw,relatime - ext4 /dev/vda1 rw,discard
211 201 254:1 /var/lib/docker/containers/test123/hosts /etc/hosts rw,relatime - ext4 /dev/vda1 rw,discard
85 203 0:70 /0 /dev/console rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
86 202 0:68 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
87 202 0:68 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
88 202 0:68 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
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

func (s *TestDockerSuit) TestBuildFromDockerLabels(c *check.C) {
	labels := map[string]map[string]string{
		"container1": {
			labelPrefix + "." + jobExec + ".myjob.schedule":  "* * * * *",
			labelPrefix + "." + jobExec + ".myjob.command":   "echo hello",
			labelPrefix + "." + jobExec + ".myjob.container": "container1",
		},
	}

	conf := NewConfig(&TestLogger{})
	err := conf.buildFromDockerLabels(labels)
	c.Assert(err, check.IsNil)
	c.Assert(conf.ExecJobs, check.HasLen, 1)

	job, ok := conf.ExecJobs["myjob"]
	c.Assert(ok, check.Equals, true)
	c.Assert(job.Schedule, check.Equals, "* * * * *")
	c.Assert(job.Command, check.Equals, "echo hello")
}

func (s *TestDockerSuit) TestBuildFromDockerLabelsServiceContainer(c *check.C) {
	labels := map[string]map[string]string{
		"ofelia-service": {
			serviceLabel: "true",
			labelPrefix + "." + jobRun + ".myjob.schedule": "* * * * *",
			labelPrefix + "." + jobRun + ".myjob.command":  "echo hello",
			labelPrefix + "." + jobRun + ".myjob.image":    "alpine",
		},
	}

	conf := NewConfig(&TestLogger{})
	err := conf.buildFromDockerLabels(labels)
	c.Assert(err, check.IsNil)
	c.Assert(conf.RunJobs, check.HasLen, 1)

	job, ok := conf.RunJobs["myjob"]
	c.Assert(ok, check.Equals, true)
	c.Assert(job.Image, check.Equals, "alpine")
}

func withDockerEnv(values map[string]string) func() {
	previous := make(map[string]string, len(values))
	present := make(map[string]bool, len(values))
	for key, value := range values {
		previous[key], present[key] = os.LookupEnv(key)
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}

	return func() {
		for key, value := range previous {
			if present[key] {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}
