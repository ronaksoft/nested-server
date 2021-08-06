package main

import (
	"encoding/csv"
	"fmt"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/config"
	"git.ronaksoft.com/nested/server/pkg/log"
	"go.uber.org/zap"
	"net"
	"net/mail"
	"os"
	"strings"
)

/*
	MailMap provides postfix the virtual mailbox map. It listens on a port (i.e. Default 237401)
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

var (
	_Nested *nested.Manager
)

func main() {
	listener, err := net.Listen("tcp", ":237401")
	if err != nil {
		fmt.Println(err.Error())
	}

	// Initialize Nested Model
	_Nested, err = nested.NewManager(
		config.GetString(config.InstanceID),
		config.GetString(config.MongoDSN),
		config.GetString(config.RedisDSN),
		config.GetInt(config.DebugLevel),
	)
	if err != nil {
		os.Exit(1)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
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

	log.Debug("MailMap:: Request Received", zap.Strings("Records", record))
	switch cmd {
	case ReqGet:
		Get(conn, strings.ToLower(email.Address))
	case ReqPut:
	default:
		_, _ = fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", ResError))
	}
}

func Get(conn net.Conn, email string) {
	emailParts := strings.Split(email, "@")
	if len(emailParts) != 2 {
		fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", ResError))
		return
	}
	placeID := emailParts[0]


	place := _Nested.Place.GetByID(placeID, nil)
	if place == nil || place.Privacy.Receptive != nested.PlaceReceptiveExternal {
		_, _ = fmt.Fprintln(conn, fmt.Sprintf("%s Unavailable", ResUnavailable))
		return
	}

	_, _ = fmt.Fprintln(conn, fmt.Sprintf("%s %s", ResSuccess, email))
}
