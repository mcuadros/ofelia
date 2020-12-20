package cli

import (
	"errors"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

type DockerHandler struct {
	dockerClient *docker.Client
	notifier     dockerLabelsUpdate
}

type dockerLabelsUpdate interface {
	dockerLabelsUpdate(map[string]map[string]string)
}

// TODO: Implement an interface so the code does not have to use third parties directly
func (c *DockerHandler) GetInternalDockerClient() *docker.Client {
	return c.dockerClient
}

func (c *DockerHandler) buildDockerClient() (*docker.Client, error) {
	d, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	return d, nil
}

func NewDockerHandler(notifier dockerLabelsUpdate) (*DockerHandler, error) {
	c := &DockerHandler{}
	var err error
	c.dockerClient, err = c.buildDockerClient()
	c.notifier = notifier
	if err != nil {
		return nil, err
	}
	go c.watch()
	return c, nil
}

func (c *DockerHandler) watch() {
	// Poll for changes
	tick := time.Tick(10000 * time.Millisecond)
	for {
		select {
		case <-tick:
			labels, err := c.GetDockerLabels()
			if err != nil {
				// TODO: Log here

			}
			c.notifier.dockerLabelsUpdate(labels)
		}
	}
}

func (c *DockerHandler) GetDockerLabels() (map[string]map[string]string, error) {
	conts, err := c.dockerClient.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"label": {requiredLabelFilter},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(conts) == 0 {
		return nil, errors.New("Couldn't find containers with label 'ofelia.enabled=true'")
	}

	var labels = make(map[string]map[string]string)

	for _, c := range conts {
		if len(c.Names) > 0 && len(c.Labels) > 0 {
			name := strings.TrimPrefix(c.Names[0], "/")
			for k := range c.Labels {
				// remove all not relevant labels
				if !strings.HasPrefix(k, labelPrefix) {
					delete(c.Labels, k)
					continue
				}
			}

			labels[name] = c.Labels
		}
	}

	return labels, nil
}
