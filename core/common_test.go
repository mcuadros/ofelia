package core

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/registry"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteCommon struct{}

var _ = Suite(&SuiteCommon{})

func (s *SuiteCommon) TestNewContext(c *C) {
	h := NewScheduler(&TestLogger{})
	j := &TestJob{}
	j.Use(&TestMiddleware{})

	e := NewExecution()

	ctx := NewContext(h, j, e)
	c.Assert(ctx.Scheduler, DeepEquals, h)
	c.Assert(ctx.Job, DeepEquals, j)
	c.Assert(ctx.Execution, DeepEquals, e)
	c.Assert(ctx.middlewares, HasLen, 1)
}

func (s *SuiteCommon) TestContextNextError(c *C) {
	mA := &TestMiddlewareAltA{}
	mB := &TestMiddlewareAltB{}
	mC := &TestMiddlewareAltC{}
	mB.Error, mC.Error = fmt.Errorf("foo"), fmt.Errorf("foo")

	j := &TestJob{}
	j.Use(mA, mB, mC)

	e := NewExecution()

	h := NewScheduler(&TestLogger{})
	ctx := NewContext(h, j, e)
	ctx.Start()

	err := ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mA.Called, Equals, 1)
	c.Assert(mB.Called, Equals, 0)
	c.Assert(mC.Called, Equals, 0)
	c.Assert(j.Called, Equals, 0)
	c.Assert(ctx.Execution.IsRunning, Equals, true)

	err = ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mB.Called, Equals, 1)
	c.Assert(mC.Called, Equals, 0)
	c.Assert(j.Called, Equals, 0)
	c.Assert(ctx.Execution.IsRunning, Equals, false)

	err = ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mC.Called, Equals, 0)
	c.Assert(j.Called, Equals, 0)

	err = ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(j.Called, Equals, 0)
}

func (s *SuiteCommon) TestContextNextNested(c *C) {
	mA := &TestMiddlewareAltA{}
	mB := &TestMiddlewareAltB{}
	mC := &TestMiddlewareAltC{}
	mA.Nested, mB.Nested, mC.Nested = true, true, true

	j := &TestJob{}
	j.Use(mA, mB, mC)

	e := NewExecution()

	h := NewScheduler(&TestLogger{})
	ctx := NewContext(h, j, e)
	ctx.Start()

	err := ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mA.Called, Equals, 1)
	c.Assert(mB.Called, Equals, 1)
	c.Assert(mC.Called, Equals, 1)
	c.Assert(j.Called, Equals, 1)
}

func (s *SuiteCommon) TestContextNextNestedError(c *C) {
	mA := &TestMiddlewareAltA{}
	mB := &TestMiddlewareAltB{}
	mC := &TestMiddlewareAltC{}
	mA.Nested, mB.Nested, mC.Nested = true, true, true
	mA.Stop = errors.New("foo")

	j := &TestJob{}
	j.Use(mA, mB, mC)

	e := NewExecution()

	h := NewScheduler(&TestLogger{})
	ctx := NewContext(h, j, e)
	ctx.Start()

	err := ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mA.Called, Equals, 1)
	c.Assert(mB.Called, Equals, 0)
	c.Assert(mC.Called, Equals, 0)
	c.Assert(j.Called, Equals, 0)
}

func (s *SuiteCommon) TestContextNextContinueOnStop(c *C) {
	mA := &TestMiddlewareAltA{}
	mB := &TestMiddlewareAltB{}
	mC := &TestMiddlewareAltC{}
	mA.Nested, mB.Nested, mC.Nested = true, true, true
	mA.Stop = errors.New("foo")
	mC.OnStop = true

	j := &TestJob{}
	j.Use(mA, mB, mC)

	e := NewExecution()

	h := NewScheduler(&TestLogger{})
	ctx := NewContext(h, j, e)
	ctx.Start()

	err := ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mA.Called, Equals, 1)
	c.Assert(mB.Called, Equals, 0)
	c.Assert(mC.Called, Equals, 1)
	c.Assert(j.Called, Equals, 0)
}

func (s *SuiteCommon) TestContextNext(c *C) {
	mA := &TestMiddlewareAltA{}
	mB := &TestMiddlewareAltB{}
	mC := &TestMiddlewareAltC{}

	j := &TestJob{}
	j.Use(mA, mB, mC)

	e := NewExecution()

	h := NewScheduler(&TestLogger{})
	ctx := NewContext(h, j, e)
	ctx.Start()

	err := ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mA.Called, Equals, 1)
	c.Assert(mB.Called, Equals, 0)
	c.Assert(mC.Called, Equals, 0)
	c.Assert(j.Called, Equals, 0)
	c.Assert(ctx.Execution.IsRunning, Equals, true)

	err = ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mB.Called, Equals, 1)
	c.Assert(mC.Called, Equals, 0)
	c.Assert(j.Called, Equals, 0)
	c.Assert(ctx.Execution.IsRunning, Equals, true)

	err = ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(mC.Called, Equals, 1)
	c.Assert(j.Called, Equals, 0)

	err = ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(j.Called, Equals, 1)

	err = ctx.Next()
	c.Assert(err, IsNil)
	c.Assert(j.Called, Equals, 1)
}

