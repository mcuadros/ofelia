package core

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gobs/args"
)

type RunJob struct {
	BareJob `mapstructure:",squash"`
	Client  DockerClient `json:"-"`
	User    string       `default:"root"`

	TTY bool `default:"false"`

	// do not use bool values with "default:true" because if
	// user would set it to "false" explicitly, it still will be
	// changed to "true" https://github.com/mcuadros/ofelia/issues/135
	// so lets use strings here as workaround
	Delete string `default:"true"`
	Pull   string `default:"true"`

	Image       string
	Network     string
	Hostname    string
	Container   string
	Entrypoint  *string
	Volume      []string
	VolumesFrom []string `gcfg:"volumes-from" mapstructure:"volumes-from,"`
	Environment []string

	containerID string
}

func NewRunJob(c DockerClient) *RunJob {
	return &RunJob{Client: c}
}

func (j *RunJob) Run(ctx *Context) error {
	var containerID string
	var err error
	pull, _ := strconv.ParseBool(j.Pull)

	if j.Image != "" && j.Container == "" {
		if err = func() error {
			var pullError error

			if pull {
				if pullError = j.pullImage(); pullError == nil {
					ctx.Logger.Debug("Pulled new image", "image", j.Image, "pull", pull)
					return nil
				}
			}

			searchErr := j.searchLocalImage()
			if searchErr == nil {
				ctx.Logger.Debug("Found image locally", "image", j.Image, "pull", pull)
				return nil
			}

			if !pull && searchErr == ErrLocalImageNotFound {
				if pullError = j.pullImage(); pullError == nil {
					ctx.Logger.Debug("Pulled new image", "image", j.Image, "pull", pull)
					return nil
				}
			}

			if pullError != nil {
				return pullError
			}

			if searchErr != nil {
				return searchErr
			}

			return nil
		}(); err != nil {
			return err
		}

		containerID, err = j.buildContainer()
		if err != nil {
			return err
		}
	} else {
		resp, inspectErr := j.Client.ContainerInspect(context.Background(), j.Container)
		if inspectErr != nil {
			return inspectErr
		}
		containerID = resp.ID
	}

	j.containerID = containerID

	if j.Container == "" {
		defer func() {
			if delErr := j.deleteContainer(); delErr != nil {
				ctx.Warn("failed to delete container: " + delErr.Error())
			}
		}()
	}

	startTime := time.Now()
	if err := j.startContainer(); err != nil {
		return err
	}

	err = j.watchContainer()
	if err == ErrUnexpected {
		return err
	}

	if logsErr := j.fetchLogs(ctx, startTime); logsErr != nil {
		ctx.Warn("failed to fetch container logs: " + logsErr.Error())
	}

	return err
}

func (j *RunJob) fetchLogs(ctx *Context, startTime time.Time) error {
	reader, err := j.Client.ContainerLogs(context.Background(), j.containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      startTime.Format(time.RFC3339Nano),
	})
	if err != nil {
		return err
	}
	defer reader.Close()

	if j.TTY {
		_, err = io.Copy(ctx.Execution.OutputStream, reader)
	} else {
		_, err = stdcopy.StdCopy(ctx.Execution.OutputStream, ctx.Execution.ErrorStream, reader)
	}
	return err
}

func (j *RunJob) searchLocalImage() error {
	imgs, err := j.Client.ImageList(context.Background(), buildFindLocalImageOptions(j.Image))
	if err != nil {
		return err
	}

	if len(imgs) != 1 {
		return ErrLocalImageNotFound
	}

	return nil
}

func (j *RunJob) pullImage() error {
	ref, encodedAuth := buildPullOptions(j.Image)
	reader, err := j.Client.ImagePull(context.Background(), ref, image.PullOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return fmt.Errorf("error pulling image %q: %s", j.Image, err)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

func (j *RunJob) buildContainer() (string, error) {
	config := &container.Config{
		Image:        j.Image,
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          j.TTY,
		Cmd:          args.GetArgs(j.Command),
		User:         j.User,
		Env:          j.Environment,
		Hostname:     j.Hostname,
	}
	if j.Entrypoint != nil {
		config.Entrypoint = args.GetArgs(*j.Entrypoint)
	}

	resp, err := j.Client.ContainerCreate(context.Background(), config, &container.HostConfig{
		Binds:       j.Volume,
		VolumesFrom: j.VolumesFrom,
	}, &network.NetworkingConfig{}, "")
	if err != nil {
		return "", fmt.Errorf("error creating container: %s", err)
	}

	if j.Network != "" {
		networkFilter := filters.NewArgs(filters.Arg("name", j.Network))
		networks, err := j.Client.NetworkList(context.Background(), network.ListOptions{Filters: networkFilter})
		if err == nil {
			for _, net := range networks {
				if err := j.Client.NetworkConnect(context.Background(), net.ID, resp.ID, nil); err != nil {
					return resp.ID, fmt.Errorf("error connecting container to network: %s", err)
				}
			}
		}
	}

	return resp.ID, nil
}

func (j *RunJob) startContainer() error {
	return j.Client.ContainerStart(context.Background(), j.containerID, container.StartOptions{})
}

func (j *RunJob) stopContainer(timeout uint) error {
	t := int(timeout)
	return j.Client.ContainerStop(context.Background(), j.containerID, container.StopOptions{Timeout: &t})
}

const (
	watchDuration      = time.Millisecond * 100
	maxProcessDuration = time.Hour * 24
)

func (j *RunJob) watchContainer() error {
	var r time.Duration
	for {
		time.Sleep(watchDuration)
		r += watchDuration

		if r > maxProcessDuration {
			return ErrMaxTimeRunning
		}

		resp, err := j.Client.ContainerInspect(context.Background(), j.containerID)
		if err != nil {
			return err
		}

		if resp.State != nil && !resp.State.Running {
			switch resp.State.ExitCode {
			case 0:
				return nil
			case -1:
				return ErrUnexpected
			default:
				return fmt.Errorf("error non-zero exit code: %d", resp.State.ExitCode)
			}
		}
	}
}

func (j *RunJob) deleteContainer() error {
	if delete, _ := strconv.ParseBool(j.Delete); !delete {
		return nil
	}

	return j.Client.ContainerRemove(context.Background(), j.containerID, container.RemoveOptions{})
}
