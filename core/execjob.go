package core

import (
	"fmt"
	"io"

	"github.com/gobs/args"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
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
	execID, err := j.buildExec(ctx)
	if err != nil {
		return err
	}

	j.execID = execID

	if err := j.startExec(ctx); err != nil {
		return err
	}

	inspect, err := j.inspectExec(ctx)
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

func (j *ExecJob) buildExec(ctx *Context) (string, error) {
	resp, err := j.Client.ExecCreate(ctx.Context(), j.Container, client.ExecCreateOptions{
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          j.TTY,
		Cmd:          args.GetArgs(j.Command),
		User:         j.User,
		Env:          j.Environment,
	})

	if err != nil {
		return "", fmt.Errorf("error creating exec: %s", err)
	}

	return resp.ID, nil
}

func (j *ExecJob) startExec(ctx *Context) error {
	resp, err := j.Client.ExecAttach(ctx.Context(), j.execID, client.ExecAttachOptions{
		TTY: j.TTY,
	})
	if err != nil {
		return fmt.Errorf("error starting exec: %s", err)
	}
	defer resp.Close()

	if j.TTY {
		_, err = io.Copy(ctx.Execution.OutputStream, resp.Reader)
	} else {
		_, err = stdcopy.StdCopy(ctx.Execution.OutputStream, ctx.Execution.ErrorStream, resp.Reader)
	}
	if err != nil {
		return fmt.Errorf("error reading exec output: %s", err)
	}

	return nil
}

func (j *ExecJob) inspectExec(ctx *Context) (client.ExecInspectResult, error) {
	i, err := j.Client.ExecInspect(ctx.Context(), j.execID, client.ExecInspectOptions{})
	if err != nil {
		return i, fmt.Errorf("error inspecting exec: %s", err)
	}

	return i, nil
}
