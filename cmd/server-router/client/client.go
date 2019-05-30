package router

import (
	"errors"
	"strings"

	"github.com/nats-io/go-nats"
)

const (
	ROUTER_SUBJECT_PREFIX string = "ROUTER"
	BUNDLE_ID_SEPERATOR   string = "-"
)

type Client struct {
	conn *nats.EncodedConn

	bundleId    string
	bundleGroup string
	bundleIndex string

	Router *cRouter
}

func NewClient(address, bundleId string) (*Client, error) {
	c := &Client{}

	// Extract BundleGroup & BundleIndex from the BundleID
	if bidSpl := strings.SplitN(bundleId, BUNDLE_ID_SEPERATOR, 2); len(bidSpl) != 2 {
		return nil, errors.New("invalid bundle id")
	} else {
		c.bundleGroup = bidSpl[0]
		c.bundleIndex = bidSpl[1]
		c.bundleId = bundleId
	}

	if conn, err := nats.Connect(address); err != nil {
		return nil, err
	} else if encc, err := nats.NewEncodedConn(conn, nats.JSON_ENCODER); err != nil {
		return nil, err
	} else {
		c.conn = encc
	}

	c.Router, _ = newRouterClient(c)

	return c, nil
}

func (c Client) BundleGroup() string {
	return c.bundleGroup
}

func (c Client) BundleIndex() string {
	return c.bundleIndex
}

func (c Client) BundleId() string {
	return c.bundleId
}
