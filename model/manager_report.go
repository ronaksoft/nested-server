package nested

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"go.uber.org/zap"
	"strconv"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/gomodule/redigo/redis"
)

const (
	ReportCounterPostAdd                 = "post_add"
	ReportCounterPostExternalAdd         = "post_ext_add"
	ReportCounterPostAttachSize          = "post_attach_size"
	ReportCounterPostAttachCount         = "post_attach_count"
	ReportCounterPostPerAccount          = "post_per_account"
	ReportCounterPostPerPlace            = "post_per_place"
	ReportCounterCommentAdd              = "comment_add"
	ReportCounterCommentPerAccount       = "comment_per_account"
	ReportCounterCommentPerPlace         = "comment_per_place"
	ReportCounterTaskAdd                 = "task_add"
	ReportCounterTaskComment             = "task_comment"
	ReportCounterTaskCompleted           = "task_completed"
	ReportCounterTaskAddPerAccount       = "task_add_per_account"
	ReportCounterTaskCompletedPerAccount = "task_completed_per_account"
	ReportCounterTaskAssignedPerAccount  = "task_assigned_per_account"
	ReportCounterTaskCommentPerAccount   = "task_comment_per_account"
	ReportCounterSessionLogin            = "session_login"
	ReportCounterSessionRecall           = "session_recall"
	ReportCounterRequests                = "requests"
	ReportCounterDataIn                  = "data_in"
	ReportCounterDataOut                 = "data_out"
	ReportCounterProcessTime             = "process_time"
)

const (
	ReportResolutionHour  string = "h"
	ReportResolutionDay   string = "d"
	ReportResolutionMonth string = "m"
)

type TimeSeriesSingleValueHourly struct {
	ID     bson.ObjectId `bson:"_id" json:"-"`
	Date   string        `bson:"date" json:"date"`
	Key    string        `bson:"key" json:"key"`
	Sum    int           `bson:"sum" json:"sum"`
	Values MI            `bson:"values" json:"values"`
}

type ReportManager struct{}

func NewReportManager() *ReportManager {
	return new(ReportManager)
}

func (rcm *ReportManager) CountAPI(cmd string) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:api")
	c.Do("HINCRBY", key, cmd, 1)
}

func (rcm *ReportManager) CountPostAdd() {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostAdd)
	c.Do("INCR", key)
}

func (rcm *ReportManager) CountPostExternalAdd() {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostExternalAdd)
	c.Do("INCR", key)
}

func (rcm *ReportManager) CountPostAttachSize(n int64) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostAttachSize)
	c.Do("INCRBY", key, n)
}

func (rcm *ReportManager) CountPostAttachCount(n int) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostAttachCount)
	c.Do("INCRBY", key, n)
}

func (rcm *ReportManager) CountPostPerAccount(accountID string) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostPerAccount)
	c.Do("HINCRBY", key, accountID, 1)
}

func (rcm *ReportManager) CountPostPerPlace(placeIDs []string) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostPerPlace)
	for _, placeID := range placeIDs {
		c.Send("HINCRBY", key, placeID, 1)
	}
	c.Flush()
}

func (rcm *ReportManager) CountCommentAdd() {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterCommentAdd)
	c.Do("INCR", key)
}

func (rcm *ReportManager) CountCommentPerAccount(accountID string) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterCommentPerAccount)
	c.Do("HINCRBY", key, accountID, 1)
}

func (rcm *ReportManager) CountCommentPerPlace(placeIDs []string) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterCommentPerPlace)
	for _, placeID := range placeIDs {
		c.Send("HINCRBY", key, placeID, 1)
	}
	c.Flush()
}

func (rcm *ReportManager) CountTaskAdd() {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskAdd)
	c.Do("INCR", key)
}

func (rcm *ReportManager) CountTaskComment() {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskComment)
	c.Do("INCR", key)
}

func (rcm *ReportManager) CountTaskCompleted() {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskCompleted)
	c.Do("INCR", key)
}

func (rcm *ReportManager) CountTaskAddPerAccount(accountID string) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskAddPerAccount)
	c.Do("HINCRBY", key, accountID, 1)
}

func (rcm *ReportManager) CountTaskAssignedPerAccount(accountID string) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskAssignedPerAccount)
	c.Do("HINCRBY", key, accountID, 1)
}

