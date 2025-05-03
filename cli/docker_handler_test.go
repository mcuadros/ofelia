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

func buildFromDockerLabels(dockerFilters ...string) (*Config, error) {
	mockLogger := &TestLogger{}
	c := &Config{
		sh: core.NewScheduler(mockLogger),
	}

	var err error
	c.dockerHandler, err = NewDockerHandler(c, dockerFilters, true, mockLogger)
	if err != nil {
		return nil, err
	}
	dockerLabels, err := c.dockerHandler.GetDockerLabels()
	if err != nil {
		return nil, err
	}

	c.buildFromDockerLabels(dockerLabels)
	return c, nil
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

	conf, err := buildFromDockerLabels("label=" + strings.Join(filterLabel, "="))
	c.Assert(err, check.IsNil)
	c.Assert(conf.sh, check.NotNil)

	c.Assert(conf.JobsCount(), check.Equals, 1)
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
		conf, err := buildFromDockerLabels()
		c.Assert(strings.Contains(err.Error(), requiredLabelFilter), check.Equals, true)
		c.Assert(conf, check.IsNil)
	}

	customLabelFilter := []string{"label", "test=123"}
	{
		conf, err := buildFromDockerLabels(strings.Join(customLabelFilter, "="))
		c.Assert(errors.Is(err, errNoContainersMatchingFilters), check.Equals, true)
		c.Assert(err, check.ErrorMatches, fmt.Sprintf(`.*%s:.*%s.*`, "label", requiredLabel))
		c.Assert(err, check.ErrorMatches, fmt.Sprintf(`.*%s:.*%s.*`, customLabelFilter[0], customLabelFilter[1]))
		c.Assert(conf, check.IsNil)
	}

	{
		customNameFilter := []string{"name", "test-name"}
		conf, err := buildFromDockerLabels(strings.Join(customLabelFilter, "="), strings.Join(customNameFilter, "="))
		c.Assert(errors.Is(err, errNoContainersMatchingFilters), check.Equals, true)
		c.Assert(err, check.ErrorMatches, fmt.Sprintf(`.*%s:.*%s.*`, "label", requiredLabel))
		c.Assert(err, check.ErrorMatches, fmt.Sprintf(`.*%s:.*%s.*`, customLabelFilter[0], customLabelFilter[1]))
		c.Assert(err, check.ErrorMatches, fmt.Sprintf(`.*%s:.*%s.*`, customNameFilter[0], customNameFilter[1]))
		c.Assert(conf, check.IsNil)
	}

	{
		customBadFilter := "label-test"
		conf, err := buildFromDockerLabels(customBadFilter)
		c.Assert(errors.Is(err, errInvalidDockerFilter), check.Equals, true)
		c.Assert(conf, check.IsNil)
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

func (s *TestDockerSuit) TestGetContainerID(c *check.C) {
	tests := []struct {
		content string
		expect  string
	}{
		{
			content: `
206 205 0:29 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw
207 203 0:67 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
208 203 0:72 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k
209 201 254:1 /docker/containers/test123/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/vda1 rw,discard
210 201 254:1 /docker/containers/test123/hostname /etc/hostname rw,relatime - ext4 /dev/vda1 rw,discard
211 201 254:1 /docker/containers/test123/hosts /etc/hosts rw,relatime - ext4 /dev/vda1 rw,discard
85 203 0:70 /0 /dev/console rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
86 202 0:68 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
87 202 0:68 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
88 202 0:68 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
`,
			expect: "test123",
		},
		{
			content: `
206 205 0:29 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw
207 203 0:67 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
208 203 0:72 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k
209 201 254:1 /var/lib/docker/containers/test123/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/vda1 rw,discard
210 201 254:1 /var/lib/docker/containers/test123/hostname /etc/hostname rw,relatime - ext4 /dev/vda1 rw,discard
211 201 254:1 /var/lib/docker/containers/test123/hosts /etc/hosts rw,relatime - ext4 /dev/vda1 rw,discard
85 203 0:70 /0 /dev/console rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
86 202 0:68 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
87 202 0:68 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
88 202 0:68 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
`,
			expect: "test123",
		},
	}

	for _, tt := range tests {
		tmpFile, _ := os.CreateTemp(os.TempDir(), "mountinfo")
		tmpFile.WriteString(tt.content)
		defer os.Remove(tmpFile.Name())

		id, err := getContainerID(tmpFile.Name())
		c.Assert(err, check.IsNil)
		c.Assert(id, check.Equals, tt.expect)
	}
}
