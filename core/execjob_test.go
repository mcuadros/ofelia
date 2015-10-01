package core

import (
	"archive/tar"
	"bytes"

	"github.com/fsouza/go-dockerclient"
	"github.com/mcuadros/go-dockerclient/testing"
	. "gopkg.in/check.v1"
)

const ContainerFixture = "test-container"

type SuiteExecJob struct {
	server *testing.DockerServer
	client *docker.Client
}

var _ = Suite(&SuiteExecJob{})

func (s *SuiteExecJob) SetUpTest(c *C) {
	var err error
	s.server, err = testing.NewServer("127.0.0.1:0", nil, nil)
	c.Assert(err, IsNil)

	s.client, err = docker.NewClient(s.server.URL())
	c.Assert(err, IsNil)

	s.buildContainer(c)
}

func (s *SuiteExecJob) TestRun(c *C) {
	var executed bool
	s.server.PrepareExec("*", func() {
		executed = true
	})

	job := &ExecJob{Client: s.client}
	job.Container = ContainerFixture
	job.Command = `echo -a "foo bar"`
	job.User = "foo"
	job.TTY = true

	e := NewExecution()

	err := job.Run(&Context{Execution: e})
	c.Assert(err, IsNil)
	c.Assert(executed, Equals, true)

	container, err := s.client.InspectContainer(ContainerFixture)
	c.Assert(err, IsNil)

	exec, err := s.client.InspectExec(container.ExecIDs[0])
	c.Assert(err, IsNil)
	c.Assert(exec.ProcessConfig.EntryPoint, Equals, "echo")
	c.Assert(exec.ProcessConfig.Arguments, DeepEquals, []string{"-a", "foo bar"})
	c.Assert(exec.ProcessConfig.User, Equals, "foo")
	c.Assert(exec.ProcessConfig.Tty, Equals, true)
}

func (s *SuiteExecJob) buildContainer(c *C) {
	inputbuf := bytes.NewBuffer(nil)
	tr := tar.NewWriter(inputbuf)
	tr.WriteHeader(&tar.Header{Name: "Dockerfile"})
	tr.Write([]byte("FROM base\n"))
	tr.Close()

	err := s.client.BuildImage(docker.BuildImageOptions{
		Name:         "test",
		InputStream:  inputbuf,
		OutputStream: bytes.NewBuffer(nil),
	})
	c.Assert(err, IsNil)

	_, err = s.client.CreateContainer(docker.CreateContainerOptions{
		Name:   ContainerFixture,
		Config: &docker.Config{Image: "test"},
	})

}
