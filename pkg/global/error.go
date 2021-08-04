package global

import (
	"encoding/json"
)

/*
   Creation Time: 2021 - Aug - 05
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type ErrorCode int

const (
	ErrUnknown        ErrorCode = 0x00
	ErrAccess         ErrorCode = 0x01
	ErrUnavailable    ErrorCode = 0x02
	ErrInvalid        ErrorCode = 0x03
	ErrIncomplete     ErrorCode = 0x04
	ErrDuplicate      ErrorCode = 0x05
	ErrLimit          ErrorCode = 0x06
	ErrTimeout        ErrorCode = 0x07
	ErrSession        ErrorCode = 0x08
	ErrNotImplemented ErrorCode = 0x09
)

type Payload interface{}
type DataPayload map[string]interface{}

type Error struct {
	Code ErrorCode
	Data Payload
}

func (e Error) Error() string {
	b, _ := json.Marshal(e)

	return string(b)
}

func (e Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"code": e.Code,
		"data": e.Data,
	})
}

func (e *Error) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if v, ok := raw["code"].(float64); ok {
		e.Code = ErrorCode(v)
	} else {
		e.Code = ErrUnknown
	}

	if v, ok := raw["data"].(map[string]interface{}); ok {
		e.Data = DataPayload(v)
	} else {
		e.Data = DataPayload{}
	}

	return nil
}

func NewNotImplementedError(data Payload) Error {
	return Error{
		Code: ErrNotImplemented,
		Data: data,
	}
}

func NewUnknownError(data Payload) Error {
	return Error{
		Code: ErrUnknown,
		Data: data,
	}
}

func NewInvalidError(items []string, data map[string]interface{}) Error {
	var ed DataPayload
	if data != nil {
		ed = DataPayload(data)
	} else {
		ed = make(DataPayload)
	}
	ed["items"] = items

	return Error{
		Code: ErrInvalid,
		Data: ed,
	}
}
