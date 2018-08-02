package nested

import (
    "github.com/globalsign/mgo/bson"
    "time"
    "fmt"
)

type TimeBucket struct {
    ID            string          `bson:"_id"`
    OverdueTasks  []bson.ObjectId `bson:"overdue_tasks"`
    TaskReminders []bson.ObjectId `bson:"task_reminders"`
    DeferPosts    []bson.ObjectId `bson:"deferred_posts"`
}

type TimeBucketManager struct{}

func NewTimeBucketManager() *TimeBucketManager {
    return new(TimeBucketManager)
}

func (bm *TimeBucketManager) GetBucketID(timestamp uint64) string {
    t := time.Unix(int64(timestamp/1000), 0)
    year, month, day := t.Date()
    hour, min, _ := t.Clock()
    return fmt.Sprintf("%d-%02d-%02d.%02d:%02d", year, month, day, hour, min)
}

func (bm *TimeBucketManager) GetBucketsBefore(timestamp uint64) []TimeBucket {
    _funcName := "TimeBucketManager::GetBucketsBefore"
    _Log.FunctionStarted(_funcName, timestamp)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    bucketID := _Manager.TimeBucket.GetBucketID(timestamp)
    buckets := make([]TimeBucket, 0)
    if err := db.C(COLLECTION_TIME_BUCKETS).Find(
        bson.M{"_id": bson.M{"$lt": bucketID}},
    ).All(&buckets); err != nil {
        _Log.Error(_funcName, err.Error(), timestamp, bucketID)
    }
    return buckets
}

func (bm *TimeBucketManager) GetByID(bucketID string) *TimeBucket {
    _funcName := "TimeBucketManager::GetByID"
    _Log.FunctionStarted(_funcName, bucketID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    bucket := new(TimeBucket)
    if err := db.C(COLLECTION_TIME_BUCKETS).FindId(bucketID).One(bucket); err != nil {
        _Log.Error(_funcName, err.Error(), bucketID)
        return nil
    }
    return bucket
}

func (bm *TimeBucketManager) AddOverdueTask(timestamp uint64, taskID bson.ObjectId) bool {
    _funcName := "TimeBucketManager::AddOverdueTask"
    _Log.FunctionStarted(_funcName, taskID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    bucketID := bm.GetBucketID(timestamp)
    if _, err := db.C(COLLECTION_TIME_BUCKETS).Upsert(
        bson.M{"_id": bucketID},
        bson.M{"$addToSet": bson.M{"overdue_tasks": taskID}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }
    return true
}

func (bm *TimeBucketManager) RemoveOverdueTask(timestamp uint64, taskID bson.ObjectId) bool {
    _funcName := "TimeBucketManager::RemoveOverdueTask"
    _Log.FunctionStarted(_funcName, taskID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    bucketID := bm.GetBucketID(timestamp)
    if err := db.C(COLLECTION_TIME_BUCKETS).Update(
        bson.M{"_id": bucketID, "overdue_tasks": taskID},
        bson.M{"$pull": bson.M{"overdue_tasks": taskID}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }
    return true
}

func (bm *TimeBucketManager) Remove(bucketID string) bool {
    _funcName := "TimeBucketManager::Remove"
    _Log.FunctionStarted(_funcName, bucketID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TIME_BUCKETS).RemoveId(bucketID); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }
    return true
}
