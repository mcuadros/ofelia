package core

import (
	"fmt"
	"io"

	"github.com/fsouza/go-dockerclient"
)

type ExecJob struct {
	BasicJob
	Client                    *docker.Client
	Command                   []string
	Container                 string
	User                      string
	TTY                       bool
	InputStream               io.Reader
	OutputStream, ErrorStream io.Writer
}

func (j *ExecJob) Run() {
	var err error
	var exec *docker.Exec

	e := j.Start()
	defer func() { j.Stop(e, err) }()

	exec, err = j.buildExec()
	if err != nil {
		return
	}

	err = j.startExec(exec)
	return
}

func (j *ExecJob) buildExec() (*docker.Exec, error) {
	exec, err := j.Client.CreateExec(docker.CreateExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          j.TTY,
		Cmd:          j.Command,
		Container:    j.Container,
		User:         j.User,
	})

	if err != nil {
		return exec, fmt.Errorf("error creating exec: %s", err)
	}

	return exec, nil
}
func (j *ExecJob) startExec(exec *docker.Exec) error {
	err := j.Client.StartExec(exec.ID, docker.StartExecOptions{
		Tty:          j.TTY,
		InputStream:  j.InputStream,
		OutputStream: j.OutputStream,
		ErrorStream:  j.ErrorStream,
		RawTerminal:  j.TTY,
	})

	if err != nil {
		return fmt.Errorf("error starting exec: %s", err)
	}

	return nil
}
