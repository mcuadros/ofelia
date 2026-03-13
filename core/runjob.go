package core

import (
	"fmt"
	"strconv"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/gobs/args"
)

var dockercfg *docker.AuthConfigurations

func init() {
	dockercfg, _ = docker.NewAuthConfigurationsFromDockerCfg()
}

type RunJob struct {
	BareJob `mapstructure:",squash"`
	Client  *docker.Client `json:"-"`
	User    string         `default:"root"`

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
	Entrypoint  string
	Volume      []string
	VolumesFrom []string `gcfg:"volumes-from" mapstructure:"volumes-from,"`
	Environment []string

	containerID string
}

func NewRunJob(c *docker.Client) *RunJob {
	return &RunJob{Client: c}
}

func (j *RunJob) Run(ctx *Context) error {
	var container *docker.Container
	var err error
	pull, _ := strconv.ParseBool(j.Pull)

	if j.Image != "" && j.Container == "" {
		if err = func() error {
			var pullError error

			// if Pull option "true"
			// try pulling image first
			if pull {
				if pullError = j.pullImage(); pullError == nil {
					ctx.Log("Pulled image " + j.Image)
					return nil
				}
			}

			// if Pull option "false"
			// try to find image locally first
			searchErr := j.searchLocalImage()
			if searchErr == nil {
				ctx.Log("Found locally image " + j.Image)
				return nil
			}

			// if couldn't find image locally, still try to pull
			if !pull && searchErr == ErrLocalImageNotFound {
				if pullError = j.pullImage(); pullError == nil {
					ctx.Log("Pulled image " + j.Image)
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

		container, err = j.buildContainer()
		if err != nil {
			return err
		}
	} else {
		container, err = j.Client.InspectContainer(j.Container)
		if err != nil {
			return err
		}
	}

	if container != nil {
		j.containerID = container.ID
	}

	// cleanup container if it is a created one
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

	if logsErr := j.Client.Logs(docker.LogsOptions{
		Container:    container.ID,
		OutputStream: ctx.Execution.OutputStream,
		ErrorStream:  ctx.Execution.ErrorStream,
		Stdout:       true,
		Stderr:       true,
		Since:        startTime.Unix(),
		RawTerminal:  j.TTY,
	}); logsErr != nil {
		ctx.Warn("failed to fetch container logs: " + logsErr.Error())
	}

	return err
}

func (j *RunJob) searchLocalImage() error {
	imgs, err := j.Client.ListImages(buildFindLocalImageOptions(j.Image))
	if err != nil {
		return err
	}

	if len(imgs) != 1 {
		return ErrLocalImageNotFound
	}

	return nil
}

func (j *RunJob) pullImage() error {
	o, a := buildPullOptions(j.Image)
	if err := j.Client.PullImage(o, a); err != nil {
		return fmt.Errorf("error pulling image %q: %s", j.Image, err)
	}

	return nil
}

func (j *RunJob) buildContainer() (*docker.Container, error) {
	c, err := j.Client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:        j.Image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          j.TTY,
			Entrypoint:   args.GetArgs(j.Entrypoint),
			Cmd:          args.GetArgs(j.Command),
			User:         j.User,
			Env:          j.Environment,
			Hostname:     j.Hostname,
		},
		NetworkingConfig: &docker.NetworkingConfig{},
		HostConfig: &docker.HostConfig{
			Binds:       j.Volume,
			VolumesFrom: j.VolumesFrom,
		},
	})

	if err != nil {
		return c, fmt.Errorf("error creating exec: %s", err)
	}

	if j.Network != "" {
		networkOpts := docker.NetworkFilterOpts{}
		networkOpts["name"] = map[string]bool{}
		networkOpts["name"][j.Network] = true
		if networks, err := j.Client.FilteredListNetworks(networkOpts); err == nil {
			for _, network := range networks {
				if err := j.Client.ConnectNetwork(network.ID, docker.NetworkConnectionOptions{
					Container: c.ID,
				}); err != nil {
					return c, fmt.Errorf("error connecting container to network: %s", err)
				}
			}
		}
	}

	return c, nil
}

func (j *RunJob) startContainer() error {
	return j.Client.StartContainer(j.containerID, &docker.HostConfig{})
}

func (j *RunJob) stopContainer(timeout uint) error {
	return j.Client.StopContainer(j.containerID, timeout)
}

func (j *RunJob) getContainer() (*docker.Container, error) {
	container, err := j.Client.InspectContainer(j.containerID)
	if err != nil {
		return nil, err
	}
	return container, nil
}

const (
	watchDuration      = time.Millisecond * 100
	maxProcessDuration = time.Hour * 24
)

func (j *RunJob) watchContainer() error {
	var s docker.State
	var r time.Duration
	for {
		time.Sleep(watchDuration)
		r += watchDuration

		if r > maxProcessDuration {
			return ErrMaxTimeRunning
		}

		c, err := j.Client.InspectContainer(j.containerID)
		if err != nil {
			return err
		}

		if !c.State.Running {
			s = c.State
			break
		}
	}

	switch s.ExitCode {
	case 0:
		return nil
	case -1:
		return ErrUnexpected
	default:
		return fmt.Errorf("error non-zero exit code: %d", s.ExitCode)
	}
}

func (j *RunJob) deleteContainer() error {
	if delete, _ := strconv.ParseBool(j.Delete); !delete {
		return nil
	}

	return j.Client.RemoveContainer(docker.RemoveContainerOptions{
		ID: j.containerID,
	})
}
