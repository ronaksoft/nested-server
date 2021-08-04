package protocol

import "encoding/json"

type ErrorCode int

const (
	ErrorNotImplemented ErrorCode = iota - 1 // -1
	ErrorUnknown                             // 0
	ErrorForbidden                           // 1
	ErrorUnavailable                         // 2
	ErrorInvalid                             // 3
	ErrorIncomplete                          // 4
	ErrorDuplicate                           // 5
	ErrorLimit                               // 6
	ErrorTimeout                             // 7
)

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
		e.Code = ErrorUnknown
	}

	if v, ok := raw["data"].(map[string]interface{}); ok {
		e.Data = D(v)
	} else {
		e.Data = D{}
	}

	return nil
}

func NewError(code ErrorCode, data Payload) Error {
	return Error{
		Code: code,
		Data: data,
	}
}

func NewNotImplementedError(data Payload) Error {
	return Error{
		Code: ErrorNotImplemented,
		Data: data,
	}
}

func NewUnknownError(data Payload) Error {
	return Error{
		Code: ErrorUnknown,
		Data: data,
	}
}

func NewForbiddenError(data Payload) Error {
	return Error{
		Code: ErrorForbidden,
		Data: data,
	}
}

func NewUnavailableError(items []string, data map[string]interface{}) Error {
	var ed D
	if data != nil {
		ed = D(data)
	} else {
		ed = make(D)
	}
	ed["items"] = items

	return Error{
		Code: ErrorUnavailable,
		Data: ed,
	}
}

func NewInvalidError(items []string, data map[string]interface{}) Error {
	var ed D
	if data != nil {
		ed = D(data)
	} else {
		ed = make(D)
	}
	ed["items"] = items

	return Error{
		Code: ErrorInvalid,
		Data: ed,
	}
}

func NewIncompleteError(data Payload) Error {
	return Error{
		Code: ErrorIncomplete,
		Data: data,
	}
}

func NewDuplicateError(items []string, data map[string]interface{}) Error {
	var ed D
	if data != nil {
		ed = D(data)
	} else {
		ed = make(D)
	}
	ed["items"] = items

	return Error{
		Code: ErrorDuplicate,
		Data: ed,
	}
}

func NewLimitError(items []string, data map[string]interface{}) Error {
	var ed D
	if data != nil {
		ed = D(data)
	} else {
		ed = make(D)
	}
	ed["items"] = items

	return Error{
		Code: ErrorLimit,
		Data: ed,
	}
}

func NewTimeoutError() Error {
	return Error{
		Code: ErrorTimeout,
	}
}
