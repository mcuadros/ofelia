package core

import (
	"context"
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
	if err := j.pullImage(); err != nil {
		return err
	}

	svcID, err := j.buildService()
	if err != nil {
		return err
	}

	ctx.Logger.Info("Created new service", "id", svcID, "job", j.Name)

	if err := j.watchContainer(ctx, svcID); err != nil {
		return err
	}

	return j.deleteService(ctx, svcID)
}

func (j *RunServiceJob) pullImage() error {
	ref, encodedAuth := buildPullOptions(j.Image)
	resp, err := j.Client.ImagePull(context.Background(), ref, client.ImagePullOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return fmt.Errorf("error pulling image %q: %s", j.Image, err)
	}
	defer resp.Close()
	if err := resp.Wait(context.Background()); err != nil {
		return fmt.Errorf("error pulling image %q: %s", j.Image, err)
	}
	return nil
}

func (j *RunServiceJob) buildService() (string, error) {
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

	resp, err := j.Client.ServiceCreate(context.Background(), client.ServiceCreateOptions{
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

var svcChecker = time.NewTicker(watchDuration)

func (j *RunServiceJob) watchContainer(ctx *Context, svcID string) error {
	exitCode := swarmError

	ctx.Logger.Info("Checking for service termination", "id", svcID, "job", j.Name)

	svc, err := j.Client.ServiceInspect(context.Background(), svcID, client.ServiceInspectOptions{})
	if err != nil {
		return fmt.Errorf("Failed to inspect service %s: %s", svcID, err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for range svcChecker.C {
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
	resp, err := j.Client.TaskList(context.Background(), client.TaskListOptions{
		Filters: client.Filters{}.Add("service", taskID),
	})

	if err != nil {
		ctx.Logger.Error("Failed to find task ID. Considering the task terminated.", "id", taskID, "error", err)
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

	_, err := j.Client.ServiceRemove(context.Background(), svcID, client.ServiceRemoveOptions{})
	if err != nil && errdefs.IsNotFound(err) {
		ctx.Logger.Warning("Service cannot be removed. An error may have happened, "+
			"or it might have been removed by another process", "id", svcID)
		return nil
	}

	return err
}
