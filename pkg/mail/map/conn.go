package mailmap

import (
	"encoding/csv"
	"fmt"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/log"
	"go.uber.org/zap"
	"net"
	"net/mail"
	"strings"
)

/*
   Creation Time: 2021 - Aug - 06
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

// Requests
const (
	ReqGet = "get"
	ReqPut = "put"
)

// Responses
const (
	ResUnavailable = "500"
	ResError       = "400"
	ResSuccess     = "200"
)

func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	r := csv.NewReader(conn)
	r.Comma = ' '
	record, err := r.Read()
	if err != nil {
		log.Warn("got error on read postfix command", zap.Error(err))
		_, _ = fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", ResError))
	}
	if !(len(record) == 2) {
		_, _ = fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", ResError))
		return
	}

	cmd := strings.ToLower(record[0])
	email, err := mail.ParseAddress(record[1])
	if err != nil {
		_, _ = fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", ResError))
		return
	}

	log.Info("MailMap:: Request Received", zap.Strings("Records", record))
	switch cmd {
	case ReqGet:
		s.Get(conn, strings.ToLower(email.Address))
	case ReqPut:
	default:
		_, _ = fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", ResError))
	}
}

func (s *Server) Get(conn net.Conn, email string) {
	emailParts := strings.Split(email, "@")
	if len(emailParts) != 2 {
		fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", ResError))
		return
	}
	placeID := emailParts[0]

	place := s.model.Place.GetByID(placeID, nil)
	if place == nil || place.Privacy.Receptive != nested.PlaceReceptiveExternal {
		_, _ = fmt.Fprintln(conn, fmt.Sprintf("%s Unavailable", ResUnavailable))
		return
	}

	_, _ = fmt.Fprintln(conn, fmt.Sprintf("%s %s", ResSuccess, email))
}
