package core

import (
	"context"

	"github.com/moby/moby/client"
)

// DockerClient defines the subset of Docker API operations used by Ofelia.
// The official *client.Client from github.com/moby/moby/client satisfies this interface.
type DockerClient interface {
	Info(ctx context.Context, options client.InfoOptions) (client.SystemInfoResult, error)
	Events(ctx context.Context, options client.EventsListOptions) client.EventsResult

	ContainerList(ctx context.Context, options client.ContainerListOptions) (client.ContainerListResult, error)
	ContainerInspect(ctx context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error)
	ContainerCreate(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error)
	ContainerStart(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error)
	ContainerStop(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error)
	ContainerRemove(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error)
	ContainerLogs(ctx context.Context, containerID string, options client.ContainerLogsOptions) (client.ContainerLogsResult, error)

	ExecCreate(ctx context.Context, containerID string, options client.ExecCreateOptions) (client.ExecCreateResult, error)
	ExecAttach(ctx context.Context, execID string, options client.ExecAttachOptions) (client.ExecAttachResult, error)
	ExecInspect(ctx context.Context, execID string, options client.ExecInspectOptions) (client.ExecInspectResult, error)

	ImageList(ctx context.Context, options client.ImageListOptions) (client.ImageListResult, error)
	ImagePull(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error)

	NetworkList(ctx context.Context, options client.NetworkListOptions) (client.NetworkListResult, error)
	NetworkConnect(ctx context.Context, networkID string, options client.NetworkConnectOptions) (client.NetworkConnectResult, error)

	ServiceCreate(ctx context.Context, options client.ServiceCreateOptions) (client.ServiceCreateResult, error)
	ServiceInspect(ctx context.Context, serviceID string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error)
	ServiceRemove(ctx context.Context, serviceID string, options client.ServiceRemoveOptions) (client.ServiceRemoveResult, error)
	TaskList(ctx context.Context, options client.TaskListOptions) (client.TaskListResult, error)
}
