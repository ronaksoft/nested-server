package protocol

type DatagramType string

const (
	DATAGRAM_TYPE_REQUEST  DatagramType = "q"
	DATAGRAM_TYPE_RESPONSE DatagramType = "r"
	DATAGRAM_TYPE_PUSH     DatagramType = "p"
)

type Datagram interface {
	Type() DatagramType
}
