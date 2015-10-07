package core

import (
	"errors"
	"fmt"

	"github.com/fsouza/go-dockerclient"
	"github.com/gobs/args"
)

var ErrUnexpected = errors.New("error unexpected, docker has returned exit code -1, maybe wrong user?")

type ExecJob struct {
	BareJob
	Client    *docker.Client `json:"-"`
	Container string
	User      string `default:"root"`
	TTY       bool   `default:"false"`
}

func NewExecJob(c *docker.Client) *ExecJob {
	return &ExecJob{Client: c}
}

func (j *ExecJob) Run(ctx *Context) error {
	exec, err := j.buildExec()
	if err != nil {
		return err
	}

	if err := j.startExec(ctx.Execution, exec); err != nil {
		return err
	}

	return j.inspectExec(exec)
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
	})

	if err != nil {
		return exec, fmt.Errorf("error creating exec: %s", err)
	}

	return exec, nil
}

func (j *ExecJob) startExec(e *Execution, exec *docker.Exec) error {
	err := j.Client.StartExec(exec.ID, docker.StartExecOptions{
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

func (j *ExecJob) inspectExec(exec *docker.Exec) error {
	i, err := j.Client.InspectExec(exec.ID)

	if err != nil {
		return fmt.Errorf("error inspecting exec: %s", err)
	}

	switch i.ExitCode {
	case 0:
		return nil
	case -1:
		return ErrUnexpected
	default:
		return fmt.Errorf("error non-zero exit code: %d", i.ExitCode)
	}
}
