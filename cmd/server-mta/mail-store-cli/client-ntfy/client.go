package client_ntfy

import (
	"fmt"
	"github.com/nats-io/go-nats"
)

// NtfyClient
type NtfyClient struct {
	Address string
	Nat     *nats.Conn
	Domain  string
}

// NewNtfyClient
func NewNtfyClient(address, domain string) *NtfyClient {
	c := new(NtfyClient)
	c.Address = address
	c.Domain = domain
	if nat, err := nats.Connect(address); err != nil {
		fmt.Println("NTFY::Client::NewClient::Error::", "address", address, err.Error())
		return nil
	} else {
		c.Nat = nat
	}
	return c
}
