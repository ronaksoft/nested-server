package router

import (
  "time"
  "bytes"

  "git.ronaksoftware.com/common/server-protocol"
)

type cRouter struct {
  c *Client
}

func newRouterClient(c *Client) (*cRouter, error) {
  pc := new(cRouter)
  pc.c = c

  return pc, nil
}

func BundleId(bundleGroup, bundleIndex string) string {
  var bundleId bytes.Buffer
  bundleId.WriteString(bundleGroup)
  bundleId.WriteRune('-')
  bundleId.WriteString(bundleIndex)

  return bundleId.String()
}

func (pc *cRouter) PrepareAnycast(data protocol.Datagram, subject string, bundleGroup string) protocol.Packet {
  var addr bytes.Buffer

  if pc.c.BundleGroup() != bundleGroup {
    addr.WriteString(ROUTER_SUBJECT_PREFIX)
    addr.WriteRune('.')
    addr.WriteString(bundleGroup)
    addr.WriteRune('.')
  }
  addr.WriteString(subject)

  return protocol.NewPacket(addr.String(), data)
}

func (pc *cRouter) PublishAnycast(data protocol.Datagram, subject string, bundleGroup string) error {
  packet := pc.PrepareAnycast(data, subject, bundleGroup)

  return pc.c.conn.Publish(packet.Address(), packet.Datagram())
}

func (pc *cRouter) RequestAnycast(data protocol.Datagram, subject string, bundleGroup string, vPtr interface{}, timeout time.Duration) error {
  packet := pc.PrepareAnycast(data, subject, bundleGroup)

  return pc.c.conn.Request(packet.Address(), packet.Datagram(), vPtr, timeout)
}

func (pc *cRouter) PrepareUnicast(data protocol.Datagram, subject string, bundleId string) protocol.Packet {
  var addr bytes.Buffer

  if pc.c.BundleId() != bundleId {
    addr.WriteString(ROUTER_SUBJECT_PREFIX)
    addr.WriteRune('.')
    addr.WriteString(bundleId)
    addr.WriteRune('.')
  }
  addr.WriteString(subject)

  return protocol.NewPacket(addr.String(), data)
}

func (pc *cRouter) PublishUnicast(data protocol.Datagram, subject string, bundleId string) error {
  packet := pc.PrepareUnicast(data, subject, bundleId)

  return pc.c.conn.Publish(packet.Address(), packet.Datagram())
}

func (pc *cRouter) RequestUnicast(data protocol.Datagram, subject string, bundleId string, vPtr interface{}, timeout time.Duration) error {
  packet := pc.PrepareUnicast(data, subject, bundleId)

  return pc.c.conn.Request(packet.Address(), packet.Datagram(), vPtr, timeout)
}
