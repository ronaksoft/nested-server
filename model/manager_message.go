package nested

import (
	"github.com/globalsign/mgo/bson"
)

const (
	CONV_TYPE_DIRECT = "direct"
	CONV_TYPE_MULTI  = "multi"

	COLLECTION_CONVERSATION = "conversation"
	COLLECTION_MESSAGE      = "message"
)

type Message struct {
	ID             bson.ObjectId `json:"_id" bson:"_id"`
	Attachment     []UniversalID `json:"attachment" bson:"attachment"`
	ConversationID bson.ObjectId `json:"conv_id" bson:"conv_id"`
	ReplyTo        string        `json:"reply_to" bson:"reply_to"`
	Removed        bool          `json:"_removed" bson:"_removed"`
	Read           bool          `json:"read" bson:"read"`
	SenderID       string        `json:"sender_id" bson:"sender_id"`
	Text           string        `json:"text" bson:"text"`
	Timestamp      uint64        `json:"timestamp" bson:"timestamp"`
}

type MessageManager struct{}

func NewMessageManager() *ConverstationManager { return new(ConverstationManager) }

func (*MessageManager) GetMany(userID, convID string, pg Pagination) (messages []*Message) {
	return messages
}

func (*MessageManager) Get(msgID, convID string) (message *Message) { return message }

func (*MessageManager) Update(msg Message, pg Pagination) bool { return true }

func (*MessageManager) Remove(msg Message, pg Pagination) bool { return true }

func (*MessageManager) Send(msg Message, pg Pagination) bool { return true }
