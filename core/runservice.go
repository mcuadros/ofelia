package core

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/swarm"
	docker "github.com/fsouza/go-dockerclient"
)

// Note: The ServiceJob is loosely inspired by https://github.com/alexellis/jaas/

type RunServiceJob struct {
	BareJob `mapstructure:",squash"`
	Client  *docker.Client `json:"-"`
	User    string         `default:"root"`
	TTY     bool           `default:"false"`
	// do not use bool values with "default:true" because if
	// user would set it to "false" explicitly, it still will be
	// changed to "true" https://github.com/mcuadros/ofelia/issues/135
	// so lets use strings here as workaround
	Delete  string `default:"true"`
	Image   string
	Network string
	Service string
}

func NewRunServiceJob(c *docker.Client) *RunServiceJob {
	return &RunServiceJob{Client: c}
}

// Main method for running a service-based job
// If the service has been provided it will start a new task for the existing service
// Otherwise it will create a new service based on the image and other parameters
func (j *RunServiceJob) Run(ctx *Context) error {

	if j.Image != "" {
		if err := j.pullImage(); err != nil {
			return err
		}
	}

	var svcID string
	if j.Service == "" {
		svc, err := j.buildService()

		if err != nil {
			return err
		}

		svcID = svc.ID
		ctx.Logger.Noticef("Created service %s for job %s\n", svcID, j.Name)
	} else {
		svc, err := j.inspectService(ctx, j.Service)
		if err != nil {
			return err
		}
		svcID = svc.ID
		ctx.Logger.Noticef("Found service %s for job %s\n", svcID, j.Name)

		_, err = j.scaleService(ctx, svcID, false)
		if err != nil {
			return err
		}

		_, err = j.scaleService(ctx, svcID, true)
		if err != nil {
			return err
		}
	}

	if err := j.watchContainer(ctx, svcID); err != nil {
		return err
	}

	if j.Service == "" {
		return j.deleteService(ctx, svcID)
	} else {
		return nil
	}
}

func (j *RunServiceJob) pullImage() error {
	o, a := buildPullOptions(j.Image)
	if err := j.Client.PullImage(o, a); err != nil {
		return fmt.Errorf("error pulling image %q: %s", j.Image, err)
	}

	return nil
}

func (j *RunServiceJob) buildService() (*swarm.Service, error) {

	//createOptions := types.ServiceCreateOptions{}

	max := uint64(1)
	createSvcOpts := docker.CreateServiceOptions{}

	createSvcOpts.ServiceSpec.TaskTemplate.ContainerSpec =
		&swarm.ContainerSpec{
			Image: j.Image,
		}

	// Make the service run once and not restart
	createSvcOpts.ServiceSpec.TaskTemplate.RestartPolicy =
		&swarm.RestartPolicy{
			MaxAttempts: &max,
			Condition:   swarm.RestartPolicyConditionNone,
		}

	// For a service to interact with other services in a stack,
	// we need to attach it to the same network
	if j.Network != "" {
		createSvcOpts.Networks = []swarm.NetworkAttachmentConfig{
			swarm.NetworkAttachmentConfig{
				Target: j.Network,
			},
		}
	}

	if j.Command != "" {
		createSvcOpts.ServiceSpec.TaskTemplate.ContainerSpec.Command = strings.Split(j.Command, " ")
	}

	svc, err := j.Client.CreateService(createSvcOpts)
	if err != nil {
		return nil, err
	}

	return svc, err
}

