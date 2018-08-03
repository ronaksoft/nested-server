package rpc

import (
    "time"
    "github.com/gogo/protobuf/proto"
    "github.com/mailru/easyjson"
)

/*
    Creation Time: 2018 - Apr - 07
    Created by:  Ehsan N. Moosa (ehsan)
    Maintainers:
        1.  Ehsan N. Moosa (ehsan)
    Auditor: Ehsan N. Moosa
    Copyright Ronak Software Group 2018
*/

// MessageConstructor is an uint32 number which identifies a message type.
// All the requests with same MessageConstructor must have same keys in general.
type MessageConstructor string

// MessageDirection identifies if the message sent by server to client or wise versa.
type MessageDirection byte

const (
    SERVER_TO_CLIENT MessageDirection = 0x01
    CLIENT_TO_SERVER MessageDirection = 0x02
)

// Message
type Message struct {
    Constructor MessageConstructor
    Direction   MessageDirection
    ID          string
    CreatedAt   time.Time
    Data        MessageStream
}

func NewServerMessage(constructor MessageConstructor, id string, data MessageStream) Message {
    return Message{
        CreatedAt:   time.Now(),
        Direction:   SERVER_TO_CLIENT,
        Constructor: constructor,
        ID:          id,
        Data:        data,
    }
}

func NewClientMessage(constructor MessageConstructor, id string, data MessageStream) Message {
    return Message{
        CreatedAt:   time.Now(),
        Direction:   CLIENT_TO_SERVER,
        Constructor: constructor,
        ID:          id,
        Data:        data,
    }
}

// MessageStream is an interface defined for incoming serialized formats to unmarshal
// to appropriate objects.
type MessageStream interface {
    Bytes() []byte
    UnMarshal(v interface{}) error
}

// JsonMessageStream
type JsonMessageStream []byte

func NewJsonMessageStream(v easyjson.Marshaler) (JsonMessageStream, error) {
    b, err := easyjson.Marshal(v)
    if err != nil {
        return nil, err
    }
    return JsonMessageStream(b), nil
}

// UnMarshal
// v must be 'easyjson.Unmarshaler' otherwise it panics
func (j JsonMessageStream) UnMarshal(v interface{}) error {
    return easyjson.Unmarshal([]byte(j), v.(easyjson.Unmarshaler))
}

func (j JsonMessageStream) Bytes() []byte {
    return []byte(j)
}

// ProtoMessageStream
type ProtoMessageStream []byte

func NewProtoMessageStream(pb proto.Message) (ProtoMessageStream, error) {
    b, err := proto.Marshal(pb)
    if err != nil {
        return nil, err
    }
    return ProtoMessageStream(b), nil
}

// UnMarshal
// v must be 'proto.Message' otherwise it panics
func (p ProtoMessageStream) UnMarshal(v interface{}) error {
    return proto.Unmarshal([]byte(p), v.(proto.Message))
}

func (p ProtoMessageStream) Bytes() []byte {
    return []byte(p)
}
