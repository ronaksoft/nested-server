package nested

import (
    "fmt"
    "github.com/globalsign/mgo/bson"
)

// POST ACTIVITY ACTIONS
const (
    POST_ACTIVITY_ACTION_COMMENT_ADD    PostAction = 0x002
    POST_ACTIVITY_ACTION_COMMENT_REMOVE PostAction = 0x003
    POST_ACTIVITY_ACTION_LABEL_ADD      PostAction = 0x011
    POST_ACTIVITY_ACTION_LABEL_REMOVE   PostAction = 0x012
    POST_ACTIVITY_ACTION_EDITED         PostAction = 0x015
    POST_ACTIVITY_ACTION_PLACE_MOVE     PostAction = 0x016
    POST_ACTIVITY_ACTION_PLACE_ATTACH   PostAction = 0x017
)

type PostActivityManager struct{}

func NewPostActivityManager() *PostActivityManager {
    return new(PostActivityManager)
}
func (pm *PostActivityManager) Remove(activityID bson.ObjectId) bool {
    _funcName := "PostActivityManager::Remove"
    _Log.FunctionStarted(_funcName, activityID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_POSTS_ACTIVITIES).UpdateId(
        activityID,
        bson.M{"$set": bson.M{"_removed": true}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }
    return true
}
func (pm *PostActivityManager) GetActivityByID(activityID bson.ObjectId) *PostActivity {
    _funcName := "PostActivityManager::GetActivityByID"
    _Log.FunctionStarted(_funcName, activityID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    postActivity := new(PostActivity)
    if err := db.C(COLLECTION_POSTS_ACTIVITIES).FindId(activityID).One(postActivity); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    return postActivity
}
func (pm *PostActivityManager) GetActivitiesByIDs(activityIDs []bson.ObjectId) []PostActivity {
    _funcName := "PostActivityManager::GetActivitiesByIDs"
    _Log.FunctionStarted(_funcName)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    postActivities := make([]PostActivity, 0, len(activityIDs))
    if err := db.C(COLLECTION_POSTS_ACTIVITIES).Find(
        bson.M{"_id": bson.M{"$in": activityIDs}},
    ).All(&postActivities); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    return postActivities
}
func (pm *PostActivityManager) GetActivitiesByPostID(postID bson.ObjectId, pg Pagination, filter []PostAction) []PostActivity {
    _funcName := "PostActivityManager::GetActivitiesByPostID"
    _Log.FunctionStarted(_funcName)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    postActivities := make([]PostActivity, pg.GetLimit())
    sortItem := "timestamp"
    sortDir := fmt.Sprintf("-%s", sortItem)
    q := bson.M{
        "post_id":  postID,
        "_removed": false,
    }
    if pg.After > 0 {
        q[sortItem] = bson.M{"$gt": pg.After}
        sortDir = sortItem
    } else if pg.Before > 0 {
        q[sortItem] = bson.M{"$lt": pg.Before}
    }
    if len(filter) > 0 {
        q["action"] = bson.M{"$in": filter}
    }

    Q := db.C(COLLECTION_POSTS_ACTIVITIES).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
    _Log.ExplainQuery(_funcName, Q)

    if err := Q.All(&postActivities); err != nil {
        _Log.Error(_funcName, err.Error())
        return []PostActivity{}
    }
    return postActivities
}

func (pm *PostActivityManager) CommentAdd(postID bson.ObjectId, actorID string, commentID bson.ObjectId) {
    _funcName := "PostActivityManager::CommentAdd"
    _Log.FunctionStarted(_funcName, actorID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ts := Timestamp()
    v := PostActivity{
        ID:        bson.NewObjectId(),
        PostID:    postID,
        Timestamp: ts,
        Action:    POST_ACTIVITY_ACTION_COMMENT_ADD,
        ActorID:   actorID,
        CommentID: commentID,
    }

    if err := db.C(COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return
}

func (pm *PostActivityManager) CommentRemove(postID bson.ObjectId, actorID string, commentID bson.ObjectId) {
    _funcName := "PostActivityManager::CommentRemove"
    _Log.FunctionStarted(_funcName, actorID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ts := Timestamp()
    v := PostActivity{
        ID:        bson.NewObjectId(),
        PostID:    postID,
        Timestamp: ts,
        Action:    POST_ACTIVITY_ACTION_COMMENT_REMOVE,
        ActorID:   actorID,
        CommentID: commentID,
    }

    if err := db.C(COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return
}

func (pm *PostActivityManager) LabelAdd(postID bson.ObjectId, actorID string, labelID string) {
    _funcName := "PostActivityManager::LabelAdd"
    _Log.FunctionStarted(_funcName, actorID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ts := Timestamp()
    v := PostActivity{
        ID:        bson.NewObjectId(),
        PostID:    postID,
        Timestamp: ts,
        Action:    POST_ACTIVITY_ACTION_LABEL_ADD,
        ActorID:   actorID,
        LabelID:   labelID,
    }

    if err := db.C(COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return
}

func (pm *PostActivityManager) LabelRemove(postID bson.ObjectId, actorID string, labelID string) {
    _funcName := "PostActivityManager::LabelRemove"
    _Log.FunctionStarted(_funcName, actorID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ts := Timestamp()
    v := PostActivity{
        ID:        bson.NewObjectId(),
        PostID:    postID,
        Timestamp: ts,
        Action:    POST_ACTIVITY_ACTION_LABEL_REMOVE,
        ActorID:   actorID,
        LabelID:   labelID,
    }

    if err := db.C(COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return
}

func (pm *PostActivityManager) PlaceMove(postID bson.ObjectId, actorID string, oldPlaceID string, newPlaceID string) {
    _funcName := "PostActivityManager::PlaceMove"
    _Log.FunctionStarted(_funcName, actorID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ts := Timestamp()
    v := PostActivity{
        ID:         bson.NewObjectId(),
        PostID:     postID,
        Timestamp:  ts,
        Action:     POST_ACTIVITY_ACTION_PLACE_MOVE,
        ActorID:    actorID,
        OldPlaceID: oldPlaceID,
        NewPlaceID: newPlaceID,
    }

    if err := db.C(COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return
}

func (pm *PostActivityManager) PlaceAttached(postID bson.ObjectId, actorID string, newPlaceID string) {
    _funcName := "PostActivityManager::PlaceAttached"
    _Log.FunctionStarted(_funcName, actorID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ts := Timestamp()
    v := PostActivity{
        ID:         bson.NewObjectId(),
        PostID:     postID,
        Timestamp:  ts,
        Action:     POST_ACTIVITY_ACTION_PLACE_ATTACH,
        ActorID:    actorID,
        NewPlaceID: newPlaceID,
    }

    if err := db.C(COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return
}

func (pm *PostActivityManager) Edit(postID bson.ObjectId, actorID string) {
    _funcName := "PostActivityManager::Edit"
    _Log.FunctionStarted(_funcName, actorID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ts := Timestamp()
    v := PostActivity{
        ID:        bson.NewObjectId(),
        PostID:    postID,
        Timestamp: ts,
        Action:    POST_ACTIVITY_ACTION_EDITED,
        ActorID:   actorID,
    }

    if err := db.C(COLLECTION_POSTS_ACTIVITIES).Insert(v); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return

}

type PostAction int
type PostActivity struct {
    ID           bson.ObjectId `bson:"_id" json:"_id"`
    PostID       bson.ObjectId `bson:"post_id" json:"post_id"`
    Timestamp    uint64        `bson:"timestamp" json:"timestamp"`
    Action       PostAction    `bson:"action" json:"action"`
    ActorID      string        `bson:"actor_id" json:"actor_id"`
    AttachmentID UniversalID   `bson:"attachment_id" json:"attachment_id,omitempty"`
    CommentID    bson.ObjectId `bson:"comment_id,omitempty" json:"comment_id,omitempty"`
    LabelID      string        `bson:"label_id" json:"label_id,omitempty"`
    OldPlaceID   string        `bson:"old_place_id,omitempty" json:"old_place_id,omitempty"`
    NewPlaceID   string        `bson:"new_place_id,omitempty" json:"new_place_id,omitempty"`
    Removed      bool          `bson:"_removed" json:"_removed,omitempty"`
    RemovedBy    string        `bson:"removed_by,omitempty" json:"removed_by,omitempty"`
}
