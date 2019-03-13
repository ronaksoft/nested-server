package protocol

import "encoding/json"

type Packet interface {
	Address() string
	Datagram() Datagram
}

type GenericPacket struct {
	Subject string
	Data    Datagram
}

func (p GenericPacket) Address() string {
	return p.Subject
}

func (p GenericPacket) Datagram() Datagram {
	return p.Data
}

func (p GenericPacket) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"subject": p.Subject,
		"data":    p.Data,
	})
}

func (p *GenericPacket) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	p.Subject = raw["subject"].(string)
	p.Data = raw["data"].(Datagram)

	return nil
}

func NewPacket(address string, userData Datagram) GenericPacket {
	return GenericPacket{
		Subject: address,
		Data:    userData,
	}
}
