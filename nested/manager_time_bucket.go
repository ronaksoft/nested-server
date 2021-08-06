package nested

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"go.uber.org/zap"
	"time"

	"github.com/globalsign/mgo/bson"
)

type TimeBucket struct {
	ID            string          `bson:"_id"`
	OverdueTasks  []bson.ObjectId `bson:"overdue_tasks"`
	TaskReminders []bson.ObjectId `bson:"task_reminders"`
	DeferPosts    []bson.ObjectId `bson:"deferred_posts"`
}

type TimeBucketManager struct{}

func newTimeBucketManager() *TimeBucketManager {
	return new(TimeBucketManager)
}

func (bm *TimeBucketManager) GetBucketID(timestamp uint64) string {
	t := time.Unix(int64(timestamp/1000), 0)
	year, month, day := t.Date()
	hour, min, _ := t.Clock()
	return fmt.Sprintf("%d-%02d-%02d.%02d:%02d", year, month, day, hour, min)
}

func (bm *TimeBucketManager) GetBucketsBefore(timestamp uint64) []TimeBucket {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	bucketID := _Manager.TimeBucket.GetBucketID(timestamp)
	buckets := make([]TimeBucket, 0)
	if err := db.C(global.CollectionTimeBuckets).Find(
		bson.M{"_id": bson.M{"$lt": bucketID}},
	).All(&buckets); err != nil {
		log.Warn("got error on GetBucketsBefore", zap.Error(err))
	}
	return buckets
}

func (bm *TimeBucketManager) GetByID(bucketID string) *TimeBucket {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	bucket := new(TimeBucket)
	if err := db.C(global.CollectionTimeBuckets).FindId(bucketID).One(bucket); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}
	return bucket
}

func (bm *TimeBucketManager) AddOverdueTask(timestamp uint64, taskID bson.ObjectId) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	bucketID := bm.GetBucketID(timestamp)
	if _, err := db.C(global.CollectionTimeBuckets).Upsert(
		bson.M{"_id": bucketID},
		bson.M{"$addToSet": bson.M{"overdue_tasks": taskID}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

func (bm *TimeBucketManager) RemoveOverdueTask(timestamp uint64, taskID bson.ObjectId) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	bucketID := bm.GetBucketID(timestamp)
	if err := db.C(global.CollectionTimeBuckets).Update(
		bson.M{"_id": bucketID, "overdue_tasks": taskID},
		bson.M{"$pull": bson.M{"overdue_tasks": taskID}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

func (bm *TimeBucketManager) Remove(bucketID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionTimeBuckets).RemoveId(bucketID); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}
