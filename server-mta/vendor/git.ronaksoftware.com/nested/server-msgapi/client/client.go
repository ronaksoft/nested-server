package msgapi

import (
  "github.com/nats-io/go-nats"
)

const (
  SUBJECT = "MSGAPI.V1"
)

type Client struct {
  conn  *nats.EncodedConn

  Email  *cEmail
}

func NewClient(Address string) (*Client, error) {
  c := &Client{}

  if conn, err := nats.Connect(Address); err != nil {
    return nil, err
  } else if encc, err := nats.NewEncodedConn(conn, nats.JSON_ENCODER); err != nil {
    return nil, err
  } else {
    c.conn = encc
  }

  c.Email, _ = newEmailClient(c)

  return c, nil
}
