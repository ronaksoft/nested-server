package pusher

import (
	"github.com/globalsign/mgo/bson"
)

/*
   Creation Time: 2021 - Aug - 04
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type Notification struct {
	ID             string           `json:"_id" bson:"_id"`
	Type           int              `json:"type" bson:"type"`
	Subject        int              `json:"subject" bson:"subject"`
	ActorID        string           `json:"actor_id" bson:"actor_id"`
	AccountID      string           `json:"account_id" bson:"account_id"`
	ClientID       string           `json:"_cid,omitempty" bson:"_cid,omitempty"`
	LabelID        string           `json:"label_id" bson:"label_id"`
	PlaceID        string           `json:"place_id" bson:"place_id"`
	InvitationID   string           `json:"invite_id,omitempty" bson:"invite_id,omitempty"`
	CommentID      bson.ObjectId    `json:"comment_id,omitempty" bson:"comment_id,omitempty"`
	PostID         bson.ObjectId    `json:"post_id,omitempty" bson:"post_id,omitempty"`
	TaskID         bson.ObjectId    `json:"task_id,omitempty" bson:"task_id,omitempty"`
	LabelRequestID bson.ObjectId    `json:"label_request_id" bson:"label_request_id,omitempty"`
	Data           NotificationData `json:"data,omitempty" bson:"data,omitempty"`
	Read           bool             `json:"read" bson:"read"`
	Timestamp      uint64           `json:"timestamp" bson:"timestamp"`
	LastUpdate     uint64           `json:"last_update" bson:"last_update"`
	Removed        bool             `json:"_removed,omitempty" bson:"_removed"`
}
type NotificationData struct {
	Others     []string      `json:"others,omitempty" bson:"others"`
	TaskTitle  string        `json:"task_title,omitempty" bson:"task_title,omitempty"`
	TaskDesc   string        `json:"task_desc,omitempty" bson:"task_desc,omitempty"`
	ActivityID bson.ObjectId `json:"activity_id,omitempty" bson:"activity_id,omitempty"`
	Text       string        `json:"text,omitempty" bson:"text,omitempty"`
}
