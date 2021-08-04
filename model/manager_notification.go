package nested

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"strconv"
	"strings"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

const (
	/*
		NESTED POST NOTIFICATIONS
	*/

	NotificationTypeMention              = 0x001
	NotificationTypeComment              = 0x002
	NotificationTypeJoinedPlace          = 0x005
	NotificationTypePromoted             = 0x006
	NotificationTypeDemoted              = 0x007
	NotificationTypePlaceSettingsChanged = 0x008
	NotificationTypeNewSession           = 0x009
	NotificationTypeLabelRequestApproved = 0x011
	NotificationTypeLabelRequestRejected = 0x012
	NotificationTypeLabelRequestCreated  = 0x013
	NotificationTypeLabelJoined          = 0x014

	/*
		NESTED TASK NOTIFICATIONS
	 */

	NotificationTypeTaskMention         = 0x101
	NotificationTypeTaskComment         = 0x102
	NotificationTypeTaskAssigned        = 0x103
	NotificationTypeTaskAssigneeChanged = 0x104
	NotificationTypeTaskAddToCandidates = 0x105
	NotificationTypeTaskAddToWatchers   = 0x106
	NotificationTypeTaskDueTimeUpdated  = 0x107
	NotificationTypeTaskOverDue         = 0x108
	NotificationTypeTaskUpdated         = 0x110
	NotificationTypeTaskRejected        = 0x111
	NotificationTypeTaskAccepted        = 0x112
	NotificationTypeTaskCompleted       = 0x113
	NotificationTypeTaskHold            = 0x114
	NotificationTypeTaskInProgress      = 0x115
	NotificationTypeTaskFailed          = 0x116
	NotificationTypeTaskAddToEditors    = 0x117
)

const (
	NotificationSubjectPost = 0x01
	NotificationSubjectTask = 0x02
)

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

type NotificationManager struct{}

func NewNotificationManager() *NotificationManager {
	return new(NotificationManager)
}

func (n *Notification) incrementCounter() {
	defer _Manager.Account.removeCache(n.AccountID)

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	db.C(global.COLLECTION_ACCOUNTS).UpdateId(
		n.AccountID,
		bson.M{
			"$inc": bson.M{
				"counters.total_notifications":  1,
				"counters.unread_notifications": 1,
			},
		},
	)
}

// GetByAccountID returns an array of Notifications which belong to accountID.
// If only_unread is set to TRUE then this function returns only unread notifications otherwise returns read or unread
// notifications.
// This function supports pagination
func (nm *NotificationManager) GetByAccountID(
	accountID string, pg Pagination, only_unread bool, subject string,
) []Notification {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := make([]Notification, 0, pg.GetLimit())
	sortItem := "last_update"
	sortDir := fmt.Sprintf("-%s", sortItem)
	query := bson.M{
		"account_id": accountID,
		"_removed":   false,
	}
	switch subject {
	case "task":
		query["subject"] = NotificationSubjectTask
	case "post":
		query["subject"] = NotificationSubjectPost
	default:

	}

	query, sortDir = pg.FillQuery(query, sortItem, sortDir)

	if only_unread {
		query["read"] = false
	}
	if err := db.C(global.COLLECTION_NOTIFICATIONS).Find(query).
		Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit()).
		All(&n); err != nil {
		log.Warn(err.Error())
	}
	return n
}

// GetByID returns a pointer to Notification object identified by notificationID
func (nm *NotificationManager) GetByID(notificationID string) (n *Notification) {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n = new(Notification)
	db.C(global.COLLECTION_NOTIFICATIONS).FindId(notificationID).One(&n)
	return
}

// MarkAsRead set the notificationID as read, if notificationID = 'all' then mark all the unread notifications as read
func (nm *NotificationManager) MarkAsRead(notificationID, accountID string) {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	switch notificationID {
	case "all":
		db.C(global.COLLECTION_NOTIFICATIONS).UpdateAll(
			bson.M{
				"read":       false,
				"account_id": accountID,
			},
			bson.M{
				"$set": bson.M{
					"read":    true,
					"read_on": Timestamp(),
				},
			},
		)
	default:
		change := mgo.Change{
			Update: bson.M{
				"$set": bson.M{
					"read":    true,
					"read_on": Timestamp(),
				},
			},
			ReturnNew: true,
		}
		db.C(global.COLLECTION_NOTIFICATIONS).Find(
			bson.M{
				"_id":        notificationID,
				"read":       false,
				"account_id": accountID,
			},
		).Apply(change, nil)
	}
}

// MarkAsReadByPostID set all notifications related to postID as read. Useful when clients read comments
func (nm *NotificationManager) MarkAsReadByPostID(postID bson.ObjectId, accountID string) []string {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	iter := db.C(global.COLLECTION_NOTIFICATIONS).Find(
		bson.M{
			"read":       false,
			"account_id": accountID,
			"post_id":    postID,
		},
	).Iter()
	notificationIDs := make([]string, 0)
	notification := new(Notification)
	for iter.Next(notification) {
		notificationIDs = append(notificationIDs, notification.ID)
	}
	iter.Close()

	db.C(global.COLLECTION_NOTIFICATIONS).UpdateAll(
		bson.M{
			"read":       false,
			"account_id": accountID,
			"post_id":    postID,
		},
		bson.M{
			"$set": bson.M{
				"read":    true,
				"read_on": Timestamp(),
			},
		},
	)

	return notificationIDs
}

