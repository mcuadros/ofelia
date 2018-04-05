package core

import (
	"archive/tar"
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/swarm"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/fsouza/go-dockerclient/testing"
	logging "github.com/op/go-logging"

	. "gopkg.in/check.v1"
)

const ServiceImageFixture = "test-image"

type SuiteRunServiceJob struct {
	server *testing.DockerServer
	client *docker.Client
}

var _ = Suite(&SuiteRunServiceJob{})

const logFormat = "%{color}%{shortfile} ▶ %{level}%{color:reset} %{message}"

var logger Logger

func (s *SuiteRunServiceJob) SetUpTest(c *C) {
	var err error

	logging.SetFormatter(logging.MustStringFormatter(logFormat))

	logger = logging.MustGetLogger("ofelia")
	s.server, err = testing.NewServer("127.0.0.1:0", nil, nil)
	c.Assert(err, IsNil)

	s.client, err = docker.NewClient(s.server.URL())
	c.Assert(err, IsNil)

	s.client.InitSwarm(docker.InitSwarmOptions{})

	s.buildImage(c)
}

func (s *SuiteRunServiceJob) TestRun(c *C) {
	job := &RunServiceJob{Client: s.client}
	job.Image = ServiceImageFixture
	job.Command = `echo -a foo bar`
	job.User = "foo"
	job.TTY = true
	job.Delete = true
	job.Network = "foo"

	e := NewExecution()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		time.Sleep(time.Millisecond * 600)

		tasks, err := s.client.ListTasks(docker.ListTasksOptions{})

		c.Assert(err, IsNil)
		fmt.Printf("found tasks %v\n", tasks[0].Spec.ContainerSpec.Command)

		c.Assert(strings.Join(tasks[0].Spec.ContainerSpec.Command, ","), Equals, "echo,-a,foo,bar")

		c.Assert(tasks[0].Status.State, Equals, swarm.TaskStateReady)

		err = s.client.RemoveService(docker.RemoveServiceOptions{
			ID: tasks[0].ServiceID,
		})

		c.Assert(err, IsNil)

		wg.Done()

	}()

	err := job.Run(&Context{Execution: e, Logger: logger})
	c.Assert(err, IsNil)
	wg.Wait()

	containers, err := s.client.ListTasks(docker.ListTasksOptions{})

	c.Assert(err, IsNil)
	c.Assert(containers, HasLen, 0)
}

func (s *SuiteRunServiceJob) TestBuildPullImageOptionsBareImage(c *C) {
	o, _ := buildPullOptions("foo")
	c.Assert(o.Repository, Equals, "foo")
	c.Assert(o.Tag, Equals, "latest")
	c.Assert(o.Registry, Equals, "")
}

func (s *SuiteRunServiceJob) TestBuildPullImageOptionsVersion(c *C) {
	o, _ := buildPullOptions("foo:qux")
	c.Assert(o.Repository, Equals, "foo")
	c.Assert(o.Tag, Equals, "qux")
	c.Assert(o.Registry, Equals, "")
}

func (s *SuiteRunServiceJob) TestBuildPullImageOptionsRegistry(c *C) {
	o, _ := buildPullOptions("quay.io/srcd/rest:qux")
	c.Assert(o.Repository, Equals, "quay.io/srcd/rest")
	c.Assert(o.Tag, Equals, "qux")
	c.Assert(o.Registry, Equals, "quay.io")
}

func (s *SuiteRunServiceJob) buildImage(c *C) {
	inputbuf := bytes.NewBuffer(nil)
	tr := tar.NewWriter(inputbuf)
	tr.WriteHeader(&tar.Header{Name: "Dockerfile"})
	tr.Write([]byte("FROM base\n"))
	tr.Close()

	err := s.client.BuildImage(docker.BuildImageOptions{
		Name:         ServiceImageFixture,
		InputStream:  inputbuf,
		OutputStream: bytes.NewBuffer(nil),
	})
	c.Assert(err, IsNil)
}
