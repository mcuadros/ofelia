package core

import (
	"context"
	"io"
	"iter"

	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/client"
)

type mockDockerClient struct {
	InfoFn             func(ctx context.Context, options client.InfoOptions) (client.SystemInfoResult, error)
	ContainerListFn    func(ctx context.Context, options client.ContainerListOptions) (client.ContainerListResult, error)
	ContainerInspectFn func(ctx context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error)
	ContainerCreateFn  func(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error)
	ContainerStartFn   func(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error)
	ContainerStopFn    func(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error)
	ContainerRemoveFn  func(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error)
	ContainerLogsFn    func(ctx context.Context, containerID string, options client.ContainerLogsOptions) (client.ContainerLogsResult, error)
	ExecCreateFn       func(ctx context.Context, containerID string, options client.ExecCreateOptions) (client.ExecCreateResult, error)
	ExecAttachFn       func(ctx context.Context, execID string, options client.ExecAttachOptions) (client.ExecAttachResult, error)
	ExecInspectFn      func(ctx context.Context, execID string, options client.ExecInspectOptions) (client.ExecInspectResult, error)
	ImageListFn        func(ctx context.Context, options client.ImageListOptions) (client.ImageListResult, error)
	ImagePullFn        func(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error)
	NetworkListFn      func(ctx context.Context, options client.NetworkListOptions) (client.NetworkListResult, error)
	NetworkConnectFn   func(ctx context.Context, networkID string, options client.NetworkConnectOptions) (client.NetworkConnectResult, error)
	ServiceCreateFn    func(ctx context.Context, options client.ServiceCreateOptions) (client.ServiceCreateResult, error)
	ServiceInspectFn   func(ctx context.Context, serviceID string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error)
	ServiceRemoveFn    func(ctx context.Context, serviceID string, options client.ServiceRemoveOptions) (client.ServiceRemoveResult, error)
	TaskListFn         func(ctx context.Context, options client.TaskListOptions) (client.TaskListResult, error)
}

func (m *mockDockerClient) Info(ctx context.Context, options client.InfoOptions) (client.SystemInfoResult, error) {
	if m.InfoFn != nil {
		return m.InfoFn(ctx, options)
	}
	return client.SystemInfoResult{}, nil
}

func (m *mockDockerClient) ContainerList(ctx context.Context, options client.ContainerListOptions) (client.ContainerListResult, error) {
	if m.ContainerListFn != nil {
		return m.ContainerListFn(ctx, options)
	}
	return client.ContainerListResult{}, nil
}

func (m *mockDockerClient) ContainerInspect(ctx context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	if m.ContainerInspectFn != nil {
		return m.ContainerInspectFn(ctx, containerID, options)
	}
	return client.ContainerInspectResult{}, nil
}

func (m *mockDockerClient) ContainerCreate(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	if m.ContainerCreateFn != nil {
		return m.ContainerCreateFn(ctx, options)
	}
	return client.ContainerCreateResult{ID: "mock-container-id"}, nil
}

func (m *mockDockerClient) ContainerStart(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error) {
	if m.ContainerStartFn != nil {
		return m.ContainerStartFn(ctx, containerID, options)
	}
	return client.ContainerStartResult{}, nil
}

func (m *mockDockerClient) ContainerStop(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error) {
	if m.ContainerStopFn != nil {
		return m.ContainerStopFn(ctx, containerID, options)
	}
	return client.ContainerStopResult{}, nil
}

func (m *mockDockerClient) ContainerRemove(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	if m.ContainerRemoveFn != nil {
		return m.ContainerRemoveFn(ctx, containerID, options)
	}
	return client.ContainerRemoveResult{}, nil
}

func (m *mockDockerClient) ContainerLogs(ctx context.Context, containerID string, options client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
	if m.ContainerLogsFn != nil {
		return m.ContainerLogsFn(ctx, containerID, options)
	}
	return nopLogsResult{}, nil
}

