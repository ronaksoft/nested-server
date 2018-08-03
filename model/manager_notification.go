package nested

import (
    "fmt"
    "github.com/globalsign/mgo"
    "github.com/globalsign/mgo/bson"
    "strconv"
    "strings"
)

const (
    //  NESTED POST NOTIFICATIONS
    NOTIFICATION_TYPE_MENTION                = 0x001
    NOTIFICATION_TYPE_COMMENT                = 0x002
    NOTIFICATION_TYPE_JOINED_PLACE           = 0x005
    NOTIFICATION_TYPE_PROMOTED               = 0x006
    NOTIFICATION_TYPE_DEMOTED                = 0x007
    NOTIFICATION_TYPE_PLACE_SETTINGS_CHANGED = 0x008
    NOTIFICATION_TYPE_NEW_SESSION            = 0x009
    NOTIFICATION_TYPE_LABEL_REQUEST_APPROVED = 0x011
    NOTIFICATION_TYPE_LABEL_REQUEST_REJECTED = 0x012
    NOTIFICATION_TYPE_LABEL_REQUEST_CREATED  = 0x013
    NOTIFICATION_TYPE_LABEL_JOINED           = 0x014

    // NESTED TASK NOTIFICATIONS
    NOTIFICATION_TYPE_TASK_MENTION           = 0x101
    NOTIFICATION_TYPE_TASK_COMMENT           = 0x102
    NOTIFICATION_TYPE_TASK_ASSIGNED          = 0x103
    NOTIFICATION_TYPE_TASK_ASSIGNEE_CHANGED  = 0x104
    NOTIFICATION_TYPE_TASK_ADD_TO_CANDIDATES = 0x105
    NOTIFICATION_TYPE_TASK_ADD_TO_WATCHERS   = 0x106
    NOTIFICATION_TYPE_TASK_DUE_TIME_UPDATED  = 0x107
    NOTIFICATION_TYPE_TASK_OVER_DUE          = 0x108
    NOTIFICATION_TYPE_TASK_UPDATED           = 0x110
    NOTIFICATION_TYPE_TASK_REJECTED          = 0x111
    NOTIFICATION_TYPE_TASK_ACCEPTED          = 0x112
    NOTIFICATION_TYPE_TASK_COMPLETED         = 0x113
    NOTIFICATION_TYPE_TASK_HOLD              = 0x114
    NOTIFICATION_TYPE_TASK_IN_PROGRESS       = 0x115
    NOTIFICATION_TYPE_TASK_FAILED            = 0x116
    NOTIFICATION_TYPE_TASK_ADD_TO_EDITORS    = 0x117
)

const (
    NOTIFICATION_SUBJECT_POST = 0x01
    NOTIFICATION_SUBJECT_TASK = 0x02
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
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    db.C(COLLECTION_ACCOUNTS).UpdateId(
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
    _funcName := "NotificationManager::GetByAccountID"
    _Log.FunctionStarted(_funcName, accountID, only_unread)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
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
        query["subject"] = NOTIFICATION_SUBJECT_TASK
    case "post":
        query["subject"] = NOTIFICATION_SUBJECT_POST
    default:

    }
    if pg.After > 0 {
        query[sortItem] = bson.M{"$gt": pg.After}
        sortDir = sortItem
    } else if pg.Before > 0 {
        query[sortItem] = bson.M{"$lt": pg.Before}
    }
    if only_unread {
        query["read"] = false
    }
    if err := db.C(COLLECTION_NOTIFICATIONS).Find(query).
        Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit()).
        All(&n); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return n
}

