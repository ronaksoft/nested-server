package nested

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"

	"github.com/globalsign/mgo/bson"
)

type PostActivityManager struct{}

func NewPostActivityManager() *PostActivityManager {
	return new(PostActivityManager)
}

func (pm *PostActivityManager) Remove(activityID bson.ObjectId) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_POSTS_ACTIVITIES).UpdateId(
		activityID,
		bson.M{"$set": bson.M{"_removed": true}},
	); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

func (pm *PostActivityManager) GetActivityByID(activityID bson.ObjectId) *PostActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	postActivity := new(PostActivity)
	if err := db.C(global.COLLECTION_POSTS_ACTIVITIES).FindId(activityID).One(postActivity); err != nil {
		log.Warn(err.Error())
		return nil
	}
	return postActivity
}

func (pm *PostActivityManager) GetActivitiesByIDs(activityIDs []bson.ObjectId) []PostActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	postActivities := make([]PostActivity, 0, len(activityIDs))
	if err := db.C(global.COLLECTION_POSTS_ACTIVITIES).Find(
		bson.M{"_id": bson.M{"$in": activityIDs}},
	).All(&postActivities); err != nil {
		log.Warn(err.Error())
		return nil
	}
	return postActivities
}

func (pm *PostActivityManager) GetActivitiesByPostID(postID bson.ObjectId, pg Pagination, filter []global.PostAction) []PostActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	postActivities := make([]PostActivity, pg.GetLimit())
	sortItem := "timestamp"
	sortDir := fmt.Sprintf("-%s", sortItem)
	q := bson.M{
		"post_id":  postID,
		"_removed": false,
	}
	q, sortDir = pg.FillQuery(q, sortItem, sortDir)

	if len(filter) > 0 {
		q["action"] = bson.M{"$in": filter}
	}

	Q := db.C(global.COLLECTION_POSTS_ACTIVITIES).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
	// Log Explain Query

	if err := Q.All(&postActivities); err != nil {
		log.Warn(err.Error())
		return []PostActivity{}
	}
	return postActivities
}

func (pm *PostActivityManager) CommentAdd(postID bson.ObjectId, actorID string, commentID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := PostActivity{
		ID:        bson.NewObjectId(),
		PostID:    postID,
		Timestamp: ts,
		Action:    global.PostActivityActionCommentAdd,
		ActorID:   actorID,
		CommentID: commentID,
	}

	if err := db.C(global.COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}

func (pm *PostActivityManager) CommentRemove(postID bson.ObjectId, actorID string, commentID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := PostActivity{
		ID:        bson.NewObjectId(),
		PostID:    postID,
		Timestamp: ts,
		Action:    global.PostActivityActionCommentRemove,
		ActorID:   actorID,
		CommentID: commentID,
	}

	if err := db.C(global.COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}

func (pm *PostActivityManager) LabelAdd(postID bson.ObjectId, actorID string, labelID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := PostActivity{
		ID:        bson.NewObjectId(),
		PostID:    postID,
		Timestamp: ts,
		Action:    global.PostActivityActionLabelAdd,
		ActorID:   actorID,
		LabelID:   labelID,
	}

	if err := db.C(global.COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}

func (pm *PostActivityManager) LabelRemove(postID bson.ObjectId, actorID string, labelID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := PostActivity{
		ID:        bson.NewObjectId(),
		PostID:    postID,
		Timestamp: ts,
		Action:    global.PostActivityActionLabelRemove,
		ActorID:   actorID,
		LabelID:   labelID,
	}

	if err := db.C(global.COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}

func (pm *PostActivityManager) PlaceMove(postID bson.ObjectId, actorID string, oldPlaceID string, newPlaceID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := PostActivity{
		ID:         bson.NewObjectId(),
		PostID:     postID,
		Timestamp:  ts,
		Action:     global.PostActivityActionPlaceMove,
		ActorID:    actorID,
		OldPlaceID: oldPlaceID,
		NewPlaceID: newPlaceID,
	}

	if err := db.C(global.COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}

func (pm *PostActivityManager) PlaceAttached(postID bson.ObjectId, actorID string, newPlaceID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := PostActivity{
		ID:         bson.NewObjectId(),
		PostID:     postID,
		Timestamp:  ts,
		Action:     global.PostActivityActionPlaceAttach,
		ActorID:    actorID,
		NewPlaceID: newPlaceID,
	}

	if err := db.C(global.COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}

func (pm *PostActivityManager) Edit(postID bson.ObjectId, actorID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	v := PostActivity{
		ID:        bson.NewObjectId(),
		PostID:    postID,
		Timestamp: ts,
		Action:    global.PostActivityActionEdited,
		ActorID:   actorID,
	}

	if err := db.C(global.COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return

}

type PostActivity struct {
	ID           bson.ObjectId     `bson:"_id" json:"_id"`
	PostID       bson.ObjectId     `bson:"post_id" json:"post_id"`
	Timestamp    uint64            `bson:"timestamp" json:"timestamp"`
	Action       global.PostAction `bson:"action" json:"action"`
	ActorID      string            `bson:"actor_id" json:"actor_id"`
	AttachmentID UniversalID       `bson:"attachment_id" json:"attachment_id,omitempty"`
	CommentID    bson.ObjectId     `bson:"comment_id,omitempty" json:"comment_id,omitempty"`
	LabelID      string            `bson:"label_id" json:"label_id,omitempty"`
	OldPlaceID   string            `bson:"old_place_id,omitempty" json:"old_place_id,omitempty"`
	NewPlaceID   string            `bson:"new_place_id,omitempty" json:"new_place_id,omitempty"`
	Removed      bool              `bson:"_removed" json:"_removed,omitempty"`
	RemovedBy    string            `bson:"removed_by,omitempty" json:"removed_by,omitempty"`
}
