package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/gobs/args"
)

var dockercfg *docker.AuthConfigurations

func init() {
	dockercfg, _ = docker.NewAuthConfigurationsFromDockerCfg()
}

type RunJob struct {
	BareJob
	Client *docker.Client `json:"-"`
	User   string         `default:"root"`
	TTY    bool           `default:"false"`
	Delete bool           `default:"true"`
	Image  string
}

func NewRunJob(c *docker.Client) *RunJob {
	return &RunJob{Client: c}
}

func (j *RunJob) Run(ctx *Context) error {
	if err := j.pullImage(); err != nil {
		return err
	}

	container, err := j.buildContainer()
	if err != nil {
		return err
	}

	if err := j.startContainer(ctx.Execution, container); err != nil {
		return err
	}

	if err := j.watchContainer(container.ID); err != nil {
		return err
	}

	return j.deleteContainer(container.ID)
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
			Cmd:          args.GetArgs(j.Command),
			User:         j.User,
		},
	})

	if err != nil {
		return c, fmt.Errorf("error creating exec: %s", err)
	}

	return c, nil
}

func (j *RunJob) startContainer(e *Execution, c *docker.Container) error {
	return j.Client.StartContainer(c.ID, &docker.HostConfig{})
}

const (
	watchDuration      = time.Millisecond * 100
	maxProcessDuration = time.Hour * 24
)

func (j *RunJob) watchContainer(containerID string) error {
	var s docker.State
	var r time.Duration
	for {
		time.Sleep(watchDuration)
		r += watchDuration

		if r > maxProcessDuration {
			return ErrMaxTimeRunning
		}

		c, err := j.Client.InspectContainer(containerID)
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

func (j *RunJob) deleteContainer(containerID string) error {
	if !j.Delete {
		return nil
	}

	return j.Client.RemoveContainer(docker.RemoveContainerOptions{
		ID: containerID,
	})
}

func buildPullOptions(image string) (docker.PullImageOptions, docker.AuthConfiguration) {
	tag := "latest"
	registry := ""

	parts := strings.Split(image, ":")
	if len(parts) == 2 {
		tag = parts[1]
	}

	name := parts[0]
	parts = strings.Split(name, "/")
	if len(parts) > 2 {
		registry = parts[0]
	}

	return docker.PullImageOptions{
		Repository: name,
		Registry:   registry,
		Tag:        tag,
	}, buildAuthConfiguration(registry)
}

func buildAuthConfiguration(registry string) docker.AuthConfiguration {
	var auth docker.AuthConfiguration
	if dockercfg == nil {
		return auth
	}

	auth, _ = dockercfg.Configs[registry]
	return auth
}
