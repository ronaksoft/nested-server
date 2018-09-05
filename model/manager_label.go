package nested

import (
    "fmt"
    "github.com/globalsign/mgo/bson"
    "log"
    "strings"
)

const (
    LABEL_COLOUR_CODE_A = "A"
    LABEL_COLOUR_CODE_B = "B"
    LABEL_COLOUR_CODE_C = "C"
    LABEL_COLOUR_CODE_D = "D"
    LABEL_COLOUR_CODE_E = "E"
    LABEL_COLOUR_CODE_F = "F"
    LABEL_COLOUR_CODE_G = "G"
)
const (
    LABEL_FILTER_MY_LABELS   = "my_labels"
    LABEL_FILTER_MY_PRIVATES = "my_privates"
    LABEL_FILTER_PRIVATES    = "privates"
    LABEL_FILTER_PUBLIC      = "public"
    LABEL_FILTER_ALL         = "all"
)
const (
    LABEL_REQUEST_STATUS_APPROVED = "approved"
    LABEL_REQUEST_STATUS_REJECTED = "rejected"
    LABEL_REQUEST_STATUS_FAILED   = "failed"
    LABEL_REQUEST_STATUS_CANCELED = "canceled"
    LABEL_REQUEST_STATUS_PENDING  = "pending"
)
const (
    PUBLIC_LABELS_ID = "_PUBLIC_LABELS"
)

type LabelManager struct{}

type Label struct {
    ID         string        `bson:"_id" json:"_id"`
    LowerTitle string        `bson:"lower_title" json:"lower_title"`
    Title      string        `bson:"title" json:"title"`
    Members    []string      `bson:"members" json:"members"`
    CreatorID  string        `bson:"creator_id" json:"creator_id"`
    ColourCode string        `bson:"colour_code" json:"colour_code"`
    Public     bool          `bson:"public" json:"public"`
    Counters   LabelCounters `bson:"counters" json:"counters"`
}
type LabelCounters struct {
    Tasks   int `bson:"tasks" json:"tasks"`
    Posts   int `bson:"posts" json:"posts"`
    Members int `bson:"members" json:"members"`
}
type LabelRequest struct {
    ID          bson.ObjectId `bson:"_id" json:"_id"`
    RequesterID string        `bson:"requester_id" json:"requester_id"`
    ResponderID string        `bson:"responder_id" json:"responder_id"`
    LabelID     string        `bson:"label_id" json:"label_id"`
    Title       string        `bson:"title" json:"title"`
    ColourCode  string        `bson:"colour_code" json:"colour_code"`
    Status      string        `bson:"status" json:"status"`
    Timestamp   uint64        `bson:"timestamp" json:"timestamp"`
    LastUpdate  uint64        `bson:"last_update" json:"last_update"`
}

func NewLabelManager() *LabelManager {
    return new(LabelManager)
}

