package cli

import (
	"errors"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mitchellh/mapstructure"
)

const (
	labelPrefix = "ofelia"

	requiredLabel       = labelPrefix + ".enabled"
	requiredLabelFilter = requiredLabel + "=true"
	serviceLabel        = labelPrefix + ".service"
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

func (c *Config) buildFromDockerLabels(labels map[string]map[string]string) error {
	execJobs := make(map[string]map[string]string)
	localJobs := make(map[string]map[string]string)
	runJobs := make(map[string]map[string]string)
	serviceJobs := make(map[string]map[string]string)
	globalConfigs := make(map[string]string)

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
					execJobs[jobName] = make(map[string]string)
				}
				execJobs[jobName][jopParam] = v

				// since this label was placed not on the service container
				// this means we need to `exec` command in this container
				if !isServiceContainer {
					execJobs[jobName]["container"] = c
				}
			case jobType == jobLocal && isServiceContainer:
				if _, ok := localJobs[jobName]; !ok {
					localJobs[jobName] = make(map[string]string)
				}
				localJobs[jobName][jopParam] = v
			case jobType == jobServiceRun && isServiceContainer:
				if _, ok := serviceJobs[jobName]; !ok {
					serviceJobs[jobName] = make(map[string]string)
				}
				serviceJobs[jobName][jopParam] = v
			case jobType == jobRun && isServiceContainer:
				if _, ok := runJobs[jobName]; !ok {
					runJobs[jobName] = make(map[string]string)
				}
				runJobs[jobName][jopParam] = v
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
