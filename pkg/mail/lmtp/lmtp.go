package lmtp

import (
	"fmt"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/config"
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
	model    *nested.Manager
	uploader *uploadClient
	pusher   *pusherClient
	s        *smtp.Server
	addr     string
}

func New(model *nested.Manager, addr string) *Server {
	s := &Server{
		model: model,
		addr:  addr,
	}
	if uploader, err := newUploadClient(config.GetString(config.MailUploadBaseURL), config.GetString(config.SystemAPIKey), true); err != nil {
		panic(fmt.Sprintf("could not create uploader client: %v", err))
	} else {
		s.uploader = uploader
	}

	if pusher, err := newPusherClient(config.GetString(config.MailUploadBaseURL), config.GetString(config.SystemAPIKey), true); err != nil {
		panic(fmt.Sprintf("could not create pusher client: %v", err))
	} else {
		s.pusher = pusher
	}

	s.s = smtp.NewServer(s)
	s.s.Addr = addr
	s.s.LMTP = true
	s.s.ReadTimeout = time.Second * 30
	s.s.WriteTimeout = time.Second * 30
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
	return s.addr
}

func (s *Server) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return s.AnonymousLogin(state)
}

func (s *Server) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return &Session{
		hostname:   state.Hostname,
		remoteAddr: state.RemoteAddr.String(),
		model:      s.model,
		uploader:   s.uploader,
		pusher:     s.pusher,
	}, nil
}
