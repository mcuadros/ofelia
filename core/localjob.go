package core

import (
	"os/exec"

	"github.com/gobs/args"
)

type LocalJob struct {
	BareJob     `mapstructure:",squash"`
	Dir         string
	Environment []string
}

func NewLocalJob() *LocalJob {
	return &LocalJob{}
}

func (j *LocalJob) Run(ctx *Context) error {
	cmd, err := j.buildCommand(ctx)
	if err != nil {
		return err
	}

	return cmd.Run()
}

func (j *LocalJob) buildCommand(ctx *Context) (*exec.Cmd, error) {
	args := args.GetArgs(j.Command)
	bin, err := exec.LookPath(args[0])
	if err != nil {
		return nil, err
	}

	return &exec.Cmd{
		Path:   bin,
		Args:   args,
		Stdout: ctx.Execution.OutputStream,
		Stderr: ctx.Execution.ErrorStream,
		Env:    j.Environment,
		Dir:    j.Dir,
	}, nil
}
