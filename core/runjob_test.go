package core

import (
	"archive/tar"
	"bytes"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/fsouza/go-dockerclient/testing"
	logging "github.com/op/go-logging"
	. "gopkg.in/check.v1"
)

const ImageFixture = "test-image"

type SuiteRunJob struct {
	server *testing.DockerServer
	client *docker.Client
}

var _ = Suite(&SuiteRunJob{})

func (s *SuiteRunJob) SetUpTest(c *C) {
	var err error
	s.server, err = testing.NewServer("127.0.0.1:0", nil, nil)
	c.Assert(err, IsNil)

	s.client, err = docker.NewClient(s.server.URL())
	c.Assert(err, IsNil)

	s.buildImage(c)
	s.createNetwork(c)
}

func (s *SuiteRunJob) TestRun(c *C) {
	job := &RunJob{Client: s.client}
	job.Image = ImageFixture
	job.Entrypoint = "/bin/bash -c"
	job.Command = `echo -a "foo bar"`
	job.User = "foo"
	job.TTY = true
	job.Delete = "true"
	job.Network = "foo"
	job.Hostname = "test-host"
	job.Name = "test"
	job.Environment = []string{"test_Key1=value1", "test_Key2=value2"}
	job.Volume = []string{"/test/tmp:/test/tmp:ro", "/test/tmp:/test/tmp:rw"}

	ctx := &Context{}
	ctx.Execution = NewExecution()
	logging.SetFormatter(logging.MustStringFormatter(logFormat))
	ctx.Logger = logging.MustGetLogger("ofelia")
	ctx.Job = job

	go func() {
		// Docker Test Server doesn't actually start container
		// so "job.Run" will hang until container is stopped
		if err := job.Run(ctx); err != nil {
			c.Fatal(err)
		}
	}()

	time.Sleep(200 * time.Millisecond)
	container, err := job.getContainer()
	c.Assert(err, IsNil)
	c.Assert(container.Config.Entrypoint, DeepEquals, []string{"/bin/bash", "-c"})
	c.Assert(container.Config.Cmd, DeepEquals, []string{"echo", "-a", "foo bar"})
	c.Assert(container.Config.User, Equals, job.User)
	c.Assert(container.Config.Image, Equals, job.Image)
	c.Assert(container.State.Running, Equals, true)
	c.Assert(container.Config.Env, DeepEquals, job.Environment)

	// this doesn't seem to be working with DockerTestServer
	// c.Assert(container.Config.Hostname, Equals, job.Hostname)
	// c.Assert(container.HostConfig.Binds, DeepEquals, job.Volume)

	// stop container, we don't need it anymore
	err = job.stopContainer(0)
	c.Assert(err, IsNil)

	// wait and double check if container was deleted on "stop"
	time.Sleep(watchDuration * 2)
	container, _ = job.getContainer()
	c.Assert(container, IsNil)

	containers, err := s.client.ListContainers(docker.ListContainersOptions{All: true})
	c.Assert(err, IsNil)
	c.Assert(containers, HasLen, 0)
}

func (s *SuiteRunJob) TestBuildPullImageOptionsBareImage(c *C) {
	o, _ := buildPullOptions("foo")
	c.Assert(o.Repository, Equals, "foo")
	c.Assert(o.Tag, Equals, "latest")
	c.Assert(o.Registry, Equals, "")
}

func (s *SuiteRunJob) TestBuildPullImageOptionsVersion(c *C) {
	o, _ := buildPullOptions("foo:qux")
	c.Assert(o.Repository, Equals, "foo")
	c.Assert(o.Tag, Equals, "qux")
	c.Assert(o.Registry, Equals, "")
}

func (s *SuiteRunJob) TestBuildPullImageOptionsRegistry(c *C) {
	o, _ := buildPullOptions("quay.io/srcd/rest:qux")
	c.Assert(o.Repository, Equals, "quay.io/srcd/rest")
	c.Assert(o.Tag, Equals, "qux")
	c.Assert(o.Registry, Equals, "quay.io")
}

func (s *SuiteRunJob) buildImage(c *C) {
	inputbuf := bytes.NewBuffer(nil)
	tr := tar.NewWriter(inputbuf)
	tr.WriteHeader(&tar.Header{Name: "Dockerfile"})
	tr.Write([]byte("FROM base\n"))
	tr.Close()

	err := s.client.BuildImage(docker.BuildImageOptions{
		Name:         ImageFixture,
		InputStream:  inputbuf,
		OutputStream: bytes.NewBuffer(nil),
	})
	c.Assert(err, IsNil)
}

func (s *SuiteRunJob) createNetwork(c *C) {
	_, err := s.client.CreateNetwork(docker.CreateNetworkOptions{
		Name:   "foo",
		Driver: "bridge",
	})
	c.Assert(err, IsNil)
}