func (rcm *ReportManager) CountTaskCommentPerAccount(accountID string) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskCommentPerAccount)
	c.Do("HINCRBY", key, accountID, 1)
}

func (rcm *ReportManager) CountTaskCompletedPerAccount(accountID string) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskCompletedPerAccount)
	c.Do("HINCRBY", key, accountID, 1)
}

func (rcm *ReportManager) CountSessionLogin() {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterSessionLogin)
	c.Do("INCR", key)
}

func (rcm *ReportManager) CountSessionRecall() {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterSessionRecall)
	c.Do("INCR", key)
}

func (rcm *ReportManager) CountRequests() {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterRequests)
	c.Do("INCR", key)
}

func (rcm *ReportManager) CountDataIn(n int) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterDataIn)
	c.Do("INCRBY", key, n)
}

func (rcm *ReportManager) CountDataOut(n int) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterDataOut)
	c.Do("INCRBY", key, n)
}

func (rcm *ReportManager) CountProcessTime(n int) {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterProcessTime)
	c.Do("INCRBY", key, n)
}

func (rcm *ReportManager) getAPI() MS {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:api")
	ms, _ := redis.StringMap(c.Do("HGETALL", key))
	return ms
}

func (rcm *ReportManager) getPostAdd() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostAdd)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getPostExternalAdd() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostExternalAdd)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getPostAttachSize() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostAttachSize)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getPostAttachCount() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostAttachCount)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getPostPerAccount() MI {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostPerAccount)
	mi, _ := redis.IntMap(c.Do("HGETALL", key))
	return mi
}

func (rcm *ReportManager) getPostPerPlace() MI {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterPostPerPlace)
	mi, _ := redis.IntMap(c.Do("HGETALL", key))
	return mi
}

func (rcm *ReportManager) getCommentAdd() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterCommentAdd)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getCommentPerAccount() MI {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterCommentPerAccount)
	mi, _ := redis.IntMap(c.Do("HGETALL", key))
	return mi
}

func (rcm *ReportManager) getCommentPerPlace() MI {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterCommentPerPlace)
	mi, _ := redis.IntMap(c.Do("HGETALL", key))
	return mi
}

func (rcm *ReportManager) getTaskAdd() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskAdd)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getTaskComment() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskComment)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getTaskCompleted() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskCompleted)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getTaskAddPerAccount() MI {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskAddPerAccount)
	mi, _ := redis.IntMap(c.Do("HGETALL", key))
	return mi

}

func (rcm *ReportManager) getTaskAssignedPerAccount() MI {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskAssignedPerAccount)
	mi, _ := redis.IntMap(c.Do("HGETALL", key))
	return mi
}

func (rcm *ReportManager) getTaskCommentPerAccount() MI {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskCommentPerAccount)
	mi, _ := redis.IntMap(c.Do("HGETALL", key))
	return mi
}

func (rcm *ReportManager) getTaskCompletedPerAccount() MI {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterTaskCompletedPerAccount)
	mi, _ := redis.IntMap(c.Do("HGETALL", key))
	return mi
}

func (rcm *ReportManager) getSessionLogin() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterSessionLogin)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getSessionRecall() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterSessionRecall)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getRequests() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterRequests)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getDataIn() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterDataIn)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getDataOut() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterDataOut)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) getProcessTime() int {
	c := _Cache.GetConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", ReportCounterProcessTime)
	n, _ := redis.Int(c.Do("GET", key))
	return n
}

func (rcm *ReportManager) GetCounters() tools.M {
	m := tools.M{
		ReportCounterPostAdd:           rcm.getPostAdd(),
		ReportCounterPostExternalAdd:   rcm.getPostExternalAdd(),
		ReportCounterPostAttachSize:    rcm.getPostAttachSize(),
		ReportCounterPostAttachCount:   rcm.getPostAttachCount(),
		ReportCounterCommentAdd:        rcm.getCommentAdd(),
		ReportCounterTaskAdd:           rcm.getTaskAdd(),
		ReportCounterTaskCompleted:     rcm.getTaskCompleted(),
		ReportCounterTaskComment:       rcm.getTaskComment(),
		ReportCounterSessionLogin:      rcm.getSessionLogin(),
		ReportCounterSessionRecall:     rcm.getSessionRecall(),
		ReportCounterRequests:          rcm.getRequests(),
		ReportCounterDataIn:            rcm.getDataIn(),
		ReportCounterDataOut:           rcm.getDataOut(),
		ReportCounterProcessTime:       rcm.getProcessTime(),
		ReportCounterPostPerAccount:    rcm.getPostPerAccount(),
		ReportCounterPostPerPlace:      rcm.getPostPerPlace(),
		ReportCounterCommentPerAccount: rcm.getCommentPerAccount(),
		ReportCounterCommentPerPlace:   rcm.getCommentPerPlace(),
	}
	return m
}

