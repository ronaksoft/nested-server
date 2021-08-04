package nestedGateway

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
)

// Response is the json wrapper around the server's response
type Response struct {
	Format    string  `json:"-" bson:"-"`
	Type      string  `json:"type,omitempty" bson:"type,omitempty"`
	RequestID string  `json:"_reqid" bson:"_reqid"`
	Status    string  `json:"status" bson:"status"`
	Late      bool    `json:"late" bson:"late"`
	Data      tools.M `json:"data" bson:"data"`
}

func (r *Response) Error(code global.ErrorCode, items []string) {
	r.Type = "r"
	r.Status = "err"
	r.Data = tools.M{
		"err_code": code,
		"items":    items,
	}
}

func (r *Response) NotImplemented() {
	r.Type = "r"
	r.Status = "err"
	r.Data = tools.M{
		"err_code": -1,
		"items":    []string{"API Not Implemented."},
	}
}

func (r *Response) NotAuthorized() {
	r.Type = "r"
	r.Status = "err"
	r.Data = tools.M{
		"err_code": -1,
		"items":    []string{"You are not authorized."},
	}
}

func (r *Response) NotInitialized() {
	r.Type = "r"
	r.Status = "err"
	r.Data = tools.M{
		"err_code": -1,
		"items":    []string{"Response not initialized."},
	}
}

func (r *Response) SessionInvalid() {
	r.Type = "r"
	r.Status = "err"
	r.Data = tools.M{
		"err_code": global.ErrSession,
		"items":    []string{"session is invalid"},
	}
}

func (r *Response) Timeout() {
	r.Type = "r"
	r.Status = "err"
	r.Data = tools.M{
		"err_code": global.ErrTimeout,
		"items":    []string{"timeout"},
	}
}

func (r *Response) OkWithData(data tools.M) {
	r.Type = "r"
	r.Status = "ok"
	r.Data = data
}

func (r *Response) Ok() {
	r.Type = "r"
	r.Data = tools.M{}
	r.Status = "ok"
}

func (r *Response) SetLate(reqID string) {
	r.Late = true
	r.RequestID = reqID
}
