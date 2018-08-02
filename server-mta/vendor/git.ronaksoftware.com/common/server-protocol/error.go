package protocol

import "encoding/json"

type ErrorCode int

const (
  ERROR_NOT_IMPLEMENTED ErrorCode = iota - 1 // -1
	ERROR_UNKNOWN // 0
	ERROR_FORBIDDEN // 1
	ERROR_UNAVAILABLE // 2
	ERROR_INVALID // 3
	ERROR_INCOMPLETE // 4
	ERROR_DUPLICATE // 5
	ERROR_LIMIT // 6
	ERROR_TIMEOUT //7
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
    e.Code = ERROR_UNKNOWN
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
    Code: ERROR_NOT_IMPLEMENTED,
    Data: data,
  }
}

func NewUnknownError(data Payload) Error {
  return Error{
    Code: ERROR_UNKNOWN,
    Data: data,
  }
}

func NewForbiddenError(data Payload) Error {
  return Error{
    Code: ERROR_FORBIDDEN,
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
    Code: ERROR_UNAVAILABLE,
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
    Code: ERROR_INVALID,
    Data: ed,
  }
}

func NewIncompleteError(data Payload) Error {
  return Error{
    Code: ERROR_INCOMPLETE,
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
    Code: ERROR_DUPLICATE,
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
    Code: ERROR_LIMIT,
    Data: ed,
  }
}

func NewTimeoutError() Error {
	return Error{
		Code: ERROR_TIMEOUT,
	}
}