func (rcm *ReportManager) resetAllCounters() {
	c := _Cache.GetConn()
	defer c.Close()

	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterPostAdd), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterPostExternalAdd), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterPostAttachSize), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterPostAttachCount), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterCommentAdd), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterSessionLogin), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterSessionRecall), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterRequests), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterDataIn), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterDataOut), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterProcessTime), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterTaskAdd), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterTaskCompleted), 0)
	c.Send("SET", fmt.Sprintf("report:counter:%s", ReportCounterTaskComment), 0)
	c.Send("DEL", "report:counter:api")
	c.Send("DEL", fmt.Sprintf("report:counter:%s", ReportCounterPostPerAccount))
	c.Send("DEL", fmt.Sprintf("report:counter:%s", ReportCounterPostPerPlace))
	c.Send("DEL", fmt.Sprintf("report:counter:%s", ReportCounterCommentPerAccount))
	c.Send("DEL", fmt.Sprintf("report:counter:%s", ReportCounterCommentPerPlace))
	c.Send("DEL", fmt.Sprintf("report:counter:%s", ReportCounterTaskAddPerAccount))
	c.Send("DEL", fmt.Sprintf("report:counter:%s", ReportCounterTaskCommentPerAccount))
	c.Send("DEL", fmt.Sprintf("report:counter:%s", ReportCounterTaskCompletedPerAccount))
	c.Send("DEL", fmt.Sprintf("report:counter:%s", ReportCounterTaskAssignedPerAccount))

	if err := c.Flush(); err != nil {
		log.Warn(err.Error())
	}

}

