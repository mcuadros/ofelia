package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mitchellh/mapstructure"
)

const (
	labelPrefix = "ofelia"

	requiredLabelName   = labelPrefix + ".enabled"
	requiredLabelFilter = requiredLabelName + "=true"
	serviceLabelName    = labelPrefix + ".service"
)

func getLabels(d *docker.Client) (map[string]map[string]string, error) {
	// sleep before querying containers
	// because docker not always propagating labels in time
	// so ofelia app can't find it's own container
	if IsDockerEnv {
		time.Sleep(1 * time.Second)
	}

	conts, err := d.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"label": {requiredLabelFilter},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(conts) == 0 {
		return nil, fmt.Errorf("couldn't find containers with label '%s'", requiredLabelFilter)
	}

	var labels = make(map[string]map[string]string)

	for _, c := range conts {
		if len(c.Names) > 0 && len(c.Labels) > 0 {
			name := strings.TrimPrefix(c.Names[0], "/")
			for k := range c.Labels {
				// Remove all irrelevant labels
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

func (c *Config) buildFromDockerLabels(labels map[string]map[string]string) error {
	execJobs := make(map[string]map[string]interface{})
	localJobs := make(map[string]map[string]interface{})
	runJobs := make(map[string]map[string]interface{})
	serviceJobs := make(map[string]map[string]interface{})
	globalConfigs := make(map[string]interface{})

	jobTypes := map[string]map[string]map[string]interface{}{
		jobExec:       execJobs,
		jobLocal:      localJobs,
		jobRun:        runJobs,
		jobServiceRun: serviceJobs,
	}

	for containerName, containerLabels := range labels {
		serviceLabelValue, hasServiceLabel := containerLabels[serviceLabelName]
		isServiceContainer := hasServiceLabel && serviceLabelValue == "true"

		for labelName, labelValue := range containerLabels {
			selectors := strings.Split(labelName, ".")

			// Handle short labels
			if len(selectors) < 4 {
				if len(selectors) > 1 && isServiceContainer {
					// Always ignore the third selector of short labels
					// TODO: Add warning
					globalConfigs[selectors[1]] = labelValue
				}

				// Always ignore incomplete labels
				continue
			}

			// The first selector, corresponding to the prefix, is always ignored
			jobType, jobName, jobParam := selectors[1], selectors[2], selectors[3]

			// Only job exec can be provided on the non-service container
			if jobType == jobExec {
				if _, hasJob := execJobs[jobName]; !hasJob {
					execJobs[jobName] = make(map[string]interface{})
				}

				setJobParam(execJobs[jobName], jobParam, labelValue)
				// Since this label was placed not on the service container
				// this means we need to `exec` command in this container
				if !isServiceContainer {
					execJobs[jobName]["container"] = containerName
				}

				continue
			}

			// Handle remaining job types
			if isServiceContainer {
				if jobMap, hasJobMap := jobTypes[jobType]; hasJobMap {
					if _, hasJob := jobMap[jobName]; !hasJob {
						jobMap[jobName] = make(map[string]interface{})
					}
					setJobParam(jobMap[jobName], jobParam, labelValue)
				} else {
					// TODO: Warn about unknown parameter
				}
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
	switch paramName {
	case "volume":
		arr := []string{} // allow providing JSON arr of volume mounts
		if err := json.Unmarshal([]byte(paramVal), &arr); err == nil {
			params[paramName] = arr
			return
		}
	case "environment":
		arr := []string{} // allow providing JSON arr of env keyvalues
		if err := json.Unmarshal([]byte(paramVal), &arr); err == nil {
			params[paramName] = arr
			return
		}
	}

	params[paramName] = paramVal
}
