package core

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"
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
	if err := j.pullImage(); err != nil {
		return err
	}

	svc, err := j.buildService()
	if err != nil {
		return err
	}

	ctx.Logger.Info("Created new service", "id", svc.ID, "job", j.Name)

	if err := j.watchContainer(ctx, svc.ID); err != nil {
		return err
	}

	return j.deleteService(ctx, svc.ID)
}

func (j *RunServiceJob) pullImage() error {
	ref, encodedAuth := buildPullOptions(j.Image)
	reader, err := j.Client.ImagePull(context.Background(), ref, image.PullOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return fmt.Errorf("error pulling image %q: %s", j.Image, err)
	}
	defer reader.Close()
	if err := consumePullResponse(reader); err != nil {
		return fmt.Errorf("error pulling image %q: %s", j.Image, err)
	}
	return nil
}

func (j *RunServiceJob) buildService() (*swarm.ServiceCreateResponse, error) {
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

	resp, err := j.Client.ServiceCreate(context.Background(), spec, swarm.ServiceCreateOptions{})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

const (
	swarmError   = -999
	timeoutError = -998
)

var svcChecker = time.NewTicker(watchDuration)

func (j *RunServiceJob) watchContainer(ctx *Context, svcID string) error {
	exitCode := swarmError

	ctx.Logger.Info("Checking for service termination", "id", svcID, "job", j.Name)

	svc, _, err := j.Client.ServiceInspectWithRaw(context.Background(), svcID, swarm.ServiceInspectOptions{})
	if err != nil {
		return fmt.Errorf("Failed to inspect service %s: %s", svcID, err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for range svcChecker.C {
			if svc.CreatedAt.After(time.Now().Add(maxProcessDuration)) {
				err = ErrMaxTimeRunning
				return
			}

			taskExitCode, found := j.findtaskstatus(ctx, svc.ID)
			if found {
				exitCode = taskExitCode
				return
			}
		}
	}()

	wg.Wait()

	ctx.Logger.Info("Service has completed", "id", svcID, "job", j.Name, "exit_code", exitCode)
	return err
}

func (j *RunServiceJob) findtaskstatus(ctx *Context, taskID string) (int, bool) {
	tasks, err := j.Client.TaskList(context.Background(), swarm.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", taskID)),
	})

	if err != nil {
		ctx.Logger.Error("Failed to find task ID. Considering the task terminated.", "id", taskID, "error", err)
		return 0, false
	}

	if len(tasks) == 0 {
		return 0, true
	}

	exitCode := 1
	var done bool
	stopStates := []swarm.TaskState{
		swarm.TaskStateComplete,
		swarm.TaskStateFailed,
		swarm.TaskStateRejected,
	}

	for _, task := range tasks {
		stop := false
		for _, stopState := range stopStates {
			if task.Status.State == stopState {
				stop = true
				break
			}
		}

		if stop {
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

	err := j.Client.ServiceRemove(context.Background(), svcID)
	if err != nil && errdefs.IsNotFound(err) {
		ctx.Logger.Warning("Service cannot be removed. An error may have happened, "+
			"or it might have been removed by another process", "id", svcID)
		return nil
	}

	return err
}