func (rcm *ReportManager) FlushToDB() {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	t := time.Now()
	valKey := fmt.Sprintf("values.%2d", t.Minute())
	bulk := db.C(global.COLLECTION_REPORTS_COUNTERS).Bulk()
	bulk.Unordered()

	// Count General Counters
	m := MI{
		ReportCounterPostAdd:         rcm.getPostAdd(),
		ReportCounterPostExternalAdd: rcm.getPostExternalAdd(),
		ReportCounterPostAttachSize:  rcm.getPostAttachSize(),
		ReportCounterPostAttachCount: rcm.getPostAttachCount(),
		ReportCounterCommentAdd:      rcm.getCommentAdd(),
		ReportCounterTaskAdd:         rcm.getTaskAdd(),
		ReportCounterTaskComment:     rcm.getTaskComment(),
		ReportCounterTaskCompleted:   rcm.getTaskCompleted(),
		ReportCounterSessionLogin:    rcm.getSessionLogin(),
		ReportCounterSessionRecall:   rcm.getSessionRecall(),
		ReportCounterRequests:        rcm.getRequests(),
		ReportCounterDataIn:          rcm.getDataIn(),
		ReportCounterDataOut:         rcm.getDataOut(),
		ReportCounterProcessTime:     rcm.getProcessTime(),
	}
	for k, v := range m {
		bulk.Upsert(
			bson.M{"date": t.Local().Format("2006-01-02:15"), "key": k},
			bson.M{"$inc": bson.M{"sum": v, valKey: v}},
		)
	}

	// Count API Commands
	q := bson.M{}
	for k, v := range rcm.getAPI() {
		q[k], _ = strconv.Atoi(v)
	}
	bulk.Upsert(bson.M{"_id": "apiCommands"}, bson.M{"$inc": q})

	postsPerPlace := rcm.getPostPerPlace()
	for k, v := range postsPerPlace {
		bulk.Upsert(
			bson.M{"date": t.Local().Format("2006-01-02:15"), "key": fmt.Sprintf("place_%s_posts", k)},
			bson.M{"$inc": bson.M{"sum": v, valKey: v}},
		)
	}

	postsPerAccount := rcm.getPostPerAccount()
	for k, v := range postsPerAccount {
		bulk.Upsert(
			bson.M{"date": t.Local().Format("2006-01-02:15"), "key": fmt.Sprintf("account_%s_posts", k)},
			bson.M{"$inc": bson.M{"sum": v, valKey: v}},
		)
	}

	commentsPerAccount := rcm.getCommentPerAccount()
	for k, v := range commentsPerAccount {
		bulk.Upsert(
			bson.M{"date": t.Local().Format("2006-01-02:15"), "key": fmt.Sprintf("account_%s_comments", k)},
			bson.M{"$inc": bson.M{"sum": v, valKey: v}},
		)
	}

	commentsPerPlace := rcm.getCommentPerPlace()
	for k, v := range commentsPerPlace {
		bulk.Upsert(
			bson.M{"date": t.Local().Format("2006-01-02:15"), "key": fmt.Sprintf("place_%s_comments", k)},
			bson.M{"$inc": bson.M{"sum": v, valKey: v}},
		)
	}

	taskAddPerAccount := rcm.getTaskAddPerAccount()
	for k, v := range taskAddPerAccount {
		bulk.Upsert(
			bson.M{"date": t.Local().Format("2006-01-02:15"), "key": fmt.Sprintf("account_%s_task_add", k)},
			bson.M{"$inc": bson.M{"sum": v, valKey: v}},
		)
	}

	taskAssignedPerAccount := rcm.getTaskAssignedPerAccount()
	for k, v := range taskAssignedPerAccount {
		bulk.Upsert(
			bson.M{"date": t.Local().Format("2006-01-02:15"), "key": fmt.Sprintf("account_%s_task_assigned", k)},
			bson.M{"$inc": bson.M{"sum": v, valKey: v}},
		)
	}

	taskCompletedPerAccount := rcm.getTaskCompletedPerAccount()
	for k, v := range taskCompletedPerAccount {
		bulk.Upsert(
			bson.M{"date": t.Local().Format("2006-01-02:15"), "key": fmt.Sprintf("account_%s_task_completed", k)},
			bson.M{"$inc": bson.M{"sum": v, valKey: v}},
		)
	}

	taskCommentPerAccount := rcm.getTaskCommentPerAccount()
	for k, v := range taskCommentPerAccount {
		bulk.Upsert(
			bson.M{"date": t.Local().Format("2006-01-02:15"), "key": fmt.Sprintf("account_%s_task_comment", k)},
			bson.M{"$inc": bson.M{"sum": v, valKey: v}},
		)
	}

	bulk.Run()

	rcm.resetAllCounters()
}

func (rcm *ReportManager) GetTimeSeriesSingleValue(from, to, key, resolution string) []TimeSeriesSingleValueHourly {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	x := make([]TimeSeriesSingleValueHourly, 0, global.DefaultMaxResultLimit)
	switch resolution {
	case ReportResolutionHour:
		if err := db.C(global.COLLECTION_REPORTS_COUNTERS).Find(bson.M{
			"date": bson.M{"$gte": from, "$lte": to},
			"key":  key,
		}).Limit(global.DefaultMaxResultLimit).All(&x); err != nil {
			log.Warn("Model::ReportManager::GetTimeSeriesSingleValue::Error 1::", zap.Error(err))
		}
	case ReportResolutionDay:
		if err := db.C(global.COLLECTION_REPORTS_COUNTERS).Pipe([]bson.M{
			{"$match": bson.M{
				"date": bson.M{"$gte": from, "$lte": to},
				"key":  key,
			}},
			{"$project": bson.M{
				"date": bson.M{"$substr": []interface{}{"$date", 0, 10}},
				"sum":  "$sum",
			}},
			{"$group": bson.M{
				"_id":  "$date",
				"date": bson.M{"$first": "$date"},
				"sum":  bson.M{"$sum": "$sum"},
			}},
		}).All(&x); err != nil {
			log.Error("Model::ReportManager::GetTimeSeriesSingleValue::Error 2::", zap.Error(err))
		}

	}
	return x
}

func (rcm *ReportManager) GetAPICounters() MI {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	m := MI{}
	if err := db.C(global.COLLECTION_REPORTS_COUNTERS).FindId("apiCommands").One(&m); err != nil {
		log.Warn(err.Error())
		return nil
	}
	return m
}
