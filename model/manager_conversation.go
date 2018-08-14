package nested

import (
	"github.com/globalsign/mgo/bson"
)

type Converstation struct {
	Admins    []bson.ObjectId `json:"admins" bson:"admins"`
	CreatedAt uint64          `json:"created_at" bson:"created_at"`
	Crearot   bson.ObjectId   `json:"creator" bson:"creator"`
	ID        bson.ObjectId   `json:"_id" bson:"_id"`
	Members   []string        `json:"member" bson:"member"`
	Title     string          `json:"title" bson:"title"`
	Type      string          `json:"type" bson:"type"`
	Private   bool            `json:"private" bson:"private"`
}

type ConverstationManager struct{}

func NewConverstationManager() *ConverstationManager { return new(ConverstationManager) }

func (*ConverstationManager) Create(conv Converstation) *Converstation {
	_funcName := "ConverstationManager::create"
	conv.ID = bson.NewObjectId()
	conv.CreatedAt = Timestamp()

	if err := _MongoDB.C(COLLECTION_CONVERSATION).Insert(conv); err != nil {
		_Log.Error(_funcName, err.Error())
	}
	return &conv
}

func (cm *ConverstationManager) GetByID(conversationID string, pg Pagination) (conversation *Converstation) {
	return conversation
}

func (*ConverstationManager) GetManyByUserID(userID string, pg Pagination) (conversations *[]Converstation) {
	return conversations
}

func (*ConverstationManager) AddMember(members []string, convID bson.ObjectId) bool { return true }

func (*ConverstationManager) Join(member string, convID bson.ObjectId) bool { return true }

func (*ConverstationManager) Leave(member string, convID bson.ObjectId) bool { return true }

func (*ConverstationManager) RemoveMember(members []string, convID bson.ObjectId) bool { return true }

func (*ConverstationManager) Update(conv Converstation) bool { return true }

func (*ConverstationManager) Remove(conv Converstation) bool { return true }

func (*ConverstationManager) AddAdmin(member string, convID bson.ObjectId) bool { return true }

func (*ConverstationManager) RemoveAdmin(member string, convID bson.ObjectId) bool { return true }