// Scale an existing service one replica up or down
func (j *RunServiceJob) scaleService(ctx *Context, svcID string, up bool) (*swarm.Service, error) {
	svc, err := j.inspectService(ctx, j.Service)
	if err != nil {
		return nil, err
	}

	replicas := *svc.Spec.Mode.Replicated.Replicas
	if up {
		replicas += 1
	} else {
		// If there already 0 replicas of a service, there is no need to scale down
		if replicas == 0 {
			return svc, err
		}
		replicas -= 1
	}

	updateSvcOpts := docker.UpdateServiceOptions{}

	updateSvcOpts.Name = svc.Spec.Name
	updateSvcOpts.Version = svc.Version.Index

	// The old spec is required, otherwise defaults will override the service
	updateSvcOpts.ServiceSpec = svc.Spec

	updateSvcOpts.Mode.Replicated =
		&swarm.ReplicatedService{
			Replicas: &replicas,
		}

	// Do the actual scaling
	err = j.Client.UpdateService(svcID, updateSvcOpts)
	if err != nil {
		return nil, err
	}

	// Give docker the time to do the scaling
	time.Sleep(time.Millisecond * 1000)
	return svc, err
}

const (

	// TODO are these const defined somewhere in the docker API?
	swarmError   = -999
	timeoutError = -998
)

var svcChecker = time.NewTicker(watchDuration)

func (j *RunServiceJob) watchContainer(ctx *Context, svcID string) error {
	exitCode := swarmError

	ctx.Logger.Noticef("Checking for service ID %s (%s) termination\n", svcID, j.Name)

	svc, err := j.inspectService(ctx, svcID)
	if err != nil {
		return err
	}

	// On every tick, check if all the services have completed, or have error out
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for _ = range svcChecker.C {

			// TODO will not work with longer existing services
			// TODO doesn't work
			if svc.CreatedAt.After(time.Now().Add(maxProcessDuration)) {
				err = ErrMaxTimeRunning
				return
			}

			taskExitCode, found := j.findTaskStatus(ctx, svc.ID)

			if found {
				exitCode = taskExitCode
				return
			}
		}
	}()

	wg.Wait()

	ctx.Logger.Noticef("Service ID %s (%s) has completed with exit code %d\n", svcID, j.Name, exitCode)

	switch exitCode {
	case 0:
		return nil
	case -1:
		return ErrUnexpected
	default:
		return fmt.Errorf("error non-zero exit code: %d", exitCode)
	}
	return err
}

func (j *RunServiceJob) findTaskStatus(ctx *Context, svcID string) (int, bool) {
	taskFilters := make(map[string][]string)
	taskFilters["service"] = []string{svcID}

	tasks, err := j.Client.ListTasks(docker.ListTasksOptions{
		Filters: taskFilters,
	})

	if err != nil {
		ctx.Logger.Errorf("Failed to find tasks fo service %s. Considering the task terminated: %s\n", svcID, err.Error())
		return 0, false
	}

	if len(tasks) == 0 {
		// That task is gone now (maybe someone else removed it. Our work here is done
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
				exitCode = 255 // force non-zero exit for task rejected
			}

			err = j.Client.GetServiceLogs(docker.LogsServiceOptions{
				Service:      svcID,
				Stderr:       true,
				Stdout:       true,
				Follow:       false,
				ErrorStream:  ctx.Execution.ErrorStream,
				OutputStream: ctx.Execution.OutputStream,
			})
			if err != nil {
				ctx.Logger.Errorf("Error getting logs for service: %s - %s \n", svcID, err.Error())
				return 0, false
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

	err := j.Client.RemoveService(docker.RemoveServiceOptions{
		ID: svcID,
	})

	if _, is := err.(*docker.NoSuchService); is {
		ctx.Logger.Warningf("Service %s cannot be removed. An error may have happened, "+
			"or it might have been removed by another process", svcID)
		return nil
	}

	return err

}

// Convenience method for inspecting a service
func (j *RunServiceJob) inspectService(ctx *Context, svcID string) (*swarm.Service, error) {
	var err error
	var svc *swarm.Service
	if j.Service == "" {
    	  svc, err = j.Client.InspectService(svcID)
	} else {
    	  svc, err = j.Client.InspectService(j.Service)
	}

	if err != nil {
		return nil, fmt.Errorf("Failed to inspect service %s: %s", j.Service, err.Error())
	}
	return svc, err
}
