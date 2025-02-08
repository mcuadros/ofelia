package core

import (
	"fmt"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/gobs/args"
)

type ExecJob struct {
	BareJob     `mapstructure:",squash"`
	Client      *docker.Client `json:"-" hash:"-"`
	Container   string
	User        string `default:"root"`
	TTY         bool   `default:"false"`
	Environment []string

	execID string
}

func NewExecJob(c *docker.Client) *ExecJob {
	return &ExecJob{Client: c}
}

func (j *ExecJob) Run(ctx *Context) error {
	exec, err := j.buildExec()
	if err != nil {
		return err
	}

	if exec != nil {
		j.execID = exec.ID
	}

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

func (j *ExecJob) buildExec() (*docker.Exec, error) {
	exec, err := j.Client.CreateExec(docker.CreateExecOptions{
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          j.TTY,
		Cmd:          args.GetArgs(j.Command),
		Container:    j.Container,
		User:         j.User,
		Env:          j.Environment,
	})

	if err != nil {
		return exec, fmt.Errorf("error creating exec: %s", err)
	}

	return exec, nil
}

func (j *ExecJob) startExec(e *Execution) error {
	err := j.Client.StartExec(j.execID, docker.StartExecOptions{
		Tty:          j.TTY,
		OutputStream: e.OutputStream,
		ErrorStream:  e.ErrorStream,
		RawTerminal:  j.TTY,
	})

	if err != nil {
		return fmt.Errorf("error starting exec: %s", err)
	}

	return nil
}

func (j *ExecJob) inspectExec() (*docker.ExecInspect, error) {
	i, err := j.Client.InspectExec(j.execID)

	if err != nil {
		return i, fmt.Errorf("error inspecting exec: %s", err)
	}

	return i, nil
}
