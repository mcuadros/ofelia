package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/mcuadros/ofelia/core"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
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
	dockerClient      core.DockerClient
	notifier          labelConfigUpdater
	configsFromLabels bool
	logger            core.Logger
	filters           []string
	cancel            context.CancelFunc
}

type labelConfigUpdater interface {
	dockerLabelsUpdate(map[string]map[string]string)
}

func (c *DockerHandler) GetInternalDockerClient() core.DockerClient {
	return c.dockerClient
}

func buildDockerClient() (core.DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return cli, nil
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
	c.dockerClient, err = buildDockerClient()
	if err != nil {
		return nil, err
	}

	_, err = c.dockerClient.Info(context.Background(), client.InfoOptions{})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *DockerHandler) StartWatching() {
	if !c.configsFromLabels || c.cancel != nil {
		return
	}
	if c.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.watch(ctx)
}

func (c *DockerHandler) Close() {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
}

func (c *DockerHandler) ConfigFromLabelsEnabled() bool {
	return c.configsFromLabels
}

func (c *DockerHandler) watch(ctx context.Context) {
	filters := client.Filters{}.
		Add("type", string(events.ContainerEventType))

	for _, f := range c.filters {
		key, value, err := parseFilter(f)
		if err != nil {
			continue
		}
		if key == "label" {
			filters = filters.Add(key, value)
		} else {
			c.logger.Debug("Docker event filter not applicable to event stream, applied only to container list", "filter", f)
		}
	}

	c.logger.Debug("Listening for Docker events to hot reload configs...", "filters", filters)

	for {
		result := c.dockerClient.Events(ctx, client.EventsListOptions{Filters: filters})

		if err := c.consumeEvents(ctx, result); err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Debug("Docker event stream error, reconnecting...", "error", err)
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			}
			c.reconcile(ctx)
		}
	}
}

func (c *DockerHandler) consumeEvents(ctx context.Context, result client.EventsResult) error {
	const debounceInterval = 500 * time.Millisecond
	debounce := time.NewTimer(debounceInterval)
	debounce.Stop()
	defer debounce.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-result.Messages:
			if !ok {
				return io.EOF
			}
			if !isReloadAction(event.Action) {
				continue
			}
			c.logger.Debug("Docker event received",
				"action", event.Action,
				"container", event.Actor.Attributes["name"],
				"id", event.Actor.ID,
			)
			debounce.Reset(debounceInterval)
		case <-debounce.C:
			c.reconcile(ctx)
		case err, ok := <-result.Err:
			if !ok {
				return io.EOF
			}
			return err
		}
	}
}

func (c *DockerHandler) reconcile(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	labels, err := c.GetDockerLabels(ctx)
	if err != nil && !errors.Is(err, errNoContainersMatchingFilters) {
		c.logger.Debug("Failed to reconcile Docker labels", "error", err)
		return
	}
	c.notifier.dockerLabelsUpdate(labels)
}

func isReloadAction(action events.Action) bool {
	switch action {
	case events.ActionCreate,
		events.ActionStart,
		events.ActionRestart,
		events.ActionStop,
		events.ActionKill,
		events.ActionDie,
		events.ActionDestroy,
		events.ActionPause,
		events.ActionUnPause,
		events.ActionRename,
		events.ActionUpdate:
		return true
	default:
		return false
	}
}

func (c *DockerHandler) WaitForLabels() {
	const maxRetries = 3
	const retryDelay = 1 * time.Second
	const dockerEnvFile = "/.dockerenv"
	const mountinfoFilePath = "/proc/self/mountinfo"

	if _, err := os.Stat(dockerEnvFile); os.IsNotExist(err) {
		c.logger.Debug(".dockerenv file not found, ofelia is not running in a Docker container")
		return
	}

	id, err := getContainerID(mountinfoFilePath)
	if err != nil {
		c.logger.Debug("Failed to extract ofelia's container ID. Trying with container hostname instead...")
		id, _ = os.Hostname()
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		_, inspectErr := c.dockerClient.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
		if inspectErr == nil {
			c.logger.Debug("Found ofelia container", "container_id", id)
			return
		}

		time.Sleep(retryDelay)
	}
}

func (c *DockerHandler) GetDockerLabels(ctx context.Context) (map[string]map[string]string, error) {
	filters := client.Filters{}.Add("label", requiredLabelFilter)
	for _, f := range c.filters {
		key, value, err := parseFilter(f)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, f)
		}
		filters = filters.Add(key, value)
	}

	result, err := c.dockerClient.ContainerList(ctx, client.ContainerListOptions{Filters: filters})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedToListContainers, err)
	} else if len(result.Items) == 0 {
		return nil, fmt.Errorf("%w: %v", errNoContainersMatchingFilters, filters)
	}

	var labels = make(map[string]map[string]string)

	for _, cont := range result.Items {
		if len(cont.Names) > 0 && len(cont.Labels) > 0 {
			name := strings.TrimPrefix(cont.Names[0], "/")
			filtered := make(map[string]string)
			for k, v := range cont.Labels {
				if strings.HasPrefix(k, labelPrefix) {
					filtered[k] = v
				}
			}
			labels[name] = filtered
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
			case jobType == jobExec:
				if _, ok := execJobs[jobName]; !ok {
					execJobs[jobName] = make(map[string]interface{})
				}

				setJobParam(execJobs[jobName], jopParam, v)
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
		arr := []string{}
		if err := json.Unmarshal([]byte(paramVal), &arr); err == nil {
			params[paramName] = arr
			return
		}
	}

	params[paramName] = paramVal
}

func getContainerID(mountinfoFilePath string) (string, error) {
	file, err := os.Open(mountinfoFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

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
