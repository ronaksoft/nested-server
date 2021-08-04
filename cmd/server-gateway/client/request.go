package nestedGateway

import (
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"

	"github.com/globalsign/mgo/bson"
)

type Request struct {
	Format          string        `json:"format"`
	Type            string        `json:"type,omitempty"`
	RequestID       string        `json:"_reqid,omitempty"`
	Command         string        `json:"cmd"`
	Compressed      bool          `json:"gzip"`
	SessionKey      bson.ObjectId `json:"_sk"`
	SessionSec      string        `json:"_ss"`
	AppID           string        `json:"_app_id"`
	AppToken        string        `json:"_app_token"`
	ClientID        string        `json:"_cid"`
	ClientVersion   int           `json:"_cver"`
	ClientIP        string        `json:"_cip"`
	UserAgent       string        `json:"_ua"`
	WebsocketID     string        `json:"ws_id"`
	Data            tools.M       `json:"data"`
	PacketSize      int           `json:"-"`
	ResponseChannel chan Response
}