// Remove removes notification from user notifications' list
func (nm *NotificationManager) Remove(notificationID string) bool {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_NOTIFICATIONS).UpdateId(
		notificationID,
		bson.M{"$set": bson.M{"_removed": true}},
	); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

// Post Notifications
func (nm *NotificationManager) AddMention(senderID, mentionedID string, postID, commentID bson.ObjectId) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("MNT" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeMention
	n.Subject = NotificationSubjectPost
	n.PostID = postID
	n.CommentID = commentID
	n.AccountID = mentionedID
	n.ActorID = senderID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) JoinedPlace(adderID, addedID, placeID string) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("JOI" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeJoinedPlace
	n.Subject = NotificationSubjectPost
	n.AccountID = addedID
	n.ActorID = adderID
	n.PlaceID = placeID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) Comment(accountID, commenterID string, postID, commentID bson.ObjectId) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("COM" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeComment
	n.Subject = NotificationSubjectPost
	n.PostID = postID
	n.CommentID = commentID
	n.AccountID = accountID
	n.ActorID = commenterID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false

	ch := mgo.Change{
		Update: bson.M{
			"$addToSet": bson.M{"data.others": n.ActorID},
			"$set": bson.M{
				"actor_id":    n.ActorID,
				"comment_id":  commentID,
				"last_update": n.LastUpdate,
			},
		},
		ReturnNew: true,
	}
	if ci, err := db.C(global.COLLECTION_NOTIFICATIONS).Find(
		bson.M{
			"account_id": n.AccountID,
			"type":       n.Type,
			"post_id":    n.PostID,
			"read":       false,
			"_removed":   false,
		},
	).Apply(ch, &n); err == nil {
		if ci.Updated > 0 {
			return n
		}
	}

	n.Data.Others = []string{n.ActorID}
	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) Promoted(promotedID, promoterID, placeID string) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("PRO" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypePromoted
	n.Subject = NotificationSubjectPost
	n.AccountID = promotedID
	n.ActorID = promoterID
	n.PlaceID = placeID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) Demoted(demotedID, demoterID, placeID string) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("DEM" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeDemoted
	n.Subject = NotificationSubjectPost
	n.AccountID = demotedID
	n.ActorID = demoterID
	n.PlaceID = placeID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) PlaceSettingsChanged(accountID, changerID, placeID string) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("SET" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypePlaceSettingsChanged
	n.Subject = NotificationSubjectPost
	n.AccountID = accountID
	n.ActorID = changerID
	n.PlaceID = placeID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false

	ch := mgo.Change{
		Update:    bson.M{"$set": bson.M{"last_update": Timestamp()}},
		ReturnNew: true,
	}
	if ci, err := db.C(global.COLLECTION_NOTIFICATIONS).Find(
		bson.M{
			"type":     n.Type,
			"place_id": n.PlaceID,
			"read":     false,
			"_removed": false,
		},
	).Apply(ch, n); err == nil {
		if ci.Updated > 0 {
			return n
		}
	}
	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) NewSession(accountID, clientID string) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("SES" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeNewSession
	n.Subject = NotificationSubjectPost
	n.ActorID = "nested"
	n.AccountID = accountID
	n.ClientID = clientID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) LabelRequestApproved(
	accountID, labelID, deciderID string, labelRequestID bson.ObjectId,
) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("LRR" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeLabelRequestApproved
	n.Subject = NotificationSubjectPost
	n.AccountID = accountID
	n.ActorID = deciderID
	n.LabelID = labelID
	n.LabelRequestID = labelRequestID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n

}

func (nm *NotificationManager) LabelRequestRejected(
	accountID, labelID, deciderID string, labelRequestID bson.ObjectId,
) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("LRR" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeLabelRequestRejected
	n.Subject = NotificationSubjectPost
	n.AccountID = accountID
	n.ActorID = deciderID
	n.LabelID = labelID
	n.LabelRequestID = labelRequestID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n

}

