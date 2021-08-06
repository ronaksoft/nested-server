package nested

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"go.uber.org/zap"
	"strings"

	"github.com/globalsign/mgo/bson"
)

const (
	LabelColourCodeA = "A"
	LabelColourCodeB = "B"
	LabelColourCodeC = "C"
	LabelColourCodeD = "D"
	LabelColourCodeE = "E"
	LabelColourCodeF = "F"
	LabelColourCodeG = "G"
)
const (
	LabelFilterMyLabels   = "my_labels"
	LabelFilterMyPrivates = "my_privates"
	LabelFilterPrivates   = "privates"
	LabelFilterPublic     = "public"
	LabelFilterAll        = "all"
)
const (
	LabelRequestStatusApproved = "approved"
	LabelRequestStatusRejected = "rejected"
	LabelRequestStatusFailed   = "failed"
	LabelRequestStatusCanceled = "canceled"
	LabelRequestStatusPending  = "pending"
)
const (
	PublicLabelsID = "_PUBLIC_LABELS"
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

func newLabelManager() *LabelManager {
	return new(LabelManager)
}

//	AddMembers adds memberIDs to the collaborators of the labelID. The accounts who are in members list
//	of a label have the right access to add or remove the label of posts.
func (lm *LabelManager) AddMembers(labelID string, memberIDs []string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	// Update POSTS.LABELS collection
	if err := db.C(global.CollectionLabels).Update(
		bson.M{"_id": labelID, "members": bson.M{"$nin": memberIDs}},
		bson.M{
			"$addToSet": bson.M{"members": bson.M{"$each": memberIDs}},
			"$inc":      bson.M{"counters.members": len(memberIDs)},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}

	// Updates ACCOUNT.LABELS collection
	bulk := db.C(global.CollectionAccountsLabels).Bulk()
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
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

//	CreatePrivate creates a new private label object in LABELS collection. Private labels can only be assigned
//	or removed by their members (collaborators) but labels are visible to everyone who access the labeled posts.
func (lm *LabelManager) CreatePrivate(id, title, code, creatorID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if len(title) > global.DefaultMaxLabelTitle {
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
	if err := db.C(global.CollectionLabels).Insert(label); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

//	CreatePublic creates a new public label object in LABELS collection. Public labels can be used by all the users.
func (lm *LabelManager) CreatePublic(id, title, code, creatorID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
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
	if err := db.C(global.CollectionLabels).Insert(label); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}

	// _PUBLIC_LABELS is a special document in ACCOUNTS.LABELS which all the public labels will
	// be added in this document
	if _, err := db.C(global.CollectionAccountsLabels).Upsert(
		bson.M{
			"_id":    PublicLabelsID,
			"labels": bson.M{"$ne": label.ID},
		},
		bson.M{
			"$addToSet": bson.M{"labels": label.ID},
			"$inc":      bson.M{"qty": 1},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}

	return true
}

// CreateRequest creates a request object to be accepted/rejected by one of label managers
func (lm *LabelManager) CreateRequest(requesterID, labelID, title, colourCode string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
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
		Status:      LabelRequestStatusPending,
	}
	if err := db.C(global.CollectionLabelsRequests).Insert(labelRequest); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

// GetByID returns a Label object identified by 'id'
func (lm *LabelManager) GetByID(id string) *Label {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	label := new(Label)
	if err := db.C(global.CollectionLabels).FindId(id).One(label); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}
	return label
}

// GetByIDs returns an array of Labels identified by []ids
func (lm *LabelManager) GetByIDs(ids []string) []Label {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	var labels []Label
	if err := db.C(global.CollectionLabels).Find(bson.M{"_id": bson.M{"$in": ids}}).All(&labels); err != nil {
		log.Warn("Got error", zap.Error(err))
		return []Label{}
	}
	return labels
}

// GetByTitles returns an array of labels identified by title
func (lm *LabelManager) GetByTitles(titles []string) []Label {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	var labels []Label
	if err := db.C(global.CollectionLabels).Find(bson.M{"title": bson.M{"$in": titles}}).All(&labels); err != nil {
		log.Warn("Got error", zap.Error(err))
		return []Label{}
	}
	return labels
}

// GetRequestByID returns the request object if request exists or return nil
func (lm *LabelManager) GetRequestByID(requestID bson.ObjectId) *LabelRequest {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	labelRequest := new(LabelRequest)
	if err := db.C(global.CollectionLabelsRequests).FindId(requestID).One(labelRequest); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}
	return labelRequest
}

// GetRequests returns an array of LabelRequests filtered by status
// Pagination Supported (skip, limit)
func (lm *LabelManager) GetRequests(status string, pg Pagination) []LabelRequest {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	labelRequests := make([]LabelRequest, 0, pg.GetLimit())
	if err := db.C(global.CollectionLabelsRequests).Find(
		bson.M{"status": status},
	).Sort("-timestamp").Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&labelRequests); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return labelRequests
}

// GetRequestsByAccountID returns an array of LabelRequests sent by accountID and their status
// is still 'pending'
func (lm *LabelManager) GetRequestsByAccountID(accountID string, pg Pagination) []LabelRequest {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	labelRequests := make([]LabelRequest, 0, pg.GetLimit())
	if err := db.C(global.CollectionLabelsRequests).Find(
		bson.M{
			"status":       LabelRequestStatusPending,
			"requester_id": accountID,
		},
	).Sort("-timestamp").Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&labelRequests); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return labelRequests
}

// IncrementCounter increase/decrease the counter value for label. valid counterName are:
//	1. posts
//	2. tasks
//	3. members
func (lm *LabelManager) IncrementCounter(labelID string, counterName string, value int) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionLabels).UpdateId(
		labelID,
		bson.M{
			"$inc": bson.M{fmt.Sprintf("counters.%s", counterName): value},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

// Remove removes the label from the POSTS.LABELS collection
func (lm *LabelManager) Remove(labelID string) bool {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionLabels).RemoveId(labelID); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}

	// Update all the posts
	// TODO:: update posts in cache?!! or let them update gradually
	if _, err := db.C(global.CollectionPosts).UpdateAll(
		bson.M{"labels": labelID},
		bson.M{"$pull": bson.M{"labels": labelID}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}

	if _, err := db.C(global.CollectionAccountsLabels).UpdateAll(
		bson.M{"labels": labelID},
		bson.M{
			"$pull": bson.M{"labels": labelID},
			"$inc":  bson.M{"qty": -1},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

// RemoveMember removes memberID from the collaborators list of the labelID
func (lm *LabelManager) RemoveMember(labelID, memberID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionLabels).Update(
		bson.M{"_id": labelID, "members": memberID},
		bson.M{
			"$pull": bson.M{"members": memberID},
			"$inc":  bson.M{"counters.members": -1},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	if err := db.C(global.CollectionAccountsLabels).Update(
		bson.M{"_id": memberID, "labels": labelID},
		bson.M{
			"$pull": bson.M{"labels": labelID},
			"$inc":  bson.M{"qty": -1},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return true
}

// SanitizeLabelCode if input code is not a valid code then it returns the default colour code
func (lm *LabelManager) SanitizeLabelCode(code string) string {
	switch code {
	case LabelColourCodeA, LabelColourCodeB, LabelColourCodeC,
		LabelColourCodeD, LabelColourCodeE, LabelColourCodeF,
		LabelColourCodeG:
	default:
		code = LabelColourCodeA
	}
	return code
}

// TitleExists check if title is already used or not
func (lm *LabelManager) TitleExists(title string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	label := new(Label)
	if err := db.C(global.CollectionLabels).Find(bson.M{"title": title}).One(label); err != nil {
		return false
	}
	return true
}

// UpdateRequestStatus updates the status of the request
func (lm *LabelManager) UpdateRequestStatus(updaterAccountID string, requestID bson.ObjectId, status string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionLabelsRequests).UpdateId(
		requestID,
		bson.M{"$set": bson.M{
			"last_update":  Timestamp(),
			"status":       status,
			"responder_id": updaterAccountID,
		}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

// Update updates labelID by values in LabelUpdateRequest
// labelID must exists and if colourCode and title are not empty strings then they will be applied
func (lm *LabelManager) Update(labelID, colourCode, title string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	q := bson.M{}
	if len(colourCode) > 0 {
		q["colour_code"] = colourCode
	}
	if len(title) > 0 {
		q["lower_title"] = strings.ToLower(title)
		q["title"] = title
	}
	if err := db.C(global.CollectionLabels).UpdateId(labelID, bson.M{"$set": q}); err != nil {
		log.Warn("Got error", zap.Error(err))
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
