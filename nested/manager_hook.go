package nested

import (
	"bytes"
	"encoding/json"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"go.uber.org/zap"
	"net/http"

	"github.com/globalsign/mgo/bson"
)

const (
	HookEventTypePlaceNewPost        = 0x101
	HookEventTypePlaceNewPostComment = 0x102
	HookEventTypePlaceNewMember      = 0x103
	HookEventTypeAccountTaskAssigned = 0x201
)

type HookEventType int
type HookEvent interface {
	GetType() HookEventType
	IncreaseTries()
}

// NewPostEvent holds information which will be sent to the hook url on each new post
type NewPostEvent struct {
	PlaceID          string        `json:"place_id"`
	SenderID         string        `json:"sender_id"`
	PostID           bson.ObjectId `json:"post_id"`
	PostTitle        string        `json:"post_title"`
	AttachmentsCount int           `json:"attachments_count"`
	retries          int
}

func (e NewPostEvent) GetType() HookEventType {
	return HookEventTypePlaceNewPost
}
func (e NewPostEvent) IncreaseTries() {
	e.retries++
}

// NewPostCommentEvent
// Holds information which will be sent to the hook url on each comment on
// posts of a place
type NewPostCommentEvent struct {
	PlaceID   string        `json:"place_id"`
	SenderID  string        `json:"sender_id"`
	PostID    bson.ObjectId `json:"post_id"`
	CommentID bson.ObjectId `json:"comment_id"`
	retries   int
}

func (e NewPostCommentEvent) GetType() HookEventType {
	return HookEventTypePlaceNewPostComment
}
func (e NewPostCommentEvent) IncreaseTries() {
	e.retries++
}

// NewMemberEvent
// Holds information which will be sent to the hook url every time a user joins
// a place
type NewMemberEvent struct {
	PlaceID         string `json:"place_id"`
	MemberID        string `json:"member_id"`
	MemberName      string `json:"member_name"`
	ProfilePicSmall string `json:"profile_pic_small"`
	ProfilePicLarge string `json:"profile_pic_large"`
	retries         int
}

func (e NewMemberEvent) GetType() HookEventType {
	return HookEventTypePlaceNewMember
}
func (e NewMemberEvent) IncreaseTries() {
	e.retries++
}

// AccountTaskAssignedEvent
// Holds information which will be sent to the hook url every time a task
// is assigned to a user
type AccountTaskAssignedEvent struct {
	AccountID    string        `json:"account_id"`
	TaskID       bson.ObjectId `json:"task_id"`
	TaskTitle    string        `json:"task_title"`
	AssignorID   string        `json:"assignor_id"`
	AssignorName string        `json:"assignor_name"`
	retries      int
}

func (e AccountTaskAssignedEvent) GetType() HookEventType {
	return HookEventTypeAccountTaskAssigned
}
func (e AccountTaskAssignedEvent) IncreaseTries() {
	e.retries++
}

type Hook struct {
	ID        bson.ObjectId `bson:"_id" json:"id"`
	Name      string        `bson:"name" json:"name"`
	SetBy     string        `bson:"set_by" json:"set_by"`
	AnchorID  interface{}   `bson:"anchor_id" json:"anchor_id"`
	EventType int           `bson:"event_type" json:"event_type"`
	Url       string        `bson:"url" json:"url"`
}

type HookManager struct {
	chLimit  chan bool
	chEvents chan HookEvent
}

func newHookManager() *HookManager {
	hm := new(HookManager)
	hm.chLimit = make(chan bool, 10)
	hm.chEvents = make(chan HookEvent, 1000)

	// Run the hooker in the background
	go hm.hooker()

	return hm
}

// AddHook registers a new hook in database.
// HookType can be:
//      HookEventTypePlaceNewPost         = 0x101
//      HookEventTypePlaceNewPostComment  = 0x102
//      HookEventTypePlaceNewMember       = 0x103
//      HookEventTypeAccountTaskAssigned  = 0x201
// AnchorID:
//      1. place_id
//      2. account_id
//      3. task_id
func (m *HookManager) AddHook(setterID, hookName string, anchorID interface{}, hookType int, url string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	hook := new(Hook)
	hook.ID = bson.NewObjectId()
	hook.Name = hookName
	hook.EventType = hookType
	hook.AnchorID = anchorID
	hook.SetBy = setterID
	hook.Url = url

	if err := db.C(global.CollectionHooks).Insert(hook); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

func (m *HookManager) RemoveHook(hookID bson.ObjectId) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionHooks).RemoveId(hookID); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true

}

func (m *HookManager) GetHooksBySetterID(setterID string, pg Pagination) []Hook {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	hooks := make([]Hook, 0, pg.GetLimit())
	if err := db.C(global.CollectionHooks).Find(
		bson.M{"set_by": setterID},
	).Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&hooks); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return hooks
}

// hooker will be run in background and listens to chEvents channel and run the appropriate function
// according to the incoming hook event it receives from the channel
func (m *HookManager) hooker() {
	for event := range m.chEvents {
		m.chLimit <- true
		go m.hHook(event)
	}

}

func (m *HookManager) hHook(e HookEvent) {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	var anchorID interface{}
	var hookType HookEventType
	switch x := e.(type) {
	case NewPostEvent:
		hookType = HookEventTypePlaceNewPost
		anchorID = x.PlaceID
	case NewPostCommentEvent:
		hookType = HookEventTypePlaceNewPostComment
		anchorID = x.PlaceID
	case NewMemberEvent:
		hookType = HookEventTypePlaceNewMember
		anchorID = x.PlaceID
	default:
		return
	}
	iter := db.C(global.CollectionHooks).Find(bson.M{"anchor_id": anchorID, "event_type": hookType}).Iter()
	defer iter.Close()

	if b, err := json.Marshal(e); err != nil {
		log.Warn("Got error", zap.Error(err))
	} else {
		postBody := new(bytes.Buffer)
		hook := new(Hook)
		for iter.Next(hook) {
			postBody.Write(b)
			if res, err := http.Post(
				hook.Url,
				"application/json",
				postBody,
			); err != nil || res.StatusCode != http.StatusOK {
				e.IncreaseTries()
				// m.chEvents <- e
			}
			postBody.Reset()
		}
	}

	<-m.chLimit
}
