package core

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gobs/args"
)

type ExecJob struct {
	BareJob     `mapstructure:",squash"`
	Client      DockerClient `json:"-" hash:"-"`
	Container   string
	User        string `default:"root"`
	TTY         bool   `default:"false"`
	Environment []string

	execID string
}

func NewExecJob(c DockerClient) *ExecJob {
	return &ExecJob{Client: c}
}

func (j *ExecJob) Run(ctx *Context) error {
	execID, err := j.buildExec()
	if err != nil {
		return err
	}

	j.execID = execID

	if err := j.startExec(ctx.Execution); err != nil {
		return err
	}

	inspect, err := j.inspectExec()
	if err != nil {
		return err
	}

	switch inspect.ExitCode {
	case 0:
		return nil
	case -1:
		return ErrUnexpected
	default:
		return fmt.Errorf("error non-zero exit code: %d", inspect.ExitCode)
	}
}

func (j *ExecJob) buildExec() (string, error) {
	resp, err := j.Client.ContainerExecCreate(context.Background(), j.Container, container.ExecOptions{
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          j.TTY,
		Cmd:          args.GetArgs(j.Command),
		User:         j.User,
		Env:          j.Environment,
	})

	if err != nil {
		return "", fmt.Errorf("error creating exec: %s", err)
	}

	return resp.ID, nil
}

func (j *ExecJob) startExec(e *Execution) error {
	resp, err := j.Client.ContainerExecAttach(context.Background(), j.execID, container.ExecAttachOptions{
		Tty: j.TTY,
	})
	if err != nil {
		return fmt.Errorf("error starting exec: %s", err)
	}
	defer resp.Close()

	if j.TTY {
		_, err = io.Copy(e.OutputStream, resp.Reader)
	} else {
		_, err = stdcopy.StdCopy(e.OutputStream, e.ErrorStream, resp.Reader)
	}
	if err != nil {
		return fmt.Errorf("error reading exec output: %s", err)
	}

	return nil
}

func (j *ExecJob) inspectExec() (container.ExecInspect, error) {
	i, err := j.Client.ContainerExecInspect(context.Background(), j.execID)
	if err != nil {
		return i, fmt.Errorf("error inspecting exec: %s", err)
	}

	return i, nil
}