func (s *SuiteCommon) TestExecutionStart(c *C) {
	exe := &Execution{}
	exe.Start()

	c.Assert(exe.IsRunning, Equals, true)
	c.Assert(exe.Date.IsZero(), Equals, false)
}

func (s *SuiteCommon) TestExecutionStop(c *C) {
	exe := &Execution{}
	exe.Start()
	exe.Stop(nil)

	c.Assert(exe.IsRunning, Equals, false)
	c.Assert(exe.Failed, Equals, false)
	c.Assert(exe.Skipped, Equals, false)
	c.Assert(exe.Error, Equals, nil)
	c.Assert(exe.Duration.Seconds() > .0, Equals, true)
}

func (s *SuiteCommon) TestExecutionStopError(c *C) {
	err := errors.New("foo")

	exe := &Execution{}
	exe.Start()
	exe.Stop(err)

	c.Assert(exe.IsRunning, Equals, false)
	c.Assert(exe.Failed, Equals, true)
	c.Assert(exe.Skipped, Equals, false)
	c.Assert(exe.Error, Equals, err)
	c.Assert(exe.Duration > 0, Equals, true)
}

func (s *SuiteCommon) TestExecutionStopErrorSkip(c *C) {
	exe := &Execution{}
	exe.Start()
	exe.Stop(ErrSkippedExecution)

	c.Assert(exe.IsRunning, Equals, false)
	c.Assert(exe.Failed, Equals, false)
	c.Assert(exe.Skipped, Equals, true)
	c.Assert(exe.Error, Equals, nil)
	c.Assert(exe.Duration > 0, Equals, true)
}

func (s *SuiteCommon) TestMiddlewareContainerUseTwice(c *C) {
	mA := &TestMiddleware{}
	mB := &TestMiddleware{}

	container := &middlewareContainer{}
	container.Use(mA)
	container.Use(mB)

	ms := container.Middlewares()
	c.Assert(ms, HasLen, 1)
	c.Assert(ms[0], Equals, mA)
}

func (s *SuiteCommon) TestMiddlewareContainerUseNil(c *C) {
	var m Middleware

	container := &middlewareContainer{}
	container.Use(m)

	ms := container.Middlewares()
	c.Assert(ms, HasLen, 0)
}

func (s *SuiteCommon) TestMiddlewareContainerUseOder(c *C) {
	mA := &TestMiddleware{}
	mB := &TestMiddlewareAltA{}

	container := &middlewareContainer{}
	container.Use(mB)
	container.Use(mA)

	ms := container.Middlewares()
	c.Assert(ms, HasLen, 2)
	c.Assert(ms[0], Equals, mB)
	c.Assert(ms[1], Equals, mA)
}

type TestMiddleware struct {
	Called int
	Nested bool
	OnStop bool
	Error  error
	Stop   error
}

func (m *TestMiddleware) ContinueOnStop() bool {
	return m.OnStop
}

func (m *TestMiddleware) Run(ctx *Context) error {
	m.Called++

	if m.Stop != nil {
		ctx.Execution.Stop(m.Stop)
	}

	if m.Nested {
		ctx.Next()
	}

	return m.Error
}

type TestMiddlewareAltA struct{ TestMiddleware }
type TestMiddlewareAltB struct{ TestMiddleware }
type TestMiddlewareAltC struct{ TestMiddleware }

type TestJob struct {
	BareJob
	Called int
}

func (j *TestJob) Run(ctx *Context) error {
	j.Called++
	time.Sleep(time.Millisecond * 500)

	return nil
}

type TestLogger struct{}

func (*TestLogger) Debug(format string, args ...any)   {}
func (*TestLogger) Error(format string, args ...any)   {}
func (*TestLogger) Info(format string, args ...any)    {}
func (*TestLogger) Warning(format string, args ...any) {}

func (s *SuiteCommon) TestParseRegistry(c *C) {
	c.Assert(parseRegistry("example.com:port/dir/image"), Equals, "example.com:port")
	c.Assert(parseRegistry("example.com:port/image"), Equals, "example.com:port")
	c.Assert(parseRegistry("dir/image"), Equals, "")
	c.Assert(parseRegistry("image"), Equals, "")
}

