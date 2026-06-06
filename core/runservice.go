package core

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

type RunServiceJob struct {
	BareJob `mapstructure:",squash"`
	Client  DockerClient `json:"-"`
	User    string       `default:"root"`
	TTY     bool         `default:"false"`
	// do not use bool values with "default:true" because if
	// user would set it to "false" explicitly, it still will be
	// changed to "true" https://github.com/mcuadros/ofelia/issues/135
	// so lets use strings here as workaround
	Delete  string `default:"true"`
	Image   string
	Network string
}

func NewRunServiceJob(c DockerClient) *RunServiceJob {
	return &RunServiceJob{Client: c}
}

func (j *RunServiceJob) Run(ctx *Context) error {
	if err := pullImage(j.Client, j.Image, ctx.Context()); err != nil {
		return err
	}

	svcID, err := j.buildService(ctx)
	if err != nil {
		return err
	}

	ctx.Logger.Info("Created new service", "id", svcID, "job", j.Name)

	if err := j.watchContainer(ctx, svcID); err != nil {
		return err
	}

	return j.deleteService(ctx, svcID)
}

func (j *RunServiceJob) buildService(ctx *Context) (string, error) {
	max := uint64(1)

	spec := swarm.ServiceSpec{}
	spec.TaskTemplate.ContainerSpec = &swarm.ContainerSpec{
		Image: j.Image,
	}

	spec.TaskTemplate.RestartPolicy = &swarm.RestartPolicy{
		MaxAttempts: &max,
		Condition:   swarm.RestartPolicyConditionNone,
	}

	if j.Network != "" {
		spec.TaskTemplate.Networks = []swarm.NetworkAttachmentConfig{
			{Target: j.Network},
		}
	}

	if j.Command != "" {
		spec.TaskTemplate.ContainerSpec.Command = strings.Split(j.Command, " ")
	}

	resp, err := j.Client.ServiceCreate(ctx.Context(), client.ServiceCreateOptions{
		Spec: spec,
	})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

const (
	swarmError   = -999
	timeoutError = -998
)

func (j *RunServiceJob) watchContainer(ctx *Context, svcID string) error {
	exitCode := swarmError

	ctx.Logger.Info("Checking for service termination", "id", svcID, "job", j.Name)

	svc, err := j.Client.ServiceInspect(ctx.Context(), svcID, client.ServiceInspectOptions{})
	if err != nil {
		return fmt.Errorf("Failed to inspect service %s: %s", svcID, err.Error())
	}

	ticker := time.NewTicker(watchDuration)
	defer ticker.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for range ticker.C {
			if time.Since(svc.Service.CreatedAt) > maxProcessDuration {
				err = ErrMaxTimeRunning
				return
			}

			taskExitCode, found := j.findtaskstatus(ctx, svc.Service.ID)
			if found {
				exitCode = taskExitCode
				return
			}
		}
	}()

	wg.Wait()

	ctx.Logger.Info("Service has completed", "id", svcID, "job", j.Name, "exit_code", exitCode)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("error non-zero exit code: %d", exitCode)
	}
	return nil
}

func (j *RunServiceJob) findtaskstatus(ctx *Context, taskID string) (int, bool) {
	resp, err := j.Client.TaskList(ctx.Context(), client.TaskListOptions{
		Filters: client.Filters{}.Add("service", taskID),
	})

	if err != nil {
		ctx.Logger.Error("Failed to list tasks, will retry", "id", taskID, "error", err)
		return 0, false
	}

	if len(resp.Items) == 0 {
		return 0, false
	}

	exitCode := 1
	var done bool
	stopStates := []swarm.TaskState{
		swarm.TaskStateComplete,
		swarm.TaskStateFailed,
		swarm.TaskStateRejected,
	}

	for _, task := range resp.Items {
		stop := false
		for _, stopState := range stopStates {
			if task.Status.State == stopState {
				stop = true
				break
			}
		}

		if stop {
			if task.Status.ContainerStatus == nil {
				exitCode = 255
				done = true
				break
			}
			exitCode = task.Status.ContainerStatus.ExitCode
			if exitCode == 0 && task.Status.State == swarm.TaskStateRejected {
				exitCode = 255
			}
			done = true
			break
		}
	}
	return exitCode, done
}

func (j *RunServiceJob) deleteService(ctx *Context, svcID string) error {
	if delete, _ := strconv.ParseBool(j.Delete); !delete {
		return nil
	}

	_, err := j.Client.ServiceRemove(ctx.Context(), svcID, client.ServiceRemoveOptions{})
	if err != nil && errdefs.IsNotFound(err) {
		ctx.Logger.Warning("Service cannot be removed. An error may have happened, "+
			"or it might have been removed by another process", "id", svcID)
		return nil
	}

	return err
}