func (m *mockDockerClient) ExecCreate(ctx context.Context, containerID string, options client.ExecCreateOptions) (client.ExecCreateResult, error) {
	if m.ExecCreateFn != nil {
		return m.ExecCreateFn(ctx, containerID, options)
	}
	return client.ExecCreateResult{ID: "mock-exec-id"}, nil
}

func (m *mockDockerClient) ExecAttach(ctx context.Context, execID string, options client.ExecAttachOptions) (client.ExecAttachResult, error) {
	if m.ExecAttachFn != nil {
		return m.ExecAttachFn(ctx, execID, options)
	}
	return client.ExecAttachResult{HijackedResponse: client.NewHijackedResponse(nil, "")}, nil
}

func (m *mockDockerClient) ExecInspect(ctx context.Context, execID string, options client.ExecInspectOptions) (client.ExecInspectResult, error) {
	if m.ExecInspectFn != nil {
		return m.ExecInspectFn(ctx, execID, options)
	}
	return client.ExecInspectResult{ExitCode: 0}, nil
}

func (m *mockDockerClient) ImageList(ctx context.Context, options client.ImageListOptions) (client.ImageListResult, error) {
	if m.ImageListFn != nil {
		return m.ImageListFn(ctx, options)
	}
	return client.ImageListResult{}, nil
}

func (m *mockDockerClient) ImagePull(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
	if m.ImagePullFn != nil {
		return m.ImagePullFn(ctx, refStr, options)
	}
	return &mockPullResponse{}, nil
}

func (m *mockDockerClient) NetworkList(ctx context.Context, options client.NetworkListOptions) (client.NetworkListResult, error) {
	if m.NetworkListFn != nil {
		return m.NetworkListFn(ctx, options)
	}
	return client.NetworkListResult{}, nil
}

func (m *mockDockerClient) NetworkConnect(ctx context.Context, networkID string, options client.NetworkConnectOptions) (client.NetworkConnectResult, error) {
	if m.NetworkConnectFn != nil {
		return m.NetworkConnectFn(ctx, networkID, options)
	}
	return client.NetworkConnectResult{}, nil
}

func (m *mockDockerClient) ServiceCreate(ctx context.Context, options client.ServiceCreateOptions) (client.ServiceCreateResult, error) {
	if m.ServiceCreateFn != nil {
		return m.ServiceCreateFn(ctx, options)
	}
	return client.ServiceCreateResult{ID: "mock-service-id"}, nil
}

func (m *mockDockerClient) ServiceInspect(ctx context.Context, serviceID string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
	if m.ServiceInspectFn != nil {
		return m.ServiceInspectFn(ctx, serviceID, options)
	}
	return client.ServiceInspectResult{}, nil
}

func (m *mockDockerClient) ServiceRemove(ctx context.Context, serviceID string, options client.ServiceRemoveOptions) (client.ServiceRemoveResult, error) {
	if m.ServiceRemoveFn != nil {
		return m.ServiceRemoveFn(ctx, serviceID, options)
	}
	return client.ServiceRemoveResult{}, nil
}

func (m *mockDockerClient) TaskList(ctx context.Context, options client.TaskListOptions) (client.TaskListResult, error) {
	if m.TaskListFn != nil {
		return m.TaskListFn(ctx, options)
	}
	return client.TaskListResult{}, nil
}

// mockPullResponse implements client.ImagePullResponse for tests.
type mockPullResponse struct{}

func (m *mockPullResponse) Read(p []byte) (n int, err error) { return 0, io.EOF }
func (m *mockPullResponse) Close() error                     { return nil }
func (m *mockPullResponse) Wait(_ context.Context) error     { return nil }
func (m *mockPullResponse) JSONMessages(_ context.Context) iter.Seq2[jsonstream.Message, error] {
	return func(yield func(jsonstream.Message, error) bool) {}
}

// nopLogsResult implements client.ContainerLogsResult for tests.
type nopLogsResult struct{}

func (n nopLogsResult) Read(p []byte) (int, error) { return 0, io.EOF }
func (n nopLogsResult) Close() error               { return nil }

// Ensure the mock types satisfy the necessary interfaces.
var _ io.ReadCloser = nopLogsResult{}
