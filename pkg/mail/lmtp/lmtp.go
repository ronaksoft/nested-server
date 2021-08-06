package lmtp

import (
	"git.ronaksoft.com/nested/server/pkg/log"
	"github.com/emersion/go-smtp"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"os"
	"time"
)

/*
   Creation Time: 2021 - Aug - 06
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type Server struct {
	s *smtp.Server
}

func New(addr string) *Server {
	ss := smtp.NewServer(&Backend{})
	ss.Addr = addr
	ss.LMTP = true
	ss.ReadTimeout = time.Second * 30
	ss.WriteTimeout = time.Second * 30
	return &Server{
		s: ss,
	}
}

func (s *Server) Run() {
	go func() {
		err := s.s.ListenAndServe()
		if err != nil {
			return
		}
	}()
	time.Sleep(time.Second)
	_ = os.Chmod(s.s.Addr, os.ModePerm)
}

func (s *Server) Close() {
	_ = s.s.Close()
}

func (s *Server) Addr() string {
	return s.s.Addr
}

type Backend struct {
	addr string
}

func (s *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return s.AnonymousLogin(state)
}

func (s *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return &Session{
		hostname:   state.Hostname,
		remoteAddr: state.RemoteAddr.String(),
	}, nil
}

type Session struct {
	hostname   string
	remoteAddr string
	from       string
	to         []string
	opts       smtp.MailOptions
}

func (s *Session) Reset() {
	s.opts = smtp.MailOptions{}
	s.from = ""
	s.to = s.to[:0]
	log.Info("Session Reset", zap.String("H", s.hostname), zap.String("Remote", s.remoteAddr))
}

func (s *Session) Logout() error {
	log.Info("Session Logout", zap.String("H", s.hostname), zap.String("Remote", s.remoteAddr))
	return nil
}

func (s *Session) Mail(from string, opts smtp.MailOptions) error {
	s.from = from
	s.opts = opts
	log.Info("Session Mail",
		zap.String("H", s.hostname),
		zap.String("Remote", s.remoteAddr),
		zap.String("From", from),
		zap.Any("MO", opts),
	)
	return nil
}

func (s *Session) Rcpt(to string) error {
	log.Info("Session To", zap.String("H", s.hostname), zap.String("Remote", s.remoteAddr), zap.String("TO", to))
	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	b, err := ioutil.ReadAll(r)
	log.Info("Data", zap.Int("L", len(b)))
	return err
}
