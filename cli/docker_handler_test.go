package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mcuadros/ofelia/core"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&TestDockerSuit{})

type TestDockerSuit struct{}

// mockCLIDockerClient implements core.DockerClient for CLI tests.
type mockCLIDockerClient struct {
	containers []container.Summary
	eventsFn   func(ctx context.Context, options client.EventsListOptions) client.EventsResult
}

func (m *mockCLIDockerClient) Info(ctx context.Context, options client.InfoOptions) (client.SystemInfoResult, error) {
	return client.SystemInfoResult{}, nil
}

func (m *mockCLIDockerClient) Events(ctx context.Context, options client.EventsListOptions) client.EventsResult {
	if m.eventsFn != nil {
		return m.eventsFn(ctx, options)
	}
	messages := make(chan events.Message)
	close(messages)
	errs := make(chan error)
	close(errs)
	return client.EventsResult{
		Messages: messages,
		Err:      errs,
	}
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

	dockerLabels, err := handler.GetDockerLabels(context.Background())
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
	_, err := handler.GetDockerLabels(context.Background())
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*no containers matching filters.*")
}

func (s *TestDockerSuit) TestFilterErrors(c *check.C) {
	mock := &mockCLIDockerClient{
		containers: []container.Summary{},
	}

	handler := newTestDockerHandler(mock, []string{"bad-filter"})
	_, err := handler.GetDockerLabels(context.Background())
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
	_, err := handler.GetDockerLabels(context.Background())
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

func (s *TestDockerSuit) TestIsReloadAction(c *check.C) {
	reloadActions := []events.Action{
		events.ActionCreate,
		events.ActionStart,
		events.ActionRestart,
		events.ActionStop,
		events.ActionKill,
		events.ActionDie,
		events.ActionDestroy,
		events.ActionPause,
		events.ActionUnPause,
		events.ActionRename,
		events.ActionUpdate,
	}
	for _, action := range reloadActions {
		c.Assert(isReloadAction(action), check.Equals, true, check.Commentf("expected %q to trigger reload", action))
	}

	nonReloadActions := []events.Action{
		events.ActionAttach,
		events.ActionDetach,
		events.ActionResize,
		events.ActionTop,
		events.ActionExecStart,
		events.ActionExecCreate,
		events.ActionExecDie,
	}
	for _, action := range nonReloadActions {
		c.Assert(isReloadAction(action), check.Equals, false, check.Commentf("expected %q to NOT trigger reload", action))
	}
}

func (s *TestDockerSuit) TestConsumeEventsTriggersLabelUpdate(c *check.C) {
	mock := &mockCLIDockerClient{
		containers: []container.Summary{
			{
				Names: []string{"/myapp"},
				Labels: map[string]string{
					requiredLabel:                                  "true",
					labelPrefix + "." + jobExec + ".job1.schedule": "* * * * *",
					labelPrefix + "." + jobExec + ".job1.command":  "echo hello",
				},
			},
		},
	}

	updatedCh := make(chan map[string]map[string]string, 1)
	notifier := &mockLabelUpdater{ch: updatedCh}

	handler := &DockerHandler{
		dockerClient:      mock,
		configsFromLabels: true,
		filters:           nil,
		logger:            &TestLogger{},
		notifier:          notifier,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messages := make(chan events.Message, 1)
	errs := make(chan error)

	messages <- events.Message{
		Type:   events.ContainerEventType,
		Action: events.ActionStart,
		Actor:  events.Actor{ID: "abc123", Attributes: map[string]string{"name": "myapp"}},
	}

	result := client.EventsResult{
		Messages: messages,
		Err:      errs,
	}

	go func() {
		handler.consumeEvents(ctx, result)
	}()

	select {
	case labels := <-updatedCh:
		c.Assert(labels["myapp"], check.NotNil)
	case <-time.After(2 * time.Second):
		c.Fatal("timed out waiting for label update")
	}
}

func (s *TestDockerSuit) TestConsumeEventsDebounces(c *check.C) {
	mock := &mockCLIDockerClient{
		containers: []container.Summary{
			{
				Names: []string{"/myapp"},
				Labels: map[string]string{
					requiredLabel:                                  "true",
					labelPrefix + "." + jobExec + ".job1.schedule": "* * * * *",
					labelPrefix + "." + jobExec + ".job1.command":  "echo hello",
				},
			},
		},
	}

	updatedCh := make(chan map[string]map[string]string, 10)
	notifier := &mockLabelUpdater{ch: updatedCh}

	handler := &DockerHandler{
		dockerClient:      mock,
		configsFromLabels: true,
		filters:           nil,
		logger:            &TestLogger{},
		notifier:          notifier,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messages := make(chan events.Message, 10)
	errs := make(chan error)

	for i := 0; i < 5; i++ {
		messages <- events.Message{
			Type:   events.ContainerEventType,
			Action: events.ActionStart,
			Actor:  events.Actor{ID: "abc123", Attributes: map[string]string{"name": "myapp"}},
		}
	}

	result := client.EventsResult{
		Messages: messages,
		Err:      errs,
	}

	go func() {
		handler.consumeEvents(ctx, result)
	}()

	select {
	case <-updatedCh:
	case <-time.After(2 * time.Second):
		c.Fatal("timed out waiting for label update")
	}

	// Allow some time for any extra calls to arrive
	time.Sleep(100 * time.Millisecond)

	select {
	case <-updatedCh:
		c.Fatal("debounce failed: received more than one update for rapid events")
	default:
	}
}

func (s *TestDockerSuit) TestConsumeEventsIgnoresNonReloadActions(c *check.C) {
	mock := &mockCLIDockerClient{
		containers: []container.Summary{
			{
				Names:  []string{"/myapp"},
				Labels: map[string]string{requiredLabel: "true"},
			},
		},
	}

	updatedCh := make(chan map[string]map[string]string, 1)
	notifier := &mockLabelUpdater{ch: updatedCh}

	handler := &DockerHandler{
		dockerClient:      mock,
		configsFromLabels: true,
		filters:           nil,
		logger:            &TestLogger{},
		notifier:          notifier,
	}

	messages := make(chan events.Message, 1)
	errs := make(chan error, 1)

	messages <- events.Message{
		Type:   events.ContainerEventType,
		Action: events.ActionAttach,
		Actor:  events.Actor{ID: "abc123", Attributes: map[string]string{"name": "myapp"}},
	}
	close(messages)

	result := client.EventsResult{
		Messages: messages,
		Err:      errs,
	}

	_ = handler.consumeEvents(context.Background(), result)

	select {
	case <-updatedCh:
		c.Fatal("should not have triggered a label update for attach event")
	default:
	}
}

func (s *TestDockerSuit) TestConsumeEventsExitsOnError(c *check.C) {
	mock := &mockCLIDockerClient{}
	handler := &DockerHandler{
		dockerClient:      mock,
		configsFromLabels: true,
		logger:            &TestLogger{},
		notifier:          &mockLabelUpdater{ch: make(chan map[string]map[string]string, 1)},
	}

	messages := make(chan events.Message)
	errs := make(chan error, 1)
	errs <- errors.New("connection lost")

	result := client.EventsResult{
		Messages: messages,
		Err:      errs,
	}

	err := handler.consumeEvents(context.Background(), result)
	c.Assert(err, check.ErrorMatches, "connection lost")
}

func (s *TestDockerSuit) TestWatchExitsOnContextCancel(c *check.C) {
	ctx, cancel := context.WithCancel(context.Background())

	messages := make(chan events.Message)
	errs := make(chan error, 1)

	mock := &mockCLIDockerClient{}
	mock.eventsFn = func(_ context.Context, _ client.EventsListOptions) client.EventsResult {
		return client.EventsResult{Messages: messages, Err: errs}
	}

	handler := &DockerHandler{
		dockerClient:      mock,
		configsFromLabels: true,
		logger:            &TestLogger{},
		notifier:          &mockLabelUpdater{ch: make(chan map[string]map[string]string, 1)},
	}

	done := make(chan struct{})
	go func() {
		handler.watch(ctx)
		close(done)
	}()

	cancel()
	errs <- context.Canceled

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		c.Fatal("watch did not exit after context cancellation")
	}
}

type mockLabelUpdater struct {
	ch chan map[string]map[string]string
}

func (m *mockLabelUpdater) dockerLabelsUpdate(labels map[string]map[string]string) {
	select {
	case m.ch <- labels:
	default:
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
