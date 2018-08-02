package nested

import (
    "fmt"
    "github.com/globalsign/mgo/bson"
    "log"
    "strconv"
    "time"
    "github.com/gomodule/redigo/redis"
)

const (
    REPORT_COUNTER_POST_ADD                   = "post_add"
    REPORT_COUNTER_POST_EXTERNAL_ADD          = "post_ext_add"
    REPORT_COUNTER_POST_ATTACH_SIZE           = "post_attach_size"
    REPORT_COUNTER_POST_ATTACH_COUNT          = "post_attach_count"
    REPORT_COUNTER_POST_PER_ACCOUNT           = "post_per_account"
    REPORT_COUNTER_POST_PER_PLACE             = "post_per_place"
    REPORT_COUNTER_COMMENT_ADD                = "comment_add"
    REPORT_COUNTER_COMMENT_PER_ACCOUNT        = "comment_per_account"
    REPORT_COUNTER_COMMENT_PER_PLACE          = "comment_per_place"
    REPORT_COUNTER_TASK_ADD                   = "task_add"
    REPORT_COUNTER_TASK_COMMENT               = "task_comment"
    REPORT_COUNTER_TASK_COMPLETED             = "task_completed"
    REPORT_COUNTER_TASK_ADD_PER_ACCOUNT       = "task_add_per_account"
    REPORT_COUNTER_TASK_COMPLETED_PER_ACCOUNT = "task_completed_per_account"
    REPORT_COUNTER_TASK_ASSIGNED_PER_ACCOUNT  = "task_assigned_per_account"
    REPORT_COUNTER_TASK_COMMENT_PER_ACCOUNT   = "task_comment_per_account"
    REPORT_COUNTER_SESSION_LOGIN              = "session_login"
    REPORT_COUNTER_SESSION_RECALL             = "session_recall"
    REPORT_COUNTER_REQUESTS                   = "requests"
    REPORT_COUNTER_DATA_IN                    = "data_in"
    REPORT_COUNTER_DATA_OUT                   = "data_out"
    REPORT_COUNTER_PROCESS_TIME               = "process_time"
)
const (
    REPORT_RESOLUTION_HOUR  string = "h"
    REPORT_RESOLUTION_DAY   string = "d"
    REPORT_RESOLUTION_MONTH string = "m"
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
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:api")
    c.Do("HINCRBY", key, cmd, 1)
}
func (rcm *ReportManager) CountPostAdd() {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_ADD)
    c.Do("INCR", key)
}
func (rcm *ReportManager) CountPostExternalAdd() {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_EXTERNAL_ADD)
    c.Do("INCR", key)
}
func (rcm *ReportManager) CountPostAttachSize(n int64) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_ATTACH_SIZE)
    c.Do("INCRBY", key, n)
}
func (rcm *ReportManager) CountPostAttachCount(n int) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_ATTACH_COUNT)
    c.Do("INCRBY", key, n)
}
func (rcm *ReportManager) CountPostPerAccount(accountID string) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_PER_ACCOUNT)
    c.Do("HINCRBY", key, accountID, 1)
}
func (rcm *ReportManager) CountPostPerPlace(placeIDs []string) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_PER_PLACE)
    for _, placeID := range placeIDs {
        c.Send("HINCRBY", key, placeID, 1)
    }
    c.Flush()
}
func (rcm *ReportManager) CountCommentAdd() {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_COMMENT_ADD)
    c.Do("INCR", key)
}
func (rcm *ReportManager) CountCommentPerAccount(accountID string) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_COMMENT_PER_ACCOUNT)
    c.Do("HINCRBY", key, accountID, 1)
}
func (rcm *ReportManager) CountCommentPerPlace(placeIDs []string) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_COMMENT_PER_PLACE)
    for _, placeID := range placeIDs {
        c.Send("HINCRBY", key, placeID, 1)
    }
    c.Flush()
}
func (rcm *ReportManager) CountTaskAdd() {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_ADD)
    c.Do("INCR", key)
}
func (rcm *ReportManager) CountTaskComment() {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMMENT)
    c.Do("INCR", key)
}
func (rcm *ReportManager) CountTaskCompleted() {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMPLETED)
    c.Do("INCR", key)
}
func (rcm *ReportManager) CountTaskAddPerAccount(accountID string) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_ADD_PER_ACCOUNT)
    c.Do("HINCRBY", key, accountID, 1)
}
func (rcm *ReportManager) CountTaskAssignedPerAccount(accountID string) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_ASSIGNED_PER_ACCOUNT)
    c.Do("HINCRBY", key, accountID, 1)
}
func (rcm *ReportManager) CountTaskCommentPerAccount(accountID string) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMMENT_PER_ACCOUNT)
    c.Do("HINCRBY", key, accountID, 1)
}
func (rcm *ReportManager) CountTaskCompletedPerAccount(accountID string) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMPLETED_PER_ACCOUNT)
    c.Do("HINCRBY", key, accountID, 1)
}
func (rcm *ReportManager) CountSessionLogin() {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_SESSION_LOGIN)
    c.Do("INCR", key)
}
func (rcm *ReportManager) CountSessionRecall() {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_SESSION_RECALL)
    c.Do("INCR", key)
}
func (rcm *ReportManager) CountRequests() {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_REQUESTS)
    c.Do("INCR", key)
}
func (rcm *ReportManager) CountDataIn(n int) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_DATA_IN)
    c.Do("INCRBY", key, n)
}
func (rcm *ReportManager) CountDataOut(n int) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_DATA_OUT)
    c.Do("INCRBY", key, n)
}
func (rcm *ReportManager) CountProcessTime(n int) {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_PROCESS_TIME)
    c.Do("INCRBY", key, n)
}
func (rcm *ReportManager) getAPI() MS {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:api")
    ms, _ := redis.StringMap(c.Do("HGETALL", key))
    return ms
}
func (rcm *ReportManager) getPostAdd() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_ADD)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getPostExternalAdd() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_EXTERNAL_ADD)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getPostAttachSize() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_ATTACH_SIZE)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getPostAttachCount() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_ATTACH_COUNT)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getPostPerAccount() MI {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_PER_ACCOUNT)
    mi, _ := redis.IntMap(c.Do("HGETALL", key))
    return mi
}
func (rcm *ReportManager) getPostPerPlace() MI {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_PER_PLACE)
    mi, _ := redis.IntMap(c.Do("HGETALL", key))
    return mi
}
func (rcm *ReportManager) getCommentAdd() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_COMMENT_ADD)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getCommentPerAccount() MI {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_COMMENT_PER_ACCOUNT)
    mi, _ := redis.IntMap(c.Do("HGETALL", key))
    return mi
}
func (rcm *ReportManager) getCommentPerPlace() MI {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_COMMENT_PER_PLACE)
    mi, _ := redis.IntMap(c.Do("HGETALL", key))
    return mi
}
func (rcm *ReportManager) getTaskAdd() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_ADD)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getTaskComment() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMMENT)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getTaskCompleted() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMPLETED)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getTaskAddPerAccount() MI {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_ADD_PER_ACCOUNT)
    mi, _ := redis.IntMap(c.Do("HGETALL", key))
    return mi

}
func (rcm *ReportManager) getTaskAssignedPerAccount() MI {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_ASSIGNED_PER_ACCOUNT)
    mi, _ := redis.IntMap(c.Do("HGETALL", key))
    return mi
}
func (rcm *ReportManager) getTaskCommentPerAccount() MI {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMMENT_PER_ACCOUNT)
    mi, _ := redis.IntMap(c.Do("HGETALL", key))
    return mi
}
func (rcm *ReportManager) getTaskCompletedPerAccount() MI {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMPLETED_PER_ACCOUNT)
    mi, _ := redis.IntMap(c.Do("HGETALL", key))
    return mi
}
func (rcm *ReportManager) getSessionLogin() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_SESSION_LOGIN)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getSessionRecall() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_SESSION_RECALL)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getRequests() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_REQUESTS)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getDataIn() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_DATA_IN)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getDataOut() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_DATA_OUT)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) getProcessTime() int {
    c := _Cache.getConn()
    defer c.Close()
    key := fmt.Sprintf("report:counter:%s", REPORT_COUNTER_PROCESS_TIME)
    n, _ := redis.Int(c.Do("GET", key))
    return n
}
func (rcm *ReportManager) GetCounters() M {
    m := M{
        REPORT_COUNTER_POST_ADD:            rcm.getPostAdd(),
        REPORT_COUNTER_POST_EXTERNAL_ADD:   rcm.getPostExternalAdd(),
        REPORT_COUNTER_POST_ATTACH_SIZE:    rcm.getPostAttachSize(),
        REPORT_COUNTER_POST_ATTACH_COUNT:   rcm.getPostAttachCount(),
        REPORT_COUNTER_COMMENT_ADD:         rcm.getCommentAdd(),
        REPORT_COUNTER_TASK_ADD:            rcm.getTaskAdd(),
        REPORT_COUNTER_TASK_COMPLETED:      rcm.getTaskCompleted(),
        REPORT_COUNTER_TASK_COMMENT:        rcm.getTaskComment(),
        REPORT_COUNTER_SESSION_LOGIN:       rcm.getSessionLogin(),
        REPORT_COUNTER_SESSION_RECALL:      rcm.getSessionRecall(),
        REPORT_COUNTER_REQUESTS:            rcm.getRequests(),
        REPORT_COUNTER_DATA_IN:             rcm.getDataIn(),
        REPORT_COUNTER_DATA_OUT:            rcm.getDataOut(),
        REPORT_COUNTER_PROCESS_TIME:        rcm.getProcessTime(),
        REPORT_COUNTER_POST_PER_ACCOUNT:    rcm.getPostPerAccount(),
        REPORT_COUNTER_POST_PER_PLACE:      rcm.getPostPerPlace(),
        REPORT_COUNTER_COMMENT_PER_ACCOUNT: rcm.getCommentPerAccount(),
        REPORT_COUNTER_COMMENT_PER_PLACE:   rcm.getCommentPerPlace(),
    }
    return m
}
func (rcm *ReportManager) resetAllCounters() {
    _funcName := "ReportManager::resetAllCounters"
    c := _Cache.getConn()
    defer c.Close()

    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_ADD), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_EXTERNAL_ADD), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_ATTACH_SIZE), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_ATTACH_COUNT), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_COMMENT_ADD), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_SESSION_LOGIN), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_SESSION_RECALL), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_REQUESTS), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_DATA_IN), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_DATA_OUT), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_PROCESS_TIME), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_ADD), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMPLETED), 0)
    c.Send("SET", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMMENT), 0)
    c.Send("DEL", "report:counter:api")
    c.Send("DEL", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_PER_ACCOUNT))
    c.Send("DEL", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_POST_PER_PLACE))
    c.Send("DEL", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_COMMENT_PER_ACCOUNT))
    c.Send("DEL", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_COMMENT_PER_PLACE))
    c.Send("DEL", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_ADD_PER_ACCOUNT))
    c.Send("DEL", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMMENT_PER_ACCOUNT))
    c.Send("DEL", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_COMPLETED_PER_ACCOUNT))
    c.Send("DEL", fmt.Sprintf("report:counter:%s", REPORT_COUNTER_TASK_ASSIGNED_PER_ACCOUNT))

    if err := c.Flush(); err != nil {
        _Log.Error(_funcName, err.Error())
    }

}
func (rcm *ReportManager) FlushToDB() {
    _funcName := "ReportManager::FlushToDB"
    _Log.FunctionStarted(_funcName)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()


    t := time.Now()
    valKey := fmt.Sprintf("values.%2d", t.Minute())
    bulk := db.C(COLLECTION_REPORTS_COUNTERS).Bulk()
    bulk.Unordered()

    // Count General Counters
    m := MI{
        REPORT_COUNTER_POST_ADD:          rcm.getPostAdd(),
        REPORT_COUNTER_POST_EXTERNAL_ADD: rcm.getPostExternalAdd(),
        REPORT_COUNTER_POST_ATTACH_SIZE:  rcm.getPostAttachSize(),
        REPORT_COUNTER_POST_ATTACH_COUNT: rcm.getPostAttachCount(),
        REPORT_COUNTER_COMMENT_ADD:       rcm.getCommentAdd(),
        REPORT_COUNTER_TASK_ADD:          rcm.getTaskAdd(),
        REPORT_COUNTER_TASK_COMMENT:      rcm.getTaskComment(),
        REPORT_COUNTER_TASK_COMPLETED:    rcm.getTaskCompleted(),
        REPORT_COUNTER_SESSION_LOGIN:     rcm.getSessionLogin(),
        REPORT_COUNTER_SESSION_RECALL:    rcm.getSessionRecall(),
        REPORT_COUNTER_REQUESTS:          rcm.getRequests(),
        REPORT_COUNTER_DATA_IN:           rcm.getDataIn(),
        REPORT_COUNTER_DATA_OUT:          rcm.getDataOut(),
        REPORT_COUNTER_PROCESS_TIME:      rcm.getProcessTime(),
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
    _funcName := "ReportManager::GetTimeSeriesSingleValue"
    _Log.FunctionStarted(_funcName, from, to, key, resolution)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    x := make([]TimeSeriesSingleValueHourly, 0, DEFAULT_MAX_RESULT_LIMIT)
    switch resolution {
    case REPORT_RESOLUTION_HOUR:
        if err := db.C(COLLECTION_REPORTS_COUNTERS).Find(bson.M{
            "date": bson.M{"$gte": from, "$lte": to},
            "key":  key,
        }).Limit(DEFAULT_MAX_RESULT_LIMIT).All(&x); err != nil {
            log.Println("Model::ReportManager::GetTimeSeriesSingleValue::Error 1::", err.Error())
        }
    case REPORT_RESOLUTION_DAY:
        if err := db.C(COLLECTION_REPORTS_COUNTERS).Pipe([]bson.M{
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
            log.Println("Model::ReportManager::GetTimeSeriesSingleValue::Error 2::", err.Error())
        }

    }
    return x
}

func (rcm *ReportManager) GetAPICounters() MI {
    _funcName := "ReportManager::GetAPICounters"
    _Log.FunctionStarted(_funcName)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()


    m := MI{}
    if err := db.C(COLLECTION_REPORTS_COUNTERS).FindId("apiCommands").One(&m); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }
    return m
}
