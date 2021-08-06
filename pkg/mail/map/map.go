package mailmap

import (
	"fmt"
	"git.ronaksoft.com/nested/server/nested"
	"net"
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
}

func New(model *nested.Manager) *Server {
	return &Server{
		model: model,
	}
}

func (s *Server) Run() {
	listener, err := net.Listen("tcp", "127.0.0.1:23741")
	if err != nil {
		fmt.Println(err.Error())
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go s.handleConn(conn)
	}
}
