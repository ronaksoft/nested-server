package lmtp

import (
	"git.ronaksoft.com/nested/server/nested"
	"github.com/emersion/go-smtp"
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
	model *nested.Manager
	s     *smtp.Server
	addr  string
}

func New(model *nested.Manager, addr string) *Server {
	s := &Server{
		model: model,
		addr:  addr,
	}
	ss := smtp.NewServer(s)
	ss.Addr = addr
	ss.LMTP = true
	ss.ReadTimeout = time.Second * 30
	ss.WriteTimeout = time.Second * 30
	return s
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

func (s *Server) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return s.AnonymousLogin(state)
}

func (s *Server) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return &Session{
		hostname:   state.Hostname,
		remoteAddr: state.RemoteAddr.String(),
		model:      s.model,
	}, nil
}