// GetByID returns a pointer to Notification object identified by notificationID
func (nm *NotificationManager) GetByID(notificationID string) (n *Notification) {
    _funcName := "NotificationManager::GetByID"
    _Log.FunctionStarted(_funcName, notificationID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n = new(Notification)
    db.C(COLLECTION_NOTIFICATIONS).FindId(notificationID).One(&n)
    return
}

// MarkAsRead set the notificationID as read, if notificationID = 'all' then mark all the unread notifications as read
func (nm *NotificationManager) MarkAsRead(notificationID, accountID string) {
    _funcName := "NotificationManager::MarkAsRead"
    _Log.FunctionStarted(_funcName, notificationID, accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    switch notificationID {
    case "all":
        db.C(COLLECTION_NOTIFICATIONS).UpdateAll(
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
        db.C(COLLECTION_NOTIFICATIONS).Find(
            bson.M{
                "_id":        notificationID,
                "read":       false,
                "account_id": accountID,
            },
        ).Apply(change, nil)
    }
}

//MarkAsReadByPostID set all notifications related to postID as read. Useful when clients read comments
func (nm *NotificationManager) MarkAsReadByPostID(postID bson.ObjectId, accountID string) []string {
    _funcName := "NotificationManager::MarkAsReadByPostID"
    _Log.FunctionStarted(_funcName, postID.Hex(), accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    iter := db.C(COLLECTION_NOTIFICATIONS).Find(
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

    db.C(COLLECTION_NOTIFICATIONS).UpdateAll(
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
    _funcName := "NotificationManager::Remove"
    _Log.FunctionStarted(_funcName, notificationID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_NOTIFICATIONS).UpdateId(
        notificationID,
        bson.M{"$set": bson.M{"_removed": true}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }
    return true
}

// Post Notifications
func (nm *NotificationManager) AddMention(senderID, mentionedID string, postID, commentID bson.ObjectId) *Notification {
    _funcName := "NotificationManager::AddMention"
    _Log.FunctionStarted(_funcName, senderID, mentionedID, postID.Hex(), commentID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("MNT" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_MENTION
    n.Subject = NOTIFICATION_SUBJECT_POST
    n.PostID = postID
    n.CommentID = commentID
    n.AccountID = mentionedID
    n.ActorID = senderID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error(), senderID, mentionedID, postID.Hex(), commentID.Hex())
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) JoinedPlace(adderID, addedID, placeID string) *Notification {
    _funcName := "NotificationManager::JoinedPlace"
    _Log.FunctionStarted(_funcName, adderID, addedID, placeID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("JOI" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_JOINED_PLACE
    n.Subject = NOTIFICATION_SUBJECT_POST
    n.AccountID = addedID
    n.ActorID = adderID
    n.PlaceID = placeID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error(), adderID, addedID, placeID)
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) Comment(accountID, commenterID string, postID, commentID bson.ObjectId) *Notification {
    _funcName := "NotificationManager::Comment"
    _Log.FunctionStarted(_funcName, accountID, commenterID, postID.Hex(), commentID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("COM" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_COMMENT
    n.Subject = NOTIFICATION_SUBJECT_POST
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
    if ci, err := db.C(COLLECTION_NOTIFICATIONS).Find(
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
    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) Promoted(promotedID, promoterID, placeID string) *Notification {
    _funcName := "NotificationManager::Promoted"
    _Log.FunctionStarted(_funcName, promotedID, promoterID, placeID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("PRO" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_PROMOTED
    n.Subject = NOTIFICATION_SUBJECT_POST
    n.AccountID = promotedID
    n.ActorID = promoterID
    n.PlaceID = placeID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) Demoted(demotedID, demoterID, placeID string) *Notification {
    _funcName := "NotificationManager::Demoted"
    _Log.FunctionStarted(_funcName, demotedID, demoterID, placeID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("DEM" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_DEMOTED
    n.Subject = NOTIFICATION_SUBJECT_POST
    n.AccountID = demotedID
    n.ActorID = demoterID
    n.PlaceID = placeID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) PlaceSettingsChanged(accountID, changerID, placeID string) *Notification {
    _funcName := "NotificationManager::PlaceSettingsChanged"
    _Log.FunctionStarted(_funcName, accountID, changerID, placeID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("SET" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_PLACE_SETTINGS_CHANGED
    n.Subject = NOTIFICATION_SUBJECT_POST
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
    if ci, err := db.C(COLLECTION_NOTIFICATIONS).Find(
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
    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) NewSession(accountID, clientID string) *Notification {
    _funcName := "NotificationManager::NewSession"
    _Log.FunctionStarted(_funcName, accountID, clientID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("SES" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_NEW_SESSION
    n.Subject = NOTIFICATION_SUBJECT_POST
    n.ActorID = "nested"
    n.AccountID = accountID
    n.ClientID = clientID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) LabelRequestApproved(
    accountID, labelID, deciderID string, labelRequestID bson.ObjectId,
) *Notification {
    _funcName := "NotificationManager::LabelRequestApproved"
    _Log.FunctionStarted(_funcName, accountID, labelID, deciderID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("LRR" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_LABEL_REQUEST_APPROVED
    n.Subject = NOTIFICATION_SUBJECT_POST
    n.AccountID = accountID
    n.ActorID = deciderID
    n.LabelID = labelID
    n.LabelRequestID = labelRequestID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n

}

func (nm *NotificationManager) LabelRequestRejected(
    accountID, labelID, deciderID string, labelRequestID bson.ObjectId,
) *Notification {
    _funcName := "NotificationManager::LabelRequestRejected"
    _Log.FunctionStarted(_funcName, accountID, labelID, deciderID, labelRequestID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("LRR" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_LABEL_REQUEST_REJECTED
    n.Subject = NOTIFICATION_SUBJECT_POST
    n.AccountID = accountID
    n.ActorID = deciderID
    n.LabelID = labelID
    n.LabelRequestID = labelRequestID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n

}

func (nm *NotificationManager) LabelRequest(accountID, requesterID string) *Notification {
    _funcName := "NotificationManager::LabelRequest"
    _Log.FunctionStarted(_funcName, accountID, requesterID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("LRC" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_LABEL_REQUEST_CREATED
    n.Subject = NOTIFICATION_SUBJECT_POST
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
    if ci, err := db.C(COLLECTION_NOTIFICATIONS).Find(
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

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) LabelJoined(accountID, labelID, adderID string) *Notification {
    _funcName := "NotificationManager::LabelJoined"
    _Log.FunctionStarted(_funcName, accountID, labelID, adderID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("LJOI" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_LABEL_JOINED
    n.Subject = NOTIFICATION_SUBJECT_POST
    n.AccountID = accountID
    n.ActorID = adderID
    n.LabelID = labelID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n
}

// Task Notifications
func (nm *NotificationManager) TaskAssigned(accountID, assignorID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskAssigned"
    _Log.FunctionStarted(_funcName, accountID, assignorID, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_ASSIGNED
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.ActorID = assignorID
    n.TaskID = task.ID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n

}

func (nm *NotificationManager) TaskWatcherAdded(accountID, adderID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskWatcherAdded"
    _Log.FunctionStarted(_funcName, accountID, adderID, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAW" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_ADD_TO_WATCHERS
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.ActorID = adderID
    n.TaskID = task.ID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) TaskEditorAdded(accountID, adderID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskEditorAdded"
    _Log.FunctionStarted(_funcName, accountID, adderID, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAE" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_ADD_TO_EDITORS
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.ActorID = adderID
    n.TaskID = task.ID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) TaskCandidateAdded(accountID, adderID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskCandidateAdded"
    _Log.FunctionStarted(_funcName, accountID, adderID, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_ADD_TO_CANDIDATES
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.ActorID = adderID
    n.TaskID = task.ID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()

    return n
}

func (nm *NotificationManager) TaskAssigneeChanged(accountID, newAssigneeID, actorID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskAssigneeChanged"
    _Log.FunctionStarted(_funcName, accountID, newAssigneeID, actorID, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_ASSIGNEE_CHANGED
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.ActorID = actorID
    n.TaskID = task.ID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) TaskUpdated(
    accountID string, changerID string, task *Task, newTitle, newDesc string,
) *Notification {
    _funcName := "NotificationManager::TaskUpdated"
    _Log.FunctionStarted(_funcName, changerID, task.ID.Hex(), newTitle, newDesc)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_UPDATED
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.ActorID = changerID
    n.TaskID = task.ID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskDesc = newDesc
    n.Data.TaskTitle = newTitle
    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) TaskOverdue(accountID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskOverdue"
    _Log.FunctionStarted(_funcName, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_OVER_DUE
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.ActorID = "nested"
    n.AccountID = accountID
    n.TaskID = task.ID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()

    return n
}

func (nm *NotificationManager) TaskDueTimeUpdated(accountID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskOverdueUpdated"
    _Log.FunctionStarted(_funcName, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_DUE_TIME_UPDATED
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.ActorID = "nested"
    n.AccountID = accountID
    n.TaskID = task.ID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()

    return n
}

func (nm *NotificationManager) TaskRejected(accountID, actorID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskRejected"
    _Log.FunctionStarted(_funcName, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_REJECTED
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.TaskID = task.ID
    n.ActorID = actorID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()

    return n
}

func (nm *NotificationManager) TaskAccepted(accountID, actorID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskAccepted"
    _Log.FunctionStarted(_funcName, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_ACCEPTED
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.TaskID = task.ID
    n.ActorID = actorID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()

    return n
}

func (nm *NotificationManager) TaskCompleted(accountID, actorID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskCompleted"
    _Log.FunctionStarted(_funcName, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_COMPLETED
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.TaskID = task.ID
    n.ActorID = actorID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()

    return n
}

func (nm *NotificationManager) TaskHold(accountID, actorID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskHold"
    _Log.FunctionStarted(_funcName, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_HOLD
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.TaskID = task.ID
    n.ActorID = actorID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()

    return n
}

func (nm *NotificationManager) TaskInProgress(accountID, actorID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskInProgress"
    _Log.FunctionStarted(_funcName, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_IN_PROGRESS
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.TaskID = task.ID
    n.ActorID = actorID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()

    return n
}

func (nm *NotificationManager) TaskFailed(accountID, actorID string, task *Task) *Notification {
    _funcName := "NotificationManager::TaskFailed"
    _Log.FunctionStarted(_funcName, task.ID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("TAS" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_FAILED
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.AccountID = accountID
    n.TaskID = task.ID
    n.ActorID = actorID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()

    return n
}

func (nm *NotificationManager) TaskCommentMentioned(
    mentionedID, actorID string, task *Task, activityID bson.ObjectId,
) *Notification {
    _funcName := "NotificationManager::AddMention"
    _Log.FunctionStarted(_funcName, actorID, mentionedID, task.ID.Hex(), activityID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("MNT" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_MENTION
    n.Subject = NOTIFICATION_SUBJECT_TASK
    n.TaskID = task.ID
    n.AccountID = mentionedID
    n.ActorID = actorID
    n.Timestamp = Timestamp()
    n.LastUpdate = n.Timestamp
    n.Read = false
    n.Removed = false
    n.Data.ActivityID = activityID
    n.Data.TaskTitle = task.Title

    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error(), actorID, mentionedID, task.ID.Hex(), activityID.Hex())
        return nil
    }
    n.incrementCounter()
    return n
}

func (nm *NotificationManager) TaskComment(accountID, actorID string, task *Task, activityID bson.ObjectId) *Notification {
    _funcName := "NotificationManager::Comment"
    _Log.FunctionStarted(_funcName, accountID, actorID, activityID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n := new(Notification)
    n.ID = strings.ToUpper("COM" + strconv.Itoa(int(Timestamp())) + RandomID(32))
    n.Type = NOTIFICATION_TYPE_TASK_COMMENT
    n.Subject = NOTIFICATION_SUBJECT_TASK
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
    if ci, err := db.C(COLLECTION_NOTIFICATIONS).Find(
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
    if err := db.C(COLLECTION_NOTIFICATIONS).Insert(n); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    n.incrementCounter()
    return n
}
