package protocol

import "encoding/json"

type Request interface {
  Datagram

  Command() string
  UserData() Payload
}

type GenericRequest struct {
  Cmd  string
  Data Payload
}

func (r GenericRequest) Type() DatagramType {
  return DATAGRAM_TYPE_REQUEST
}

func (r GenericRequest) Command() string {
  return r.Cmd
}

func (r GenericRequest) UserData() Payload {
  return r.Data
}

func (r GenericRequest) MarshalJSON() ([]byte, error) {
  return json.Marshal(map[string]interface{}{
    "type": r.Type(),
    "cmd": r.Command(),
    "data": r.UserData(),
  })
}

func (r *GenericRequest) UnmarshalJSON(data []byte) error {
  var raw map[string]interface{}
  if err := json.Unmarshal(data, &raw); err != nil {
    return err
  }

  r.Cmd, _ = raw["cmd"].(string)
  if v, ok := raw["data"].(map[string]interface{}); ok {
    r.Data = D(v)
  } else {
    r.Data = D{}
  }

  return nil
}

func NewRequest(command string, userData Payload) GenericRequest {
  return GenericRequest{
    Cmd: command,
    Data: userData,
  }
}

type StreamRequest struct {
  Cmd   string
  ReqId string
  Data  Payload
}

func (r StreamRequest) Type() DatagramType {
  return DATAGRAM_TYPE_REQUEST
}

func (r StreamRequest) Command() string {
  return r.Cmd
}

func (r StreamRequest) UserData() Payload {
  return r.Data
}

func (r StreamRequest) RequestId() string {
  return r.ReqId
}

func (r StreamRequest) MarshalJSON() ([]byte, error) {
  return json.Marshal(map[string]interface{}{
    "type": r.Type(),
    "cmd": r.Command(),
    "data": r.UserData(),
    "_reqid": r.RequestId(),
  })
}

func (r *StreamRequest) UnmarshalJSON(data []byte) error {
  var raw map[string]interface{}
  if err := json.Unmarshal(data, &raw); err != nil {
    return err
  }

  r.Cmd, _ = raw["cmd"].(string)
  r.ReqId, _ = raw["_reqid"].(string)
  if v, ok := raw["data"].(map[string]interface{}); ok {
    r.Data = D(v)
  } else {
    r.Data = D{}
  }

  return nil
}

func NewStreamRequest(command string, userData Payload, reqId string) StreamRequest {
  return StreamRequest{
    ReqId: reqId,
    Cmd: command,
    Data: userData,
  }
}

type GenericPush struct {
  Cmd  string
  Data Payload
}

func (r GenericPush) Type() DatagramType {
  return DATAGRAM_TYPE_PUSH
}

func (r GenericPush) Command() string {
  return r.Cmd
}

func (r GenericPush) UserData() Payload {
  return r.Data
}

func (p GenericPush) MarshalJSON() ([]byte, error) {
  return json.Marshal(map[string]interface{}{
    "type": p.Type(),
    "cmd": p.Command(),
    "data": p.UserData(),
  })
}

func (p *GenericPush) UnmarshalJSON(data []byte) error {
  var raw map[string]interface{}
  if err := json.Unmarshal(data, &raw); err != nil {
    return err
  }

  p.Cmd, _ = raw["cmd"].(string)
  if v, ok := raw["data"].(map[string]interface{}); ok {
    p.Data = D(v)
  } else {
    p.Data = D{}
  }

  return nil
}

func NewPush(command string, userData Payload) GenericPush {
  return GenericPush{
    Cmd: command,
    Data: userData,
  }
}
