package nested

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"github.com/globalsign/mgo/bson"
	"go.uber.org/zap"
)

const (
	PlaceActivityActionMemberRemove  = 0x002
	PlaceActivityActionMemberJoin    = 0x008
	PlaceActivityActionPlaceAdd      = 0x010
	PlaceActivityActionPostAdd       = 0x100
	PlaceActivityActionPostRemove    = 0x105
	PlaceActivityActionPostRemoveAll = 0x106
	PlaceActivityActionPostMoveTo    = 0x206
	PlaceActivityActionPostMoveFrom  = 0x207
)

type PlaceActivityManager struct{}

func newPlaceActivityManager() *PlaceActivityManager {
	return new(PlaceActivityManager)
}

func (tm *PlaceActivityManager) Exists(activityID bson.ObjectId) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n, _ := db.C(global.CollectionPlacesActivities).FindId(activityID).Count()

	return n > 0
}

func (tm *PlaceActivityManager) GetByID(activityID bson.ObjectId) *PlaceActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	t := new(PlaceActivity)
	if err := db.C(global.CollectionPlacesActivities).FindId(activityID).One(&t); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}
	return t
}

func (tm *PlaceActivityManager) GetActivitiesByPlace(placeID string, pg Pagination) []PlaceActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	sortItem := "timestamp"
	sortDir := fmt.Sprintf("-%s", sortItem)
	q := bson.M{
		"place_id": placeID,
	}

	q, sortDir = pg.FillQuery(q, sortItem, sortDir)

	a := make([]PlaceActivity, 0, pg.GetLimit())
	db.C(global.CollectionPlacesActivities).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&a)
	return a
}

func (tm *PlaceActivityManager) PostAdd(actorID string, placeIDs []string, postID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	bulk := db.C(global.CollectionPlacesActivities).Bulk()
	bulk.Unordered()
	v := PlaceActivity{
		Timestamp:  ts,
		LastUpdate: ts,
		Action:     PlaceActivityActionPostAdd,
		Actor:      actorID,
		PostID:     postID,
	}

	for _, placeID := range placeIDs {
		v.ID = bson.NewObjectId()
		v.PlaceID = placeID
		bulk.Insert(v)
	}
	if _, err := bulk.Run(); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *PlaceActivityManager) PostAttachPlace(actorID, newPlaceID string, postID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := PlaceActivity{
		ID:         bson.NewObjectId(),
		Timestamp:  ts,
		LastUpdate: ts,
		Action:     PlaceActivityActionPostAdd,
		Actor:      actorID,
		NewPlaceID: newPlaceID,
		PostID:     postID,
		PlaceID:    newPlaceID,
	}
	if err := db.C(global.CollectionPlacesActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}

}

func (tm *PlaceActivityManager) PostMove(actorID, oldPlaceID, newPlaceID string, postID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	bulk := db.C(global.CollectionPlacesActivities).Bulk()
	bulk.Unordered()
	v := PlaceActivity{
		Timestamp:  ts,
		LastUpdate: ts,
		Actor:      actorID,
		PostID:     postID,
		OldPlaceID: oldPlaceID,
		NewPlaceID: newPlaceID,
	}

	v.ID = bson.NewObjectId()
	v.PlaceID = oldPlaceID
	v.Action = PlaceActivityActionPostMoveFrom
	bulk.Insert(v)

	v.ID = bson.NewObjectId()
	v.PlaceID = newPlaceID
	v.Action = PlaceActivityActionPostMoveTo
	bulk.Insert(v)

	if _, err := bulk.Run(); err != nil {
		log.Warn("Got error", zap.Error(err))
	}

	tm.PostRemove(actorID, oldPlaceID, postID)
}

func (tm *PlaceActivityManager) PostRemove(actorID, placeID string, postID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := PlaceActivity{
		ID:        bson.NewObjectId(),
		Timestamp: ts,
		Action:    PlaceActivityActionPostRemove,
		Actor:     actorID,
		PlaceID:   placeID,
		PostID:    postID,
	}
	if err := db.C(global.CollectionPlacesActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}

	return
}

func (tm *PlaceActivityManager) PostRemoveAll(actorID, placeID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := PlaceActivity{
		ID:        bson.NewObjectId(),
		Timestamp: ts,
		Action:    PlaceActivityActionPostRemoveAll,
		Actor:     actorID,
		PlaceID:   placeID,
	}
	if err := db.C(global.CollectionPlacesActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}

	return
}

func (tm *PlaceActivityManager) PlaceAdd(actor, placeID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := bson.M{
		"timestamp":   ts,
		"last_update": ts,
		"_removed":    false,
		"action":      PlaceActivityActionPlaceAdd,
		"actor":       actor,
		"place_id":    placeID,
	}
	db.C(global.CollectionPlacesActivities).Insert(v)
	return
}

func (tm *PlaceActivityManager) PlaceRemove(placeID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()

	// remove all the activities of PLACE_ADD
	if _, err := db.C(global.CollectionPlacesActivities).UpdateAll(
		bson.M{"action": PlaceActivityActionPlaceAdd, "place_id": placeID, "_removed": false},
		bson.M{"$set": bson.M{"_removed": true, "last_update": ts}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}

	// remove all the MEMBER_JOIN actions
	if _, err := db.C(global.CollectionPlacesActivities).UpdateAll(
		bson.M{"action": PlaceActivityActionMemberJoin, "place_id": placeID},
		bson.M{"$set": bson.M{"_removed": true, "last_update": ts}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *PlaceActivityManager) MemberRemove(actor, placeID, memberID string, reason string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := bson.M{
		"timestamp":   ts,
		"last_update": ts,
		"_removed":    false,
		"action":      PlaceActivityActionMemberRemove,
		"actor":       actor,
		"place_id":    placeID,
		"member_id":   memberID,
		"reason":      reason}
	db.C(global.CollectionPlacesActivities).Insert(v)
	return
}

func (tm *PlaceActivityManager) MemberJoin(actor, placeID, by string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := bson.M{
		"timestamp":   ts,
		"last_update": ts,
		"_removed":    false,
		"action":      PlaceActivityActionMemberJoin,
		"actor":       actor,
		"place_id":    placeID,
		"by":          by}
	db.C(global.CollectionPlacesActivities).Insert(v)
	return
}

type PlaceActivity struct {
	ID            bson.ObjectId `json:"_id" bson:"_id"`
	Action        int           `json:"action" bson:"action"`
	Actor         string        `json:"actor" bson:"actor"`
	Timestamp     uint64        `json:"timestamp" bson:"timestamp"`
	LastUpdate    uint64        `json:"last_update" bson:"last_update"`
	CommentID     bson.ObjectId `json:"comment_id,omitempty" bson:"comment_id,omitempty"`
	LabelID       string        `json:"label_id,omitempty" bson:"label_id,omitempty"`
	MemberID      string        `json:"member_id" bson:"member_id"`
	NewPlaceID    string        `json:"new_place_id" bson:"new_place_id"`
	OldPlaceID    string        `json:"old_place_id" bson:"old_place_id"`
	PlaceID       string        `json:"place_id" bson:"place_id"`
	PostID        bson.ObjectId `json:"post_id,omitempty" bson:"post_id,omitempty"`
	RemovedPlaces []string      `json:"removed_places" bson:"removed_places"`
}