func (nm *NotificationManager) LabelRequest(accountID, requesterID string) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("LRC" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeLabelRequestCreated
	n.Subject = NotificationSubjectPost
	n.AccountID = accountID
	n.ActorID = requesterID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false

	ch := mgo.Change{
		Update: bson.M{
			"$addToSet": bson.M{"data.others": n.ActorID},
			"$set": bson.M{
				"actor_id":    n.ActorID,
				"last_update": n.LastUpdate,
			},
		},
	}
	if ci, err := db.C(global.COLLECTION_NOTIFICATIONS).Find(
		bson.M{
			"account_id": n.AccountID,
			"type":       n.Type,
			"read":       false,
			"_removed":   false,
		},
	).Apply(ch, &n); err == nil {
		if ci.Updated > 0 {
			return n
		}
	}

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) LabelJoined(accountID, labelID, adderID string) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("LJOI" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeLabelJoined
	n.Subject = NotificationSubjectPost
	n.AccountID = accountID
	n.ActorID = adderID
	n.LabelID = labelID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

// Task Notifications
func (nm *NotificationManager) TaskAssigned(accountID, assignorID string, task *Task) *Notification {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskAssigned
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.ActorID = assignorID
	n.TaskID = task.ID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n

}

func (nm *NotificationManager) TaskWatcherAdded(accountID, adderID string, task *Task) *Notification {
	//
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAW" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskAddToWatchers
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.ActorID = adderID
	n.TaskID = task.ID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) TaskEditorAdded(accountID, adderID string, task *Task) *Notification {
	//
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAE" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskAddToEditors
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.ActorID = adderID
	n.TaskID = task.ID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) TaskCandidateAdded(accountID, adderID string, task *Task) *Notification {
	//
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskAddToCandidates
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.ActorID = adderID
	n.TaskID = task.ID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()

	return n
}

func (nm *NotificationManager) TaskAssigneeChanged(accountID, newAssigneeID, actorID string, task *Task) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskAssigneeChanged
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.ActorID = actorID
	n.TaskID = task.ID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) TaskUpdated(
	accountID string, changerID string, task *Task, newTitle, newDesc string,
) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskUpdated
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.ActorID = changerID
	n.TaskID = task.ID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskDesc = newDesc
	n.Data.TaskTitle = newTitle
	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) TaskOverdue(accountID string, task *Task) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskOverDue
	n.Subject = NotificationSubjectTask
	n.ActorID = "nested"
	n.AccountID = accountID
	n.TaskID = task.ID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()

	return n
}

func (nm *NotificationManager) TaskDueTimeUpdated(accountID string, task *Task) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskDueTimeUpdated
	n.Subject = NotificationSubjectTask
	n.ActorID = "nested"
	n.AccountID = accountID
	n.TaskID = task.ID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()

	return n
}

func (nm *NotificationManager) TaskRejected(accountID, actorID string, task *Task) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskRejected
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.TaskID = task.ID
	n.ActorID = actorID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()

	return n
}

func (nm *NotificationManager) TaskAccepted(accountID, actorID string, task *Task) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskAccepted
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.TaskID = task.ID
	n.ActorID = actorID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()

	return n
}

func (nm *NotificationManager) TaskCompleted(accountID, actorID string, task *Task) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskCompleted
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.TaskID = task.ID
	n.ActorID = actorID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()

	return n
}

func (nm *NotificationManager) TaskHold(accountID, actorID string, task *Task) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskHold
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.TaskID = task.ID
	n.ActorID = actorID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()

	return n
}

func (nm *NotificationManager) TaskInProgress(accountID, actorID string, task *Task) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskInProgress
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.TaskID = task.ID
	n.ActorID = actorID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()

	return n
}

func (nm *NotificationManager) TaskFailed(accountID, actorID string, task *Task) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskFailed
	n.Subject = NotificationSubjectTask
	n.AccountID = accountID
	n.TaskID = task.ID
	n.ActorID = actorID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()

	return n
}

func (nm *NotificationManager) TaskCommentMentioned(
	mentionedID, actorID string, task *Task, activityID bson.ObjectId,
) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("MNT" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskMention
	n.Subject = NotificationSubjectTask
	n.TaskID = task.ID
	n.AccountID = mentionedID
	n.ActorID = actorID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Read = false
	n.Removed = false
	n.Data.ActivityID = activityID
	n.Data.TaskTitle = task.Title

	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}

func (nm *NotificationManager) TaskComment(accountID, actorID string, task *Task, activityID bson.ObjectId) *Notification {
	// removed LOG Function
	// removed LOG Function

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n := new(Notification)
	n.ID = strings.ToUpper("COM" + strconv.Itoa(int(Timestamp())) + RandomID(32))
	n.Type = NotificationTypeTaskComment
	n.Subject = NotificationSubjectTask
	n.TaskID = task.ID
	n.AccountID = accountID
	n.ActorID = actorID
	n.Timestamp = Timestamp()
	n.LastUpdate = n.Timestamp
	n.Data.ActivityID = activityID
	n.Data.TaskTitle = task.Title

	n.Read = false
	n.Removed = false

	ch := mgo.Change{
		Update: bson.M{
			"$addToSet": bson.M{"data.others": n.ActorID},
			"$set": bson.M{
				"actor_id":         n.ActorID,
				"data.activity_id": activityID,
				"last_update":      n.LastUpdate,
			},
		},
		ReturnNew: true,
	}
	if ci, err := db.C(global.COLLECTION_NOTIFICATIONS).Find(
		bson.M{
			"account_id": n.AccountID,
			"type":       n.Type,
			"task_id":    n.TaskID,
			"read":       false,
			"_removed":   false,
		},
	).Apply(ch, &n); err == nil {
		if ci.Updated > 0 {
			return n
		}
	}

	n.Data.Others = []string{n.ActorID}
	if err := db.C(global.COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
		log.Warn(err.Error())
		return nil
	}
	n.incrementCounter()
	return n
}
