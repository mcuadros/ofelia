package core

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// DockerClient defines the subset of Docker API operations used by Ofelia.
type DockerClient interface {
	Info(ctx context.Context) (system.Info, error)

	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error)
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error)

	ContainerExecCreate(ctx context.Context, container string, config container.ExecOptions) (container.ExecCreateResponse, error)
	ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error)
	ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error)

	ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)

	NetworkList(ctx context.Context, options network.ListOptions) ([]network.Inspect, error)
	NetworkConnect(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error

	ServiceCreate(ctx context.Context, service swarm.ServiceSpec, options swarm.ServiceCreateOptions) (swarm.ServiceCreateResponse, error)
	ServiceInspectWithRaw(ctx context.Context, serviceID string, options swarm.ServiceInspectOptions) (swarm.Service, []byte, error)
	ServiceRemove(ctx context.Context, serviceID string) error
	TaskList(ctx context.Context, options swarm.TaskListOptions) ([]swarm.Task, error)
}

// NewFilterArgs creates a filters.Args from a map of key-value pairs.
func NewFilterArgs(filtersMap map[string][]string) filters.Args {
	f := filters.NewArgs()
	for key, values := range filtersMap {
		for _, v := range values {
			f.Add(key, v)
		}
	}
	return f
}
