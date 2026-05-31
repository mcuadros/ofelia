package core

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
)

type mockDockerClient struct {
	InfoFn                  func(ctx context.Context) (system.Info, error)
	ContainerListFn         func(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	ContainerInspectFn      func(ctx context.Context, containerID string) (container.InspectResponse, error)
	ContainerCreateFn       func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.CreateResponse, error)
	ContainerStartFn        func(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStopFn         func(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRemoveFn       func(ctx context.Context, containerID string, options container.RemoveOptions) error
	ContainerLogsFn         func(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error)
	ContainerExecCreateFn   func(ctx context.Context, ctr string, config container.ExecOptions) (container.ExecCreateResponse, error)
	ContainerExecAttachFn   func(ctx context.Context, execID string, config container.ExecAttachOptions) (HijackedResponse, error)
	ContainerExecInspectFn  func(ctx context.Context, execID string) (container.ExecInspect, error)
	ImageListFn             func(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	ImagePullFn             func(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)
	NetworkListFn           func(ctx context.Context, options network.ListOptions) ([]network.Inspect, error)
	NetworkConnectFn        func(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error
	ServiceCreateFn         func(ctx context.Context, service swarm.ServiceSpec, options swarm.ServiceCreateOptions) (swarm.ServiceCreateResponse, error)
	ServiceInspectWithRawFn func(ctx context.Context, serviceID string, options swarm.ServiceInspectOptions) (swarm.Service, []byte, error)
	ServiceRemoveFn         func(ctx context.Context, serviceID string) error
	TaskListFn              func(ctx context.Context, options swarm.TaskListOptions) ([]swarm.Task, error)
}

func (m *mockDockerClient) Info(ctx context.Context) (system.Info, error) {
	if m.InfoFn != nil {
		return m.InfoFn(ctx)
	}
	return system.Info{}, nil
}

func (m *mockDockerClient) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	if m.ContainerListFn != nil {
		return m.ContainerListFn(ctx, options)
	}
	return nil, nil
}

func (m *mockDockerClient) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	if m.ContainerInspectFn != nil {
		return m.ContainerInspectFn(ctx, containerID)
	}
	return container.InspectResponse{}, nil
}

func (m *mockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.CreateResponse, error) {
	if m.ContainerCreateFn != nil {
		return m.ContainerCreateFn(ctx, config, hostConfig, networkingConfig, containerName)
	}
	return container.CreateResponse{ID: "mock-container-id"}, nil
}

func (m *mockDockerClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	if m.ContainerStartFn != nil {
		return m.ContainerStartFn(ctx, containerID, options)
	}
	return nil
}

func (m *mockDockerClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	if m.ContainerStopFn != nil {
		return m.ContainerStopFn(ctx, containerID, options)
	}
	return nil
}

func (m *mockDockerClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	if m.ContainerRemoveFn != nil {
		return m.ContainerRemoveFn(ctx, containerID, options)
	}
	return nil
}

func (m *mockDockerClient) ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	if m.ContainerLogsFn != nil {
		return m.ContainerLogsFn(ctx, containerID, options)
	}
	return io.NopCloser(io.LimitReader(nil, 0)), nil
}

func (m *mockDockerClient) ContainerExecCreate(ctx context.Context, ctr string, config container.ExecOptions) (container.ExecCreateResponse, error) {
	if m.ContainerExecCreateFn != nil {
		return m.ContainerExecCreateFn(ctx, ctr, config)
	}
	return container.ExecCreateResponse{ID: "mock-exec-id"}, nil
}

func (m *mockDockerClient) ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (HijackedResponse, error) {
	if m.ContainerExecAttachFn != nil {
		return m.ContainerExecAttachFn(ctx, execID, config)
	}
	return HijackedResponse{Reader: io.LimitReader(nil, 0)}, nil
}

func (m *mockDockerClient) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	if m.ContainerExecInspectFn != nil {
		return m.ContainerExecInspectFn(ctx, execID)
	}
	return container.ExecInspect{ExitCode: 0}, nil
}

func (m *mockDockerClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	if m.ImageListFn != nil {
		return m.ImageListFn(ctx, options)
	}
	return []image.Summary{{}}, nil
}

func (m *mockDockerClient) ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error) {
	if m.ImagePullFn != nil {
		return m.ImagePullFn(ctx, refStr, options)
	}
	return io.NopCloser(io.LimitReader(nil, 0)), nil
}

func (m *mockDockerClient) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Inspect, error) {
	if m.NetworkListFn != nil {
		return m.NetworkListFn(ctx, options)
	}
	return nil, nil
}

func (m *mockDockerClient) NetworkConnect(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error {
	if m.NetworkConnectFn != nil {
		return m.NetworkConnectFn(ctx, networkID, containerID, config)
	}
	return nil
}

func (m *mockDockerClient) ServiceCreate(ctx context.Context, service swarm.ServiceSpec, options swarm.ServiceCreateOptions) (swarm.ServiceCreateResponse, error) {
	if m.ServiceCreateFn != nil {
		return m.ServiceCreateFn(ctx, service, options)
	}
	return swarm.ServiceCreateResponse{ID: "mock-service-id"}, nil
}

func (m *mockDockerClient) ServiceInspectWithRaw(ctx context.Context, serviceID string, options swarm.ServiceInspectOptions) (swarm.Service, []byte, error) {
	if m.ServiceInspectWithRawFn != nil {
		return m.ServiceInspectWithRawFn(ctx, serviceID, options)
	}
	return swarm.Service{}, nil, nil
}

func (m *mockDockerClient) ServiceRemove(ctx context.Context, serviceID string) error {
	if m.ServiceRemoveFn != nil {
		return m.ServiceRemoveFn(ctx, serviceID)
	}
	return nil
}

func (m *mockDockerClient) TaskList(ctx context.Context, options swarm.TaskListOptions) ([]swarm.Task, error) {
	if m.TaskListFn != nil {
		return m.TaskListFn(ctx, options)
	}
	return nil, nil
}
