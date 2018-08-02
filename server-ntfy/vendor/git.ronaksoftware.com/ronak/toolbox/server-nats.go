package ronak

import (
    "github.com/nats-io/go-nats"
    "time"
    "bytes"
)

var (
    NATS_ERR = []byte("ERR")
    NATS_OK  = []byte("OK")
)

type NatsServerConfig struct {
    Name           string
    Username       string
    Password       string
    Servers        []string
    MaxConcurrent  int
    RequestTimeout time.Duration
    RequestRetries int
    OnMessage      func(string, []byte)
}

type NatsServer struct {
    chLimit        chan bool
    conn           *nats.Conn
    requestRetries int
    requestTimeout time.Duration
    onMessage      func(string, []byte)
}

func NewNatsServer(config NatsServerConfig) *NatsServer {
    _funcName := "NewBackGateway"
    natsServer := new(NatsServer)
    options := nats.GetDefaultOptions()
    options.User = config.Username
    options.Password = config.Password
    options.Name = config.Name
    options.Servers = config.Servers
    options.AllowReconnect = true

    if c, err := options.Connect(); err != nil {
        _LOG.Fatal(_funcName, err.Error())
    } else {
        natsServer.conn = c
    }

    // Set a buffered channel for rate limiting
    natsServer.chLimit = make(chan bool, config.MaxConcurrent)

    // Set the message handler for incoming messages
    natsServer.onMessage = config.OnMessage

    // Set the request timeout and number of retries
    natsServer.requestTimeout = config.RequestTimeout

    natsServer.conn.Subscribe(config.Name, func(m *nats.Msg) {
        select {
        case natsServer.chLimit <- true:
            defer func() { <-natsServer.chLimit }()
        case <-time.After(time.Millisecond):
            natsServer.conn.Publish(m.Reply, NATS_ERR)
            return
        }
        natsServer.onMessage(m.Reply, m.Data)
    })

    return natsServer
}

func (s *NatsServer) SendMessage(subject string, bytesOut []byte) bool {
    retries := 0
    success := false
    for retries < s.requestRetries {
        if res, err := s.conn.Request(subject, bytesOut, s.requestTimeout); err == nil && bytes.Equal(res.Data, NATS_OK) {
            success = true
            break
        }
        retries++
    }
    return success
}
