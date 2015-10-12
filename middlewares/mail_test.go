package middlewares

import (
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/bradfitz/go-smtpd/smtpd"

	. "gopkg.in/check.v1"
)

type MailSuite struct {
	BaseSuite

	l         net.Listener
	smtpd     *smtpd.Server
	smtpdHost string
	smtpdPort int
}

var _ = Suite(&MailSuite{})

func (s *MailSuite) SetUpTest(c *C) {
	s.BaseSuite.SetUpTest(c)

	s.smtpd = &smtpd.Server{
		Addr: ":0",
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	c.Assert(err, IsNil)

	s.l = ln
	go func() {
		err := s.smtpd.Serve(ln)
		c.Assert(err, IsNil)
	}()

	p := strings.Split(s.l.Addr().String(), ":")
	s.smtpdHost = p[0]
	s.smtpdPort, _ = strconv.Atoi(p[1])
}

func (s *MailSuite) TearDownTest(c *C) {
	s.l.Close()
}

func (s *MailSuite) TestNewSlackEmpty(c *C) {
	c.Assert(NewMail(&MailConfig{}), IsNil)
}

func (s *MailSuite) TestRunSuccess(c *C) {
	s.ctx.Start()
	s.ctx.Stop(nil)

	m := NewMail(&MailConfig{
		SMTPHost:  s.smtpdHost,
		SMTPPort:  s.smtpdPort,
		EmailTo:   "foo@foo.com",
		EmailFrom: "qux@qux.com",
	})

	var wg sync.WaitGroup
	s.smtpd.OnNewMail = func(_ smtpd.Connection, from smtpd.MailAddress) (smtpd.Envelope, error) {
		c.Assert(from.Email(), Equals, "qux@qux.com")
		wg.Done()

		return nil, nil
	}

	wg.Add(1)
	go func() {
		c.Assert(m.Run(s.ctx), IsNil)
	}()

	wg.Wait()
}
