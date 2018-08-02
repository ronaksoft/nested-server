package nestedGateway

import (
    "git.ronaksoftware.com/nested/server-model-nested"
    "encoding/json"
)

// Response
type Response struct {
    Format    string   `json:"-" bson:"-"`
    Type      string   `json:"type,omitempty" bson:"type,omitempty"`
    RequestID string   `json:"_reqid" bson:"_reqid"`
    Status    string   `json:"status" bson:"status"`
    Late      bool     `json:"late" bson:"late"`
    Data      nested.M `json:"data" bson:"data"`
}

func (r *Response) Error(code nested.ErrorCode, items []string) {
    r.Type = "r"
    r.Status = "err"
    r.Data = nested.M{
        "err_code": code,
        "items":    items,
    }
}
func (r *Response) NotImplemented() {
    r.Type = "r"
    r.Status = "err"
    r.Data = nested.M{
        "err_code": -1,
        "items":    []string{"API Not Implemented."},
    }
}
func (r *Response) NotAuthorized() {
    r.Type = "r"
    r.Status = "err"
    r.Data = nested.M{
        "err_code": -1,
        "items":    []string{"You are not authorized."},
    }
}
func (r *Response) NotInitialized() {
    r.Type = "r"
    r.Status = "err"
    r.Data = nested.M{
        "err_code": -1,
        "items":    []string{"Response not initialized."},
    }
}
func (r *Response) SessionInvalid() {
    r.Type = "r"
    r.Status = "err"
    r.Data = nested.M{
        "err_code": nested.ERR_SESSION,
        "items":    []string{"session is invalid"},
    }
}
func (r *Response) Timeout() {
    r.Type = "r"
    r.Status = "err"
    r.Data = nested.M{
        "err_code": nested.ERR_TIMEOUT,
        "items":    []string{"timeout"},
    }
}

func (r *Response) OkWithData(data nested.M) {
    r.Type = "r"
    r.Status = "ok"
    r.Data = data
}
func (r *Response) Ok() {
    r.Type = "r"
    r.Data = nested.M{}
    r.Status = "ok"
}
func (r *Response) SetLate(reqID string) {
    r.Late = true
    r.RequestID = reqID
}
func (r *Response) MarshalJSON() []byte {
    if b, err := json.Marshal(r); err != nil {
        return []byte{}
    } else {
        return b
    }
}
func (r *Response) UnMarshalJSON(b []byte) error {
    err := json.Unmarshal(b, r)
    return err
}
