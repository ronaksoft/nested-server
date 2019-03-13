package protocol

import "encoding/json"

type Response interface {
	Datagram

	Status() ResponseStatus
	Data() Payload
}

type ResponseStatus string

const (
	STATUS_SUCCESS ResponseStatus = "ok"
	STATUS_FAILURE ResponseStatus = "err"
)

type GenericResponse struct {
	Code    ResponseStatus
	Payload Payload
}

func (r GenericResponse) Type() DatagramType {
	return DATAGRAM_TYPE_RESPONSE
}

func (r GenericResponse) Status() ResponseStatus {
	return r.Code
}

func (r GenericResponse) Data() Payload {
	return r.Payload
}

func (r GenericResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":   r.Type(),
		"status": r.Status(),
		"data":   r.Data(),
	})
}

func (r *GenericResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if v, ok := raw["status"].(string); ok {
		r.Code = ResponseStatus(v)
	} else {
		r.Code = STATUS_FAILURE
	}

	switch r.Code {
	case STATUS_FAILURE:
		var err Error
		b, _ := json.Marshal(raw["data"])
		if nil == json.Unmarshal(b, &err) {
			r.Payload = err
		} else {
			r.Payload = NewError(ERROR_UNKNOWN, nil)
		}

	default:
		if v, ok := raw["data"].(map[string]interface{}); ok {
			r.Payload = D(v)
		} else {
			r.Payload = D{}
		}
	}

	return nil
}

func NewResponse(status ResponseStatus, data Payload) GenericResponse {
	return GenericResponse{
		Code:    status,
		Payload: data,
	}
}

type StreamResponse struct {
	Code    ResponseStatus
	Payload Payload
	ReqId   string
}

func (r StreamResponse) Type() DatagramType {
	return DATAGRAM_TYPE_RESPONSE
}

func (r StreamResponse) Status() ResponseStatus {
	return r.Code
}

func (r StreamResponse) Data() Payload {
	return r.Payload
}

func (r StreamResponse) RequestId() string {
	return r.ReqId
}

func (r StreamResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":   r.Type(),
		"status": r.Status(),
		"data":   r.Data(),
		"_reqid": r.RequestId(),
	})
}

func (r *StreamResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.ReqId, _ = raw["_reqid"].(string)

	if v, ok := raw["status"].(string); ok {
		r.Code = ResponseStatus(v)
	} else {
		r.Code = STATUS_FAILURE
	}

	switch r.Code {
	case STATUS_FAILURE:
		var err Error
		b, _ := json.Marshal(raw["data"])
		if nil == json.Unmarshal(b, &err) {
			r.Payload = err
		} else {
			r.Payload = NewError(ERROR_UNKNOWN, nil)
		}

	default:
		if v, ok := raw["data"].(map[string]interface{}); ok {
			r.Payload = D(v)
		} else {
			r.Payload = D{}
		}
	}

	return nil
}

func NewStreamResponse(status ResponseStatus, data Payload, reqId string) StreamResponse {
	return StreamResponse{
		ReqId:   reqId,
		Code:    status,
		Payload: data,
	}
}
