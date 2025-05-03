package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/go-viper/mapstructure/v2"
	"github.com/mcuadros/ofelia/core"
)

const (
	labelPrefix = "ofelia"

	requiredLabel       = labelPrefix + ".enabled"
	requiredLabelFilter = requiredLabel + "=true"
	serviceLabel        = labelPrefix + ".service"
)

var (
	errNoContainersMatchingFilters = errors.New("no containers matching filters")
	errInvalidDockerFilter         = errors.New("invalid docker filter")
	errFailedToListContainers      = errors.New("failed to list containers")
)

type DockerHandler struct {
	dockerClient      *docker.Client
	notifier          labelConfigUpdater
	configsFromLabels bool
	logger            core.Logger
	filters           []string
}

type labelConfigUpdater interface {
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

func NewDockerHandler(config *Config, dockerFilters []string, configsFromLabels bool, logger core.Logger) (*DockerHandler, error) {
	if len(dockerFilters) > 0 && !configsFromLabels {
		return nil, fmt.Errorf("docker filters can only be provided together with '--docker' flag")
	}

	c := &DockerHandler{
		filters:           dockerFilters,
		configsFromLabels: configsFromLabels,
		notifier:          config,
		logger:            logger,
	}
	var err error
	c.dockerClient, err = c.buildDockerClient()
	if err != nil {
		return nil, err
	}
	// Do a sanity check on docker
	_, err = c.dockerClient.Info()
	if err != nil {
		return nil, err
	}

	if c.configsFromLabels {
		go c.watch()
	}
	return c, nil
}

func (c *DockerHandler) ConfigFromLabelsEnabled() bool {
	return c.configsFromLabels
}

func (c *DockerHandler) watch() {
	const pollInterval = 10 * time.Second
	c.logger.Debugf("Watching for Docker labels changes every %s...", pollInterval)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for range ticker.C {
		labels, err := c.GetDockerLabels()
		// Do not print or care if there is no container up right now
		if err != nil && !errors.Is(err, errNoContainersMatchingFilters) {
			c.logger.Debugf("%v", err)
		}
		c.notifier.dockerLabelsUpdate(labels)
	}
}

func (c *DockerHandler) WaitForLabels() {
	const maxRetries = 3
	const retryDelay = 1 * time.Second
	const dockerEnvFile = "/.dockerenv"
	const mountinfoFilePath = "/proc/self/mountinfo"

	// Check if .dockerenv file exists
	if _, err := os.Stat(dockerEnvFile); os.IsNotExist(err) {
		c.logger.Debugf(".dockerenv file not found, ofelia is not running in a Docker container")
		return
	}

	id, err := getContainerID(mountinfoFilePath)
	if err != nil {
		c.logger.Debugf("Failed to extract ofelia's container ID. Trying with container hostname instead...")
		id, _ = os.Hostname()
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err := c.dockerClient.InspectContainerWithOptions(docker.InspectContainerOptions{ID: id})
		if err == nil {
			c.logger.Debugf("Found ofelia container with ID: %s", id)
			return
		}

		time.Sleep(retryDelay)
	}
}

func (c *DockerHandler) GetDockerLabels() (map[string]map[string]string, error) {
	var filters = map[string][]string{
		"label": {requiredLabelFilter},
	}
	for _, f := range c.filters {
		key, value, err := parseFilter(f)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, f)
		}
		filters[key] = append(filters[key], value)
	}

	conts, err := c.dockerClient.ListContainers(docker.ListContainersOptions{Filters: filters})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedToListContainers, err)
	} else if len(conts) == 0 {
		return nil, fmt.Errorf("%w: %v", errNoContainersMatchingFilters, filters)
	}

	var labels = make(map[string]map[string]string)

	for _, cont := range conts {
		if len(cont.Names) > 0 && len(cont.Labels) > 0 {
			name := strings.TrimPrefix(cont.Names[0], "/")
			for k := range cont.Labels {
				// remove all not relevant labels
				if !strings.HasPrefix(k, labelPrefix) {
					delete(cont.Labels, k)
					continue
				}
			}

			labels[name] = cont.Labels
		}
	}

	return labels, nil
}

