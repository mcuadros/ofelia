package core

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/fsouza/go-dockerclient"
)

// Note: The ServiceJob is loosely inspired by https://github.com/alexellis/jaas/

type RunServiceJob struct {
	BareJob
	Client  *docker.Client `json:"-"`
	User    string         `default:"root"`
	TTY     bool           `default:"false"`
	Delete  bool           `default:"true"`
	Image   string
	Network string
}

func NewRunServiceJob(c *docker.Client) *RunServiceJob {
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

	fmt.Printf("Created service %s for job %s\n", svc.ID, j.Name)

	if err := j.watchContainer(svc.ID); err != nil {
		return err
	}

	return j.deleteService(svc.ID)
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

const (

	// TODO are these const defined somewhere in the docker API?
	swarmError   = -999
	timeoutError = -998
)

var svcChecker = time.NewTicker(watchDuration)

func (j *RunServiceJob) watchContainer(svcID string) error {
	svcFilters := make(map[string][]string)
	svcFilters["id"] = []string{svcID}

	exitCode := swarmError

	var listSvcOpts docker.ListServicesOptions
	listSvcOpts.Filters = svcFilters

	list, _ := j.Client.ListServices(listSvcOpts)

	fmt.Printf("Checking for service ID %s (%s) termination\n", svcID, j.Name)

	// On every tick, check if all the services have completed, or have error out
	var wg sync.WaitGroup
	wg.Add(1)

	var err error
	go func() {
		defer wg.Done()
		for _ = range svcChecker.C {
			for _, svc := range list {
				if svc.CreatedAt.After(time.Now().Add(maxProcessDuration)) {
					err = ErrMaxTimeRunning
					return
				}

				taskExitCode, found := j.findtaskstatus(svc.ID)
				if found {
					exitCode = taskExitCode
					return
				}
			}
		}
	}()

	wg.Wait()

	fmt.Printf("Service ID %s (%s) has completed\n", svcID, j.Name)
	return err
}

func (j *RunServiceJob) findtaskstatus(taskID string) (int, bool) {
	taskFilters := make(map[string][]string)
	taskFilters["service"] = []string{taskID}

	tasks, err := j.Client.ListTasks(docker.ListTasksOptions{
		Filters: taskFilters,
	})

	if err != nil {
		fmt.Printf("Failed to find task ID %s: %s\n", taskID, err.Error())
		return 0, false
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
			done = true
			break
		}
	}
	return exitCode, done
}

func (j *RunServiceJob) deleteService(svcID string) error {
	if !j.Delete {
		return nil
	}

	return j.Client.RemoveService(docker.RemoveServiceOptions{
		ID: svcID,
	})
}