//	AddMembers adds memberIDs to the collaborators of the labelID. The accounts who are in members list
//	of a label have the right access to add or remove the label of posts.
func (lm *LabelManager) AddMembers(labelID string, memberIDs []string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    // Update POSTS.LABELS collection
    if err := db.C(COLLECTION_LABELS).Update(
        bson.M{"_id": labelID, "members": bson.M{"$nin": memberIDs}},
        bson.M{
            "$addToSet": bson.M{"members": bson.M{"$each": memberIDs}},
            "$inc":      bson.M{"counters.members": len(memberIDs)},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Updates ACCOUNT.LABELS collection
    bulk := db.C(COLLECTION_ACCOUNTS_LABELS).Bulk()
    bulk.Unordered()
    for _, accountID := range memberIDs {
        bulk.Upsert(
            bson.M{"_id": accountID, "labels": bson.M{"$ne": labelID}},
            bson.M{
                "$addToSet": bson.M{"labels": labelID},
                "$inc":      bson.M{"qty": 1},
            },
        )
    }
    if _, err := bulk.Run(); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

//	CreatePrivate creates a new private label object in LABELS collection. Private labels can only be assigned
//	or removed by their members (collaborators) but labels are visible to everyone who access the labeled posts.
func (lm *LabelManager) CreatePrivate(id, title, code, creatorID string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if len(title) > DEFAULT_MAX_LABEL_TITLE {
        return false
    }
    label := Label{
        ID:         id,
        LowerTitle: strings.ToLower(title),
        Title:      title,
        CreatorID:  creatorID,
        Members:    []string{},
        ColourCode: code,
        Public:     false,
    }
    if err := db.C(COLLECTION_LABELS).Insert(label); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

//	CreatePublic creates a new public label object in LABELS collection. Public labels can be used by all the users.
func (lm *LabelManager) CreatePublic(id, title, code, creatorID string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    label := Label{
        ID:         id,
        LowerTitle: strings.ToLower(title),
        Title:      title,
        CreatorID:  creatorID,
        Members:    []string{},
        ColourCode: code,
        Public:     true,
    }
    if err := db.C(COLLECTION_LABELS).Insert(label); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // _PUBLIC_LABELS is a special document in ACCOUNTS.LABELS which all the public labels will
    // be added in this document
    if _, err := db.C(COLLECTION_ACCOUNTS_LABELS).Upsert(
        bson.M{
            "_id":    PUBLIC_LABELS_ID,
            "labels": bson.M{"$ne": label.ID},
        },
        bson.M{
            "$addToSet": bson.M{"labels": label.ID},
            "$inc":      bson.M{"qty": 1},
        },
    ); err != nil {
        _Log.Warn(err.Error())
    }

    return true
}

// CreateRequest creates a request object to be accepted/rejected by one of label managers
func (lm *LabelManager) CreateRequest(requesterID, labelID, title, colourCode string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ts := Timestamp()
    labelRequest := LabelRequest{
        ID:          bson.NewObjectId(),
        LabelID:     labelID,
        RequesterID: requesterID,
        Title:       title,
        ColourCode:  colourCode,
        Timestamp:   ts,
        LastUpdate:  ts,
        Status:      LABEL_REQUEST_STATUS_PENDING,
    }
    if err := db.C(COLLECTION_LABELS_REQUESTS).Insert(labelRequest); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

// GetByID returns a Label object identified by 'id'
func (lm *LabelManager) GetByID(id string) *Label {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    label := new(Label)
    if err := db.C(COLLECTION_LABELS).FindId(id).One(label); err != nil {
        _Log.Warn(err.Error())
        return nil
    }
    return label
}

// GetByIDs returns an array of Labels identified by []ids
func (lm *LabelManager) GetByIDs(ids []string) []Label {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    var labels []Label
    if err := db.C(COLLECTION_LABELS).Find(bson.M{"_id": bson.M{"$in": ids}}).All(&labels); err != nil {
        _Log.Warn(err.Error())
        return []Label{}
    }
    return labels
}

// GetByTitles returns an array of labels identified by title
func (lm *LabelManager) GetByTitles(titles []string) []Label {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    var labels []Label
    if err := db.C(COLLECTION_LABELS).Find(bson.M{"title": bson.M{"$in": titles}}).All(&labels); err != nil {
        _Log.Warn(err.Error())
        return []Label{}
    }
    return labels
}

// GetRequestByID returns the request object if request exists or return nil
func (lm *LabelManager) GetRequestByID(requestID bson.ObjectId) *LabelRequest {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    labelRequest := new(LabelRequest)
    if err := db.C(COLLECTION_LABELS_REQUESTS).FindId(requestID).One(labelRequest); err != nil {
        _Log.Warn(err.Error())
        return nil
    }
    return labelRequest
}

// GetRequests returns an array of LabelRequests filtered by status
// Pagination Supported (skip, limit)
func (lm *LabelManager) GetRequests(status string, pg Pagination) []LabelRequest {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    labelRequests := make([]LabelRequest, 0, pg.GetLimit())
    if err := db.C(COLLECTION_LABELS_REQUESTS).Find(
        bson.M{"status": status},
    ).Sort("-timestamp").Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&labelRequests); err != nil {
        _Log.Warn(err.Error())
    }
    return labelRequests
}

// GetRequestsByAccountID returns an array of LabelRequests sent by accountID and their status
// is still 'pending'
func (lm *LabelManager) GetRequestsByAccountID(accountID string, pg Pagination) []LabelRequest {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    labelRequests := make([]LabelRequest, 0, pg.GetLimit())
    if err := db.C(COLLECTION_LABELS_REQUESTS).Find(
        bson.M{
            "status":       LABEL_REQUEST_STATUS_PENDING,
            "requester_id": accountID,
        },
    ).Sort("-timestamp").Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&labelRequests); err != nil {
        _Log.Warn(err.Error())
    }
    return labelRequests
}

// IncrementCounter increase/decrease the counter value for label. valid counterName are:
//	1. posts
//	2. tasks
//	3. members
func (lm *LabelManager) IncrementCounter(labelID string, counterName string, value int) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_LABELS).UpdateId(
        labelID,
        bson.M{
            "$inc": bson.M{fmt.Sprintf("counters.%s", counterName): value},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

// Remove removes the label from the POSTS.LABELS collection
func (lm *LabelManager) Remove(labelID string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_LABELS).RemoveId(labelID); err != nil {
        log.Println("Model::LabelManager::Remove::Error 1::", err.Error())
        return false
    }

    // Update all the posts
    // TODO:: update posts in cache?!! or let them update gradually
    if _, err := db.C(COLLECTION_POSTS).UpdateAll(
        bson.M{"labels": labelID},
        bson.M{"$pull": bson.M{"labels": labelID}},
    ); err != nil {
        log.Println("Model::LabelManager::Remove::Error 2::", err.Error())
        return false
    }

    if _, err := db.C(COLLECTION_ACCOUNTS_LABELS).UpdateAll(
        bson.M{"labels": labelID},
        bson.M{
            "$pull": bson.M{"labels": labelID},
            "$inc":  bson.M{"qty": -1},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

// RemoveMember removes memberID from the collaborators list of the labelID
func (lm *LabelManager) RemoveMember(labelID, memberID string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_LABELS).Update(
        bson.M{"_id": labelID, "members": memberID},
        bson.M{
            "$pull": bson.M{"members": memberID},
            "$inc":  bson.M{"counters.members": -1},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    if err := db.C(COLLECTION_ACCOUNTS_LABELS).Update(
        bson.M{"_id": memberID, "labels": labelID},
        bson.M{
            "$pull": bson.M{"labels": labelID},
            "$inc":  bson.M{"qty": -1},
        },
    ); err != nil {
        _Log.Warn(err.Error())
    }
    return true
}

// SanitizeLabelCode if input code is not a valid code then it returns the default colour code
func (lm *LabelManager) SanitizeLabelCode(code string) string {
    // _funcName

    // removed LOG Function

    switch code {
    case LABEL_COLOUR_CODE_A, LABEL_COLOUR_CODE_B, LABEL_COLOUR_CODE_C,
        LABEL_COLOUR_CODE_D, LABEL_COLOUR_CODE_E, LABEL_COLOUR_CODE_F,
        LABEL_COLOUR_CODE_G:
    default:
        code = LABEL_COLOUR_CODE_A
    }
    return code
}

// TitleExists check if title is already used or not
func (lm *LabelManager) TitleExists(title string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    label := new(Label)
    if err := db.C(COLLECTION_LABELS).Find(bson.M{"title": title}).One(label); err != nil {
        return false
    }
    return true
}

// UpdateRequestStatus updates the status of the request
func (lm *LabelManager) UpdateRequestStatus(updaterAccountID string, requestID bson.ObjectId, status string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_LABELS_REQUESTS).UpdateId(
        requestID,
        bson.M{"$set": bson.M{
            "last_update":  Timestamp(),
            "status":       status,
            "responder_id": updaterAccountID,
        }},
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

// Update updates labelID by values in LabelUpdateRequest
// labelID must exists and if colourCode and title are not empty strings then they will be applied
func (lm *LabelManager) Update(labelID, colourCode, title string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    q := bson.M{}
    if len(colourCode) > 0 {
        q["colour_code"] = colourCode
    }
    if len(title) > 0 {
        q["lower_title"] = strings.ToLower(title)
        q["title"] = title
    }
    if err := db.C(COLLECTION_LABELS).UpdateId(labelID, bson.M{"$set": q}); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

// IsMember returns true if account in member of the label otherwise returns false
func (l *Label) IsMember(accountID string) bool {
    for _, memberID := range l.Members {
        if memberID == accountID {
            return true
        }
    }
    return false
}