func parseFilter(filter string) (key, value string, err error) {
	parts := strings.SplitN(filter, "=", 2)
	if len(parts) != 2 {
		return "", "", errInvalidDockerFilter
	}
	return parts[0], parts[1], nil
}

func (c *Config) buildFromDockerLabels(labels map[string]map[string]string) error {
	execJobs := make(map[string]map[string]interface{})
	localJobs := make(map[string]map[string]interface{})
	runJobs := make(map[string]map[string]interface{})
	serviceJobs := make(map[string]map[string]interface{})
	globalConfigs := make(map[string]interface{})

	for c, l := range labels {
		isServiceContainer := func() bool {
			for k, v := range l {
				if k == serviceLabel {
					return v == "true"
				}
			}
			return false
		}()

		for k, v := range l {
			parts := strings.Split(k, ".")
			if len(parts) < 4 {
				if isServiceContainer {
					globalConfigs[parts[1]] = v
				}

				continue
			}

			jobType, jobName, jopParam := parts[1], parts[2], parts[3]
			switch {
			case jobType == jobExec: // only job exec can be provided on the non-service container
				if _, ok := execJobs[jobName]; !ok {
					execJobs[jobName] = make(map[string]interface{})
				}

				setJobParam(execJobs[jobName], jopParam, v)
				// since this label was placed not on the service container
				// this means we need to `exec` command in this container
				if !isServiceContainer {
					execJobs[jobName]["container"] = c
				}
			case jobType == jobLocal && isServiceContainer:
				if _, ok := localJobs[jobName]; !ok {
					localJobs[jobName] = make(map[string]interface{})
				}
				setJobParam(localJobs[jobName], jopParam, v)
			case jobType == jobServiceRun && isServiceContainer:
				if _, ok := serviceJobs[jobName]; !ok {
					serviceJobs[jobName] = make(map[string]interface{})
				}
				setJobParam(serviceJobs[jobName], jopParam, v)
			case jobType == jobRun && isServiceContainer:
				if _, ok := runJobs[jobName]; !ok {
					runJobs[jobName] = make(map[string]interface{})
				}
				setJobParam(runJobs[jobName], jopParam, v)
			default:
				// TODO: warn about unknown parameter
			}
		}
	}

	if len(globalConfigs) > 0 {
		if err := mapstructure.WeakDecode(globalConfigs, &c.Global); err != nil {
			return err
		}
	}

	if len(execJobs) > 0 {
		if err := mapstructure.WeakDecode(execJobs, &c.ExecJobs); err != nil {
			return err
		}
	}

	if len(localJobs) > 0 {
		if err := mapstructure.WeakDecode(localJobs, &c.LocalJobs); err != nil {
			return err
		}
	}

	if len(serviceJobs) > 0 {
		if err := mapstructure.WeakDecode(serviceJobs, &c.ServiceJobs); err != nil {
			return err
		}
	}

	if len(runJobs) > 0 {
		if err := mapstructure.WeakDecode(runJobs, &c.RunJobs); err != nil {
			return err
		}
	}

	return nil
}

func setJobParam(params map[string]interface{}, paramName, paramVal string) {
	switch strings.ToLower(paramName) {
	case "volume", "environment", "volumes-from":
		arr := []string{} // allow providing JSON arr of volume mounts
		if err := json.Unmarshal([]byte(paramVal), &arr); err == nil {
			params[paramName] = arr
			return
		}
	}

	params[paramName] = paramVal
}

func getContainerID(mountinfoFilePath string) (string, error) {
	// Open the mountinfo file
	file, err := os.Open(mountinfoFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Scan the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Look for container ID in the line
		if !strings.Contains(line, "/containers/") {
			continue
		}

		splt := strings.Split(line, "/")
		for i, part := range splt {
			if part == "containers" && len(splt) > i+1 {
				return splt[i+1], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", os.ErrNotExist
}
