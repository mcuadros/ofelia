package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/fsouza/go-dockerclient/testing"
	"github.com/mcuadros/ofelia/core"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&TestDockerSuit{})

const imageFixture = "ofelia/test-image"

type TestDockerSuit struct {
	server *testing.DockerServer
	client *docker.Client
}

func (s *TestDockerSuit) SetUpTest(c *check.C) {
	var err error
	s.server, err = testing.NewServer("127.0.0.1:0", nil, nil)
	c.Assert(err, check.IsNil)

	s.client, err = docker.NewClient(s.server.URL())
	c.Assert(err, check.IsNil)

	err = core.BuildTestImage(s.client, imageFixture)
	c.Assert(err, check.IsNil)

	os.Setenv("DOCKER_HOST", s.server.URL())
}

func (s *TestDockerSuit) TearDownTest(c *check.C) {
	os.Unsetenv("DOCKER_HOST")
}

func (s *TestDockerSuit) TestLabelsFilterJobsCount(c *check.C) {
	filterLabel := []string{"test_filter_label", "yesssss"}
	containersToStartWithLabels := []map[string]string{
		{
			requiredLabel:  "true",
			filterLabel[0]: filterLabel[1],
			labelPrefix + "." + jobExec + ".job2.schedule":  "* * * * *",
			labelPrefix + "." + jobExec + ".job2.command":   "command2",
			labelPrefix + "." + jobExec + ".job2.container": "container2",
		},
		{
			requiredLabel: "true",
			labelPrefix + "." + jobExec + ".job3.schedule":  "* * * * *",
			labelPrefix + "." + jobExec + ".job3.command":   "command3",
			labelPrefix + "." + jobExec + ".job3.container": "container3",
		},
	}

	_, err := s.startTestContainersWithLabels(containersToStartWithLabels)
	c.Assert(err, check.IsNil)

	scheduler, err := BuildFromDockerLabels("DEBUG", "label="+strings.Join(filterLabel, "="))
	c.Assert(err, check.IsNil)
	c.Assert(scheduler, check.NotNil)

	c.Assert(scheduler.Jobs, check.HasLen, 1)
}

func (s *TestDockerSuit) TestFilterErrorsLabel(c *check.C) {
	containersToStartWithLabels := []map[string]string{
		{
			labelPrefix + "." + jobExec + ".job2.schedule": "schedule2",
			labelPrefix + "." + jobExec + ".job2.command":  "command2",
		},
	}

	_, err := s.startTestContainersWithLabels(containersToStartWithLabels)
	c.Assert(err, check.IsNil)

	{
		scheduler, err := BuildFromDockerLabels("DEBUG")
		c.Assert(errors.Is(err, errNoContainersMatchingFilters), check.Equals, true)
		c.Assert(strings.Contains(err.Error(), requiredLabelFilter), check.Equals, true)
		c.Assert(scheduler, check.IsNil)
	}

	customLabelFilter := []string{"label", "test=123"}
	{
		scheduler, err := BuildFromDockerLabels("DEBUG", strings.Join(customLabelFilter, "="))
		c.Assert(errors.Is(err, errNoContainersMatchingFilters), check.Equals, true)
		c.Assert(err, check.ErrorMatches, fmt.Sprintf(`.*%s:.*%s.*`, "label", requiredLabel))
		c.Assert(err, check.ErrorMatches, fmt.Sprintf(`.*%s:.*%s.*`, customLabelFilter[0], customLabelFilter[1]))
		c.Assert(scheduler, check.IsNil)
	}

	{
		customNameFilter := []string{"name", "test-name"}
		scheduler, err := BuildFromDockerLabels("DEBUG", strings.Join(customLabelFilter, "="), strings.Join(customNameFilter, "="))
		c.Assert(errors.Is(err, errNoContainersMatchingFilters), check.Equals, true)
		c.Assert(err, check.ErrorMatches, fmt.Sprintf(`.*%s:.*%s.*`, "label", requiredLabel))
		c.Assert(err, check.ErrorMatches, fmt.Sprintf(`.*%s:.*%s.*`, customLabelFilter[0], customLabelFilter[1]))
		c.Assert(err, check.ErrorMatches, fmt.Sprintf(`.*%s:.*%s.*`, customNameFilter[0], customNameFilter[1]))
		c.Assert(scheduler, check.IsNil)
	}

	{
		customBadFilter := "label-test"
		scheduler, err := BuildFromDockerLabels("DEBUG", customBadFilter)
		c.Assert(errors.Is(err, errInvalidDockerFilter), check.Equals, true)
		c.Assert(scheduler, check.IsNil)
	}
}

func (s *TestDockerSuit) startTestContainersWithLabels(containerLabels []map[string]string) ([]*docker.Container, error) {
	containers := []*docker.Container{}

	for i := range containerLabels {
		cont, err := s.client.CreateContainer(docker.CreateContainerOptions{
			Name: fmt.Sprintf("ofelia-test%d", i),
			Config: &docker.Config{
				Cmd:    []string{"sleep", "500"},
				Labels: containerLabels[i],
				Image:  imageFixture,
			},
		})
		if err != nil {
			return containers, err
		}

		containers = append(containers, cont)
		if err := s.client.StartContainer(cont.ID, nil); err != nil {
			return containers, err
		}
	}

	return containers, nil
}