func (s *SuiteCommon) TestBuildEncodedAuthFromDockerConfig(c *C) {
	dir, cleanup := tempDir(c)
	defer cleanup()
	defer withEnv("DOCKER_CONFIG", dir)()
	resetDockerAuthCache()
	defer resetDockerAuthCache()

	encodedUserPass := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{
		"auths": {
			"registry.example.com": {
				"auth": "`+encodedUserPass+`",
				"email": "me@example.com"
			}
		}
	}`), 0600)
	c.Assert(err, IsNil)

	encodedAuth := buildEncodedAuth("registry.example.com")
	c.Assert(encodedAuth, Not(Equals), "")

	auth, err := registry.DecodeAuthConfig(encodedAuth)
	c.Assert(err, IsNil)
	c.Assert(auth.Username, Equals, "user")
	c.Assert(auth.Password, Equals, "pass")
	c.Assert(auth.Email, Equals, "me@example.com")
	c.Assert(auth.ServerAddress, Equals, "registry.example.com")
}

func (s *SuiteCommon) TestBuildEncodedAuthFromPlaintextPasswordsBeforeConfig(c *C) {
	dir, cleanup := tempDir(c)
	defer cleanup()
	defer withEnv("DOCKER_CONFIG", dir)()
	resetDockerAuthCache()
	defer resetDockerAuthCache()

	firstAuth := base64.StdEncoding.EncodeToString([]byte("first:first-pass"))
	secondAuth := base64.StdEncoding.EncodeToString([]byte("second:second-pass"))
	err := os.WriteFile(filepath.Join(dir, "plaintext-passwords.json"), []byte(`{
		"auths": {
			"registry.example.com": {"auth": "`+firstAuth+`"}
		}
	}`), 0600)
	c.Assert(err, IsNil)

	err = os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{
		"auths": {
			"registry.example.com": {"auth": "`+secondAuth+`"}
		}
	}`), 0600)
	c.Assert(err, IsNil)

	auth := buildAuthConfig("registry.example.com")
	c.Assert(auth.Username, Equals, "first")
	c.Assert(auth.Password, Equals, "first-pass")
}

func (s *SuiteCommon) TestBuildEncodedAuthFromLegacyDockerCfg(c *C) {
	home, cleanup := tempDir(c)
	defer cleanup()
	defer withEnv("DOCKER_CONFIG", "")()
	defer withEnv("HOME", home)()
	resetDockerAuthCache()
	defer resetDockerAuthCache()

	encodedUserPass := base64.StdEncoding.EncodeToString([]byte("legacy:secret"))
	err := os.WriteFile(filepath.Join(home, ".dockercfg"), []byte(`{
		"legacy.example.com": {"auth": "`+encodedUserPass+`"}
	}`), 0600)
	c.Assert(err, IsNil)

	auth := buildAuthConfig("legacy.example.com")
	c.Assert(auth.Username, Equals, "legacy")
	c.Assert(auth.Password, Equals, "secret")
	c.Assert(auth.ServerAddress, Equals, "legacy.example.com")
}

func (s *SuiteCommon) TestBuildEncodedAuthDockerHubFallback(c *C) {
	dir, cleanup := tempDir(c)
	defer cleanup()
	defer withEnv("DOCKER_CONFIG", dir)()
	resetDockerAuthCache()
	defer resetDockerAuthCache()

	encodedUserPass := base64.StdEncoding.EncodeToString([]byte("hub:token"))
	err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{
		"auths": {
			"https://index.docker.io/v2/": {"auth": "`+encodedUserPass+`"}
		}
	}`), 0600)
	c.Assert(err, IsNil)

	auth := buildAuthConfig("")
	c.Assert(auth.Username, Equals, "hub")
	c.Assert(auth.Password, Equals, "token")
}

func (s *SuiteCommon) TestBuildEncodedAuthMissingConfig(c *C) {
	dir, cleanup := tempDir(c)
	defer cleanup()
	defer withEnv("DOCKER_CONFIG", dir)()
	resetDockerAuthCache()
	defer resetDockerAuthCache()

	c.Assert(buildEncodedAuth("missing.example.com"), Equals, "")
}

func resetDockerAuthCache() {
	dockerAuthOnce = sync.Once{}
	dockerAuth = nil
}

func tempDir(c *C) (string, func()) {
	dir, err := os.MkdirTemp("", "ofelia-test-")
	c.Assert(err, IsNil)
	return dir, func() {
		c.Assert(os.RemoveAll(dir), IsNil)
	}
}

func withEnv(key, value string) func() {
	oldValue, existed := os.LookupEnv(key)
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
	return func() {
		if existed {
			os.Setenv(key, oldValue)
		} else {
			os.Unsetenv(key)
		}
	}
}
