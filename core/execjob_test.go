package core

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"net/http"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/fsouza/go-dockerclient/testing"
	. "gopkg.in/check.v1"
)

const ContainerFixture = "test-container"

type SuiteExecJob struct {
	server *testing.DockerServer
	client *docker.Client
}

var _ = Suite(&SuiteExecJob{})

// overwrite version handler, because
// exec configuration Env is only supported in API#1.25 and above
// https://github.com/fsouza/go-dockerclient/blob/0f57349a7248b9b35ad2193ffe70953d5893e2b8/testing/server.go#L1607
func versionDockerHandler(w http.ResponseWriter, r *http.Request) {
	envs := map[string]interface{}{
		"Version":       "1.10.1",
		"Os":            "linux",
		"KernelVersion": "3.13.0-77-generic",
		"GoVersion":     "go1.17.1",
		"GitCommit":     "9e83765",
		"Arch":          "amd64",
		"ApiVersion":    "1.27",
		"BuildTime":     "2015-12-01T07:09:13.444803460+00:00",
		"Experimental":  false,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(envs)

}

func (s *SuiteExecJob) SetUpTest(c *C) {
	var err error
	s.server, err = testing.NewServer("127.0.0.1:0", nil, nil)
	c.Assert(err, IsNil)

	s.server.CustomHandler("/version", http.HandlerFunc(versionDockerHandler))

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
	job.Environment = []string{"test_Key1=value1", "test_Key2=value2"}
	job.User = "foo"
	job.TTY = true

	e := NewExecution()

	err := job.Run(&Context{Execution: e})
	c.Assert(err, IsNil)
	c.Assert(executed, Equals, true)

	container, err := s.client.InspectContainer(ContainerFixture)
	c.Assert(err, IsNil)
	c.Assert(len(container.ExecIDs) > 0, Equals, true)

	exec, err := job.inspectExec()
	c.Assert(err, IsNil)
	c.Assert(exec.ProcessConfig.EntryPoint, Equals, "echo")
	c.Assert(exec.ProcessConfig.Arguments, DeepEquals, []string{"-a", "foo bar"})
	c.Assert(exec.ProcessConfig.User, Equals, "foo")
	c.Assert(exec.ProcessConfig.Tty, Equals, true)
	// no way to check for env :|
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
