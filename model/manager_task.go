package nested

import (
    "bytes"
    "encoding/gob"
    "fmt"

    "github.com/globalsign/mgo/bson"
    "github.com/gomodule/redigo/redis"
)

const (
    TASK_STATUS_NOT_ASSIGNED TaskStatus = 0x01
    TASK_STATUS_ASSIGNED     TaskStatus = 0x02
    TASK_STATUS_CANCELED     TaskStatus = 0x03
    TASK_STATUS_REJECTED     TaskStatus = 0x04
    TASK_STATUS_COMPLETED    TaskStatus = 0x05
    TASK_STATUS_HOLD         TaskStatus = 0x06
    TASK_STATUS_OVERDUE      TaskStatus = 0x07
    TASK_STATUS_FAILED       TaskStatus = 0x08
)

const (
    TASK_ACCESS_PICK_TASK         = 0x0001
    TASK_ACCESS_ADD_CANDIDATE     = 0x0002
    TASK_ACCESS_READ              = 0x0003
    TASK_ACCESS_CHANGE_ASSIGNEE   = 0x0008
    TASK_ACCESS_CHANGE_PRIORITY   = 0x0020
    TASK_ACCESS_COMMENT           = 0x0080
    TASK_ACCESS_ADD_ATTACHMENT    = 0x00F0
    TASK_ACCESS_REMOVE_ATTACHMENT = 0x0100
    TASK_ACCESS_ADD_WATCHER       = 0x0200
    TASK_ACCESS_REMOVE_WATCHER    = 0x0400
    TASK_ACCESS_DELETE            = 0x0800
    TASK_ACCESS_ADD_LABEL         = 0x0101
    TASK_ACCESS_REMOVE_LABEL      = 0x0102
    TASK_ACCESS_UPDATE            = 0x0103
    TASK_ACCESS_ADD_EDITOR        = 0x0108
    TASK_ACCESS_REMOVE_EDITOR     = 0x0109
)

type TaskStatus int
type TaskAccess map[int]bool

// Task Manager and Methods
type TaskManager struct{}

func NewTaskManager() *TaskManager {
    return new(TaskManager)
}

func (tm *TaskManager) readFromCache(taskID bson.ObjectId) *Task {
    task := new(Task)
    c := _Cache.Pool.Get()
    defer c.Close()
    keyID := fmt.Sprintf("task:gob:%s", taskID.Hex())
    if gobTask, err := redis.Bytes(c.Do("GET", keyID)); err != nil {
        if err := _MongoDB.C(COLLECTION_TASKS).Find(
            bson.M{"_id": taskID, "_removed": false},
        ).One(task); err != nil {
            _Log.Warn(err.Error())
            return nil
        }
        gobTask := new(bytes.Buffer)
        if err := gob.NewEncoder(gobTask).Encode(task); err == nil {
            c.Do("SETEX", keyID, CACHE_LIFETIME, gobTask.Bytes())
        }
        return task
    } else if err := gob.NewDecoder(bytes.NewBuffer(gobTask)).Decode(task); err == nil {
        return task
    }
    return nil
}

func (tm *TaskManager) readMultiFromCache(taskIDs []bson.ObjectId) []Task {
    tasks := make([]Task, 0, len(taskIDs))
    c := _Cache.Pool.Get()
    defer c.Close()
    for _, taskID := range taskIDs {
        keyID := fmt.Sprintf("task:gob:%s", taskID.Hex())
        c.Send("GET", keyID)
    }
    c.Flush()
    for _, taskID := range taskIDs {
        if gobPlace, err := redis.Bytes(c.Receive()); err == nil {
            task := new(Task)
            if err := gob.NewDecoder(bytes.NewBuffer(gobPlace)).Decode(task); err == nil {
                tasks = append(tasks, *task)
            }
        } else {
            if task := _Manager.Task.readFromCache(taskID); task != nil {
                tasks = append(tasks, *task)
            }
        }
    }
    return tasks
}

func (tm *TaskManager) removeCache(taskID bson.ObjectId) bool {
    c := _Cache.Pool.Get()
    defer c.Close()
    keyID := fmt.Sprintf("task:gob:%s", taskID.Hex())
    c.Do("DEL", keyID)
    return true
}

// CreateTask creates the task object in database based on the date of TaskCreateRequest
func (tm *TaskManager) CreateTask(tcr TaskCreateRequest) *Task {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ts := Timestamp()
    task := new(Task)
    task.ID = bson.NewObjectId()
    task.Timestamp, task.LastUpdate = ts, ts
    task.Title = tcr.Title
    task.Description = tcr.Description
    task.AssigneeID = tcr.AssigneeID
    task.AssignorID = tcr.AssignorID
    task.CandidateIDs = tcr.CandidateIDs
    task.WatcherIDs = tcr.WatcherIDs
    task.EditorIDs = tcr.EditorIDs
    task.LabelIDs = tcr.LabelIDs
    task.ToDos = tcr.Todos
    task.AttachmentIDs = tcr.AttachmentIDs
    task.RelatedTo = tcr.RelatedTo
    task.RelatedPost = tcr.RelatedPost

    // Set Task Members
    task.MemberIDs = append(task.MemberIDs, task.WatcherIDs...)
    task.MemberIDs = append(task.MemberIDs, task.EditorIDs ...)
    task.MemberIDs = append(task.MemberIDs, task.CandidateIDs ...)
    task.MemberIDs = append(task.MemberIDs, task.AssigneeID, task.AssignorID)

    // Set counters
    task.Counters.Labels = len(tcr.LabelIDs)
    task.Counters.ToDoNextID = len(tcr.Todos) + 1
    task.Counters.Candidates = len(tcr.CandidateIDs)
    task.Counters.Attachments = len(tcr.AttachmentIDs)
    task.Counters.Watchers = len(tcr.WatcherIDs)

    // Set due date
    task.DueDate = tcr.DueDate
    task.DueDateHasClock = tcr.DueDateHasClock

    if len(task.AssigneeID) > 0 {
        task.Status = TASK_STATUS_ASSIGNED
    } else {
        task.Status = TASK_STATUS_NOT_ASSIGNED
    }

    // Create the task object in db
    if err := db.C(COLLECTION_TASKS).Insert(task); err != nil {
        _Log.Warn(err.Error())
        return nil
    }

    // Add the taskID to the bucket if due date has been set
    if task.DueDate > 0 {
        _Manager.TimeBucket.AddOverdueTask(task.DueDate, task.ID)
    }

    // Update post document by adding the taskID to the post's related tasks set
    if err := db.C(COLLECTION_POSTS).UpdateId(
        task.RelatedPost,
        bson.M{"$addToSet": bson.M{"related_tasks": task.ID}},
    ); err != nil {
        _Log.Warn(err.Error())
    }

    // Set task as the owner of the attachments
    for _, fileID := range task.AttachmentIDs {
        _Manager.File.AddTaskAsOwner(fileID, task.ID)
    }

    for _, labelID := range task.LabelIDs {
        _Manager.Label.IncrementCounter(labelID, "tasks", 1)
    }

    // Update the parent task if exists
    defer _Manager.Task.removeCache(task.RelatedTo)
    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":           task.RelatedTo,
            "related_tasks": bson.M{"$ne": task.ID},
        },
        bson.M{
            "$addToSet": bson.M{"related_tasks": task.ID},
            "$inc":      bson.M{"counters.related_tasks": 1},
        },
    ); err != nil {
        _Log.Warn(err.Error())
    }

    _Manager.TaskActivity.Created(task.ID, task.AssignorID)

    // Increment Report Counters
    _Manager.Report.CountTaskAdd()
    _Manager.Report.CountTaskAddPerAccount(task.AssignorID)
    if len(task.AssigneeID) > 0 {
        _Manager.Report.CountTaskAssignedPerAccount(task.AssigneeID)
    }

    return task
}

// GetByID returns a pointer to Task identified by taskID
func (tm *TaskManager) GetByID(taskID bson.ObjectId) *Task {
    return tm.readFromCache(taskID)
}

// GetTasksByIDs returns an array of Tasks identified by taskIDs
func (tm *TaskManager) GetTasksByIDs(taskIDs []bson.ObjectId) []Task {
    return tm.readMultiFromCache(taskIDs)
}

// GetByAssigneeID returns an array of tasks filtered by Assignee of the task
func (tm *TaskManager) GetByAssigneeID(accountID string, pg Pagination, filter []TaskStatus) []Task {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    tasks := make([]Task, 0, pg.GetLimit())
    q := bson.M{
        "members":  accountID,
        "assignee": accountID,
        "_removed": false,
    }
    if len(filter) > 0 {
        q["status"] = bson.M{"$in": filter}
    }
    if err := db.C(COLLECTION_TASKS).Find(q).Sort("-_id").Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&tasks); err != nil {
        _Log.Warn(err.Error())
    }
    return tasks
}

// GetByAssignorID returns an array of tasks filtered by Assignor of the task
func (tm *TaskManager) GetByAssignorID(accountID string, pg Pagination, filter []TaskStatus) []Task {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    tasks := make([]Task, 0, pg.GetLimit())
    q := bson.M{
        "members":  accountID,
        "assignor": accountID,
        "_removed": false,
    }
    if len(filter) > 0 {
        q["status"] = bson.M{"$in": filter}
    }
    if err := db.C(COLLECTION_TASKS).Find(q).Sort("-_id").Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&tasks); err != nil {
        _Log.Warn(err.Error())
    }
    return tasks
}

// GetByWatcherID returns an array of tasks filtered by Watcher of the task
func (tm *TaskManager) GetByWatcherEditorID(accountID string, pg Pagination, filter []TaskStatus) []Task {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    tasks := make([]Task, 0, pg.GetLimit())
    q := bson.M{
        "members": accountID,
        "$or": []bson.M{
            {"watchers": accountID},
            {"editors": accountID},
        },
        "_removed": false,
    }
    if len(filter) > 0 {
        q["status"] = bson.M{"$in": filter}
    }
    if err := db.C(COLLECTION_TASKS).Find(q).Sort("-_id").Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&tasks); err != nil {
        _Log.Warn(err.Error())
    }
    return tasks
}

// GetByCandidateID returns an array of tasks filtered by Candidate of the task
func (tm *TaskManager) GetByCandidateID(accountID string, pg Pagination, filter []TaskStatus) []Task {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    tasks := make([]Task, 0, pg.GetLimit())
    q := bson.M{
        "members":    accountID,
        "candidates": accountID,
        "_removed":   false,
    }
    if len(filter) > 0 {
        q["status"] = bson.M{"$in": filter}
    }
    if err := db.C(COLLECTION_TASKS).Find(q).Sort("-_id").Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&tasks); err != nil {
        _Log.Warn(err.Error())
    }
    return tasks
}

// GetByCustomerFilter returns an array of tasks filtered by factors such as
// 1. Assignee
// 2. Assignor
// 3. Labels
func (tm *TaskManager) GetByCustomFilter(
    accountID string, assignorIDs, assigneeIDs, labelIDs []string, labelLogic, keyword string,
    pg Pagination, filter []TaskStatus, dueDate uint64, createdAt uint64,
) []Task {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    tasks := make([]Task, 0, pg.GetLimit())
    q := bson.M{
        "members":  accountID,
        "_removed": false,
    }
    if len(assignorIDs) > 0 {
        q["assignor"] = bson.M{"$in": assignorIDs}
    }
    if len(assigneeIDs) > 0 {
        q["assignee"] = bson.M{"$in": assigneeIDs}
    }
    if len(keyword) > 0 {
        q["$text"] = bson.M{
            "$search":             keyword,
            "$caseSensitive":      false,
            "$diacriticSensitive": false,
        }
    }
	if createdAt > 0 {
		q["timestamp"] = bson.M{"$gte": createdAt}
	}
    if dueDate > 0 {
        q["due_date"] = bson.M{"$lt": dueDate}
    }
    switch len(labelIDs) {
    case 0:
    case 1:
        q["labels"] = labelIDs[0]
    default:
        if labelLogic == "and" {
            q["labels"] = bson.M{"$all": labelIDs}
        } else {
            q["labels"] = bson.M{"$in": labelIDs}
        }
    }

    if len(filter) > 0 {
        q["status"] = bson.M{"$in": filter}
    }

    task := new(Task)
    iter := db.C(COLLECTION_TASKS).Find(q).Sort("-_id").Skip(pg.GetSkip()).Iter()
    defer iter.Close()
    for iter.Next(task) {
        if task.HasAccess(accountID, TASK_ACCESS_READ) {
            tasks = append(tasks, *task)
        }
        if len(tasks) == cap(tasks) {
            break
        }
    }

    return tasks
}

// GetUpcomingTasks returns an array of tasks assigned to accountID and have their due date set
func (tm *TaskManager) GetUpcomingTasks(accountID string, pg Pagination) []Task {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    tasks := make([]Task, 0, pg.GetLimit())
    q := bson.M{
        "assignee": accountID,
        "status":   TASK_STATUS_ASSIGNED,
        "_removed": false,
    }
    if err := db.C(COLLECTION_TASKS).Find(q).Sort("-due_date").Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&tasks); err != nil {
        _Log.Warn(err.Error())
    }
    return tasks
}

// RemoveTask soft-removes the task and all of its activities. This function sets _removed to TRUE
func (tm *TaskManager) RemoveTask(taskID bson.ObjectId) bool {
    // _funcName

    // removed LOG Function
    defer _Manager.Task.removeCache(taskID)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    task := _Manager.Task.GetByID(taskID)

    // Remove the task document
    if err := db.C(COLLECTION_TASKS).UpdateId(
        taskID,
        bson.M{"$set": bson.M{"_removed": true}},
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Remove all related task activities
    if _, err := db.C(COLLECTION_TASKS_ACTIVITIES).UpdateAll(
        bson.M{"task_id": taskID},
        bson.M{"$set": bson.M{"_removed": true}},
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Remove the relation
    if len(task.RelatedTasks) > 0 {
        if _, err := db.C(COLLECTION_TASKS).UpdateAll(
            bson.M{"_id": bson.M{"$in": task.RelatedTasks}},
            bson.M{"$unset": bson.M{"related_to": true}},
        ); err != nil {
            _Log.Warn(err.Error())
        }
    }

    if len(task.RelatedTo.Hex()) > 0 {
        if err := db.C(COLLECTION_TASKS).Update(
            bson.M{"_id": task.RelatedTo, "related_tasks": task.ID},
            bson.M{
                "$pull": bson.M{"related_tasks": task.ID},
                "$inc":  bson.M{"counters.related_tasks": -1},
            },
        ); err != nil {
            _Log.Warn(err.Error())
        }
    }
    return true
}

//  /////
// Task
//  ///////

type Task struct {
    ID              bson.ObjectId   `bson:"_id"`
    AssignorID      string          `bson:"assignor"`
    AssigneeID      string          `bson:"assignee"`
    CandidateIDs    []string        `bson:"candidates"`
    WatcherIDs      []string        `bson:"watchers"`
    EditorIDs       []string        `bson:"editors"`
    MemberIDs       []string        `bson:"members"`
    LabelIDs        []string        `bson:"labels"`
    AttachmentIDs   []UniversalID   `bson:"attachments"`
    ToDos           []TaskToDo      `bson:"todos"`
    Counters        TaskCounters    `bson:"counters"`      // All the counters go into this object
    RelatedTasks    []bson.ObjectId `bson:"related_tasks"` // TaskIDs which have been derived from this task
    RelatedPost     bson.ObjectId   `bson:"related_post,omitempty"`
    RelatedTo       bson.ObjectId   `bson:"related_to,omitempty"` // TaskID which this task has been derived from
    Title           string          `bson:"title"`
    Description     string          `bson:"description"`
    Timestamp       uint64          `bson:"timestamp"`
    CompletedOn     uint64          `bson:"completed_on"`
    DueDate         uint64          `bson:"due_date"`
    DueDateHasClock bool            `bson:"due_date_has_clock"`
    LastUpdate      uint64          `bson:"last_update"`
    Status          TaskStatus      `bson:"status"`
    Removed         bool            `bson:"_removed"`
}
type TaskCounters struct {
    Comments     int `bson:"comments" json:"comments"`
    Attachments  int `bson:"attachments" json:"attachments"`
    Watchers     int `bson:"watchers" json:"watchers"`
    Labels       int `bson:"labels" json:"labels"`
    Candidates   int `bson:"candidates" json:"candidated"`
    Editors      int `bson:"editors" json:"editors"`
    Updates      int `bson:"updates" json:"updated"`
    ToDoNextID   int `bson:"todo_nid" json:"-"`
    RelatedTasks int `bson:"related_tasks" json:"related_tasks"`
}
type TaskToDo struct {
    ID     int    `bson:"_id" json:"_id"`
    Text   string `bson:"txt" json:"txt"`
    Weight int    `bson:"weight" json:"weight"`
    Done   bool   `bson:"done" json:"done"`
}
type TaskCreateRequest struct {
    AssignorID      string
    AssigneeID      string
    CandidateIDs    []string
    WatcherIDs      []string
    EditorIDs       []string
    AttachmentIDs   []UniversalID
    LabelIDs        []string
    Title           string
    Description     string
    Todos           []TaskToDo
    RelatedTo       bson.ObjectId
    RelatedPost     bson.ObjectId
    DueDate         uint64
    DueDateHasClock bool
}

// AddComment add a new activity of type comment
func (t *Task) AddComment(accountID, text string) *TaskActivity {
    defer _Manager.Task.removeCache(t.ID)
    // Create task activity

    taskActivity := _Manager.TaskActivity.Comment(t.ID, accountID, text)

    _Manager.Report.CountTaskComment()
    _Manager.Report.CountTaskCommentPerAccount(accountID)

    return taskActivity
}

// RemoveComment removes a comment
func (t *Task) RemoveComment(accountID string, commentID bson.ObjectId) bool {
    defer _Manager.Task.removeCache(t.ID)

    // Remove task activity
    return _Manager.TaskActivity.Remove(commentID)
}

// AddAttachments add fileIDs to the task and create the related task activities
func (t *Task) AddAttachments(accountID string, fileIDs []UniversalID) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":         t.ID,
            "attachments": bson.M{"$nin": fileIDs},
        },
        bson.M{
            "$addToSet": bson.M{"attachments": bson.M{"$each": fileIDs}},
            "$inc":      bson.M{"counters.attachments": len(fileIDs)},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Set task as the owner of the fileIDs
    for _, fileID := range fileIDs {
        _Manager.File.AddTaskAsOwner(fileID, t.ID)
		_Manager.File.SetStatus(fileID, FILE_STATUS_ATTACHED)
    }

    // Create the appropriate activity
    _Manager.TaskActivity.AttachmentAdded(t.ID, accountID, fileIDs)

    return true
}

// AddLabels add labelIDs to the task and create the related task activities
func (t *Task) AddLabels(accountID string, labelIDs []string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":    t.ID,
            "labels": bson.M{"$nin": labelIDs},
        },
        bson.M{
            "$addToSet": bson.M{"labels": bson.M{"$each": labelIDs}},
            "$inc":      bson.M{"counters.labels": len(labelIDs)},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    for _, labelID := range labelIDs {
        _Manager.Label.IncrementCounter(labelID, "tasks", 1)
    }

    _Manager.TaskActivity.LabelAdded(t.ID, accountID, labelIDs)
    return true
}

// AddToDo add a new "ToDoItem" to the task document and updates the todo_nid (next id)
func (t *Task) AddToDo(accountID string, txt string, weight int) *TaskToDo {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    toDo := TaskToDo{
        ID:     t.Counters.ToDoNextID,
        Text:   txt,
        Weight: weight,
    }
    t.ToDos = append(t.ToDos, toDo)
    if err := db.C(COLLECTION_TASKS).UpdateId(
        t.ID,
        bson.M{
            "$set": bson.M{"todos": t.ToDos},
            "$inc": bson.M{"counters.todo_nid": 1},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return nil
    }

    // Create task activity
    _Manager.TaskActivity.ToDoAdded(t.ID, accountID, toDo.Text)

    return &toDo
}

// AddWatchers accept an array of watcherIDs and add them to the list of the taskID, if any of the
// watcherIDs has been already in the list then none of the watcherIDs added.
// Caller must make sure that all the watcherIDs are not in the list before calling this function
func (t *Task) AddWatchers(adderID string, watcherIDs []string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":      t.ID,
            "watchers": bson.M{"$nin": watcherIDs},
        },
        bson.M{
            "$addToSet": bson.M{
                "watchers": bson.M{"$each": watcherIDs},
                "members":  bson.M{"$each": watcherIDs},
            },
            "$inc": bson.M{"counters.watchers": len(watcherIDs)},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Create task activity
    _Manager.TaskActivity.WatcherAdded(t.ID, adderID, watcherIDs)

    return true
}

// AddEditors accept an array of editorIDs and add them to the list of the taskID, if any of the
// editorIDs has been already in the list then none of the editorIDs will be added.
func (t *Task) AddEditors(adderID string, editorIDs []string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":     t.ID,
            "editors": bson.M{"$nin": editorIDs},
        },
        bson.M{
            "$addToSet": bson.M{
                "editors": bson.M{"$each": editorIDs},
                "members": bson.M{"$each": editorIDs},
            },
            "$inc": bson.M{"counters.editors": len(editorIDs)},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Create task activity
    _Manager.TaskActivity.EditorAdded(t.ID, adderID, editorIDs)

    return true
}

// AddCandidate accept and array of candidateIDs and add them to the list of the taskID, if any of the
// candidateIDs has been already in the list then none of the candidateIDs will be added.
// Caller must make sure that all the candidateIDs are not int the list before calling this function
func (t *Task) AddCandidates(adderID string, candidateIDs []string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":        t.ID,
            "candidates": bson.M{"$nin": candidateIDs},
        },
        bson.M{
            "$addToSet": bson.M{
                "candidates": bson.M{"$each": candidateIDs},
                "members":    bson.M{"$each": candidateIDs},
            },
            "$inc": bson.M{"counters.watchers": len(candidateIDs)},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Create task activity
    _Manager.TaskActivity.CandidateAdded(t.ID, adderID, candidateIDs)

    return true
}

// RemoveAttachments removes fileID from the task and creates the appropriate task activity
func (t *Task) RemoveAttachments(accountID string, fileIDs []UniversalID) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":         t.ID,
            "attachments": bson.M{"$all": fileIDs},
        },
        bson.M{
            "$pull": bson.M{"attachments": bson.M{"$in": fileIDs}},
            "$inc":  bson.M{"counters.attachments": -len(fileIDs)},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Set task as the owner of the fileIDs
    for _, fileID := range fileIDs {
        _Manager.File.RemoveTaskAsOwner(fileID, t.ID)
    }

    // Create the appropriate activity
    _Manager.TaskActivity.AttachmentRemoved(t.ID, accountID, fileIDs)

    return true
}

// RemoveLabels removes labelID from the task and creates the appropriate task activity
func (t *Task) RemoveLabels(accountID string, labelIDs []string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":    t.ID,
            "labels": bson.M{"$all": labelIDs},
        },
        bson.M{
            "$pull": bson.M{"labels": bson.M{"$in": labelIDs}},
            "$inc":  bson.M{"counters.labels": -len(labelIDs)},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    for _, labelID := range labelIDs {
        _Manager.Label.IncrementCounter(labelID, "tasks", -1)
    }

    // Create the appropriate activity
    _Manager.TaskActivity.LabelRemoved(t.ID, accountID, labelIDs)

    return true
}

// RemoveToDo removes the "ToDoItem" and creates the related task activity
func (t *Task) RemoveToDo(accountID string, todoID int) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    var todo TaskToDo
    for _, t := range t.ToDos {
        if t.ID == todoID {
            todo = t
            break
        }
    }
    if err := db.C(COLLECTION_TASKS).UpdateId(
        t.ID,
        bson.M{"$pull": bson.M{"todos": bson.M{"_id": todoID}}},
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Create task activity
    _Manager.TaskActivity.ToDoRemoved(t.ID, accountID, todo.Text)
    return true
}

// RemoveEditors removes the editorID from the watchers list of the taskID and returns true if the
// operation was successful otherwise returns false
func (t *Task) RemoveEditors(removerID string, editorIDs []string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    onlyEditorIDs := make([]string, 0)
    for _, editorID := range editorIDs {
        found := false
        for _, watcherID := range t.WatcherIDs {
            if editorID == watcherID {
                found = true
                break
            }
        }
        if !found {
            onlyEditorIDs = append(onlyEditorIDs, editorID)
        }
    }
    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":     t.ID,
            "editors": bson.M{"$all": editorIDs},
        },
        bson.M{
            "$pull": bson.M{
                "editors": bson.M{"$in": editorIDs},
                "members": bson.M{"$in": onlyEditorIDs},
            },
            "$inc": bson.M{"counters.editors": -len(editorIDs)},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Create task activity
    _Manager.TaskActivity.EditorRemoved(t.ID, removerID, editorIDs)

    return true
}

// RemoveWatchers removes the watcherID from the watchers list of the taskID and returns true if the
// operation was successful otherwise returns false
func (t *Task) RemoveWatchers(removerID string, watcherIDs []string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    onlyWatcherIDs := make([]string, 0)
    for _, watcherID := range watcherIDs {
        found := false
        for _, editorID := range t.EditorIDs {
            if watcherID == editorID {
                found = true
                break
            }
        }
        if !found {
            onlyWatcherIDs = append(onlyWatcherIDs, watcherID)
        }
    }
    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":      t.ID,
            "watchers": bson.M{"$all": watcherIDs},
        },
        bson.M{
            "$pull": bson.M{
                "watchers": bson.M{"$in": watcherIDs},
                "members":  bson.M{"$in": onlyWatcherIDs},
            },
            "$inc": bson.M{"counters.watchers": -len(watcherIDs)},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Create task activity
    _Manager.TaskActivity.WatcherRemoved(t.ID, removerID, watcherIDs)

    return true
}

// RemoveCandidates removes the watcherID from the watchers list of the taskID and returns true if the
// operation was successful otherwise returns false
func (t *Task) RemoveCandidates(removerID string, candidateIDs []string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":        t.ID,
            "candidates": bson.M{"$all": candidateIDs},
        },
        bson.M{
            "$pull": bson.M{
                "candidates": bson.M{"$in": candidateIDs},
                "members":    bson.M{"$in": candidateIDs},
            },
            "$inc": bson.M{"counters.candidates": -len(candidateIDs)},
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    // Create task activity
    _Manager.TaskActivity.CandidateRemoved(t.ID, removerID, candidateIDs)

    return true
}

// UpdateMemberIDs
func (t *Task) UpdateMemberIDs() {
    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()
    memberIDs := MB{}
    memberIDs.AddKeys(t.WatcherIDs, t.EditorIDs, t.CandidateIDs, []string{t.AssigneeID, t.AssignorID})
    if err := db.C(COLLECTION_TASKS).UpdateId(
        t.ID,
        bson.M{"$set": bson.M{"members": memberIDs.KeysToArray()}},
    ); err != nil {
        _Log.Warn(err.Error())
    }
    return
}

// UpdateStatus
func (t *Task) UpdateStatus(accountID string, newStatus TaskStatus) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if newStatus == t.Status {
        return false
    }
    switch newStatus {
    case TASK_STATUS_COMPLETED:
        if err := db.C(COLLECTION_TASKS).UpdateId(
            t.ID,
            bson.M{"$set": bson.M{
                "status":       newStatus,
                "completed_on": Timestamp(),
            }},
        ); err != nil {
            _Log.Warn(err.Error())
            return false
        }
        _Manager.Report.CountTaskCompletedPerAccount(t.AssigneeID)
        _Manager.Report.CountTaskCompleted()
    case TASK_STATUS_ASSIGNED, TASK_STATUS_CANCELED, TASK_STATUS_FAILED,
        TASK_STATUS_HOLD, TASK_STATUS_NOT_ASSIGNED, TASK_STATUS_REJECTED, TASK_STATUS_OVERDUE:
        if err := db.C(COLLECTION_TASKS).UpdateId(
            t.ID,
            bson.M{"$set": bson.M{
                "status":       newStatus,
                "completed_on": 0,
            }},
        ); err != nil {
            _Log.Warn(err.Error())
            return false
        }
        _Manager.TaskActivity.StatusChanged(t.ID, accountID, newStatus)
        return true
    }
    return false
}

func (t *Task) UpdateTodo(accountID string, todoID int, text string, weight int, done bool) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    var found bool
    var oldTodo TaskToDo
    for _, v := range t.ToDos {
        if v.ID == todoID {
            oldTodo = v
            found = true
            break
        }
    }
    if !found {
        return false
    }

    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":       t.ID,
            "todos._id": todoID,
        },
        bson.M{"$set": bson.M{
            "todos.$.txt":    text,
            "todos.$.weight": weight,
            "todos.$.done":   done,
        }},
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    if oldTodo.Done != done {
        if done {
            _Manager.TaskActivity.ToDoDone(t.ID, accountID, text)
        } else {
            _Manager.TaskActivity.ToDoUndone(t.ID, accountID, text)
        }
    }

    if oldTodo.Text != text {
        _Manager.TaskActivity.ToDoChanged(t.ID, accountID, oldTodo.Text)
    }

    return true

}

func (t *Task) Update(accountID string, title, desc string, dueDate uint64, dueDateHasClock bool) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TASKS).UpdateId(
        t.ID,
        bson.M{
            "$set": bson.M{
                "title":              title,
                "description":        desc,
                "due_date":           dueDate,
                "due_date_has_clock": dueDateHasClock,
            }},
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    if t.Title != title {
        _Manager.TaskActivity.TaskTitleChanged(t.ID, accountID, title)
    }
    if t.Description != desc {
        _Manager.TaskActivity.TaskDescriptionChanged(t.ID, accountID, desc)
    }
    if t.DueDate != dueDate {
        if dueDate == 0 {
            _Manager.TimeBucket.RemoveOverdueTask(t.DueDate, t.ID)
            _Manager.TaskActivity.DueDateRemoved(t.ID, accountID)
        } else {
            _Manager.TimeBucket.RemoveOverdueTask(t.DueDate, t.ID)
            _Manager.TimeBucket.AddOverdueTask(dueDate, t.ID)
            _Manager.TaskActivity.DueDateUpdated(t.ID, accountID, dueDate, dueDateHasClock)
        }
        if len(t.AssigneeID) > 0 {
            t.UpdateStatus(accountID, TASK_STATUS_ASSIGNED)
        } else {
            t.UpdateStatus(accountID, TASK_STATUS_NOT_ASSIGNED)
        }
    }
    return true
}

func (t *Task) UpdateAssignee(accountID string, candidateIDs []string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if len(candidateIDs) == 1 {
        if err := db.C(COLLECTION_TASKS).UpdateId(
            t.ID,
            bson.M{
                "$set": bson.M{
                    "assignee": candidateIDs[0],
                    "status":   TASK_STATUS_ASSIGNED,
                },
            },
        ); err != nil {
            _Log.Warn(err.Error())
            return false
        }
        _Manager.TaskActivity.AssigneeChanged(t.ID, accountID, candidateIDs[0])
        _Manager.Report.CountTaskAssignedPerAccount(accountID)
        if t.Status != TASK_STATUS_ASSIGNED {
            _Manager.TaskActivity.StatusChanged(t.ID, accountID, TASK_STATUS_ASSIGNED)
        }
    } else {
        if err := db.C(COLLECTION_TASKS).UpdateId(
            t.ID,
            bson.M{
				"$set": bson.M{
                    "assignee":            "",
                    "candidates":          candidateIDs,
                    "status":              TASK_STATUS_NOT_ASSIGNED,
                    "counters.candidates": len(candidateIDs),
                },
            },
        ); err != nil {
            _Log.Warn(err.Error())
            return false
        }
        _Manager.TaskActivity.CandidateAdded(t.ID, accountID, candidateIDs)
        if t.Status != TASK_STATUS_NOT_ASSIGNED {
            _Manager.TaskActivity.StatusChanged(t.ID, accountID, TASK_STATUS_NOT_ASSIGNED)
        }
    }
    return true
}

func (t *Task) IsAssignor(accountID string) bool {
    if t.AssignorID == accountID {
        return true
    }
    return false
}

func (t *Task) IsAssignee(accountID string) bool {
    if t.AssigneeID == accountID {
        return true
    }
    return false
}

func (t *Task) IsWatcher(accountID string) bool {
    for _, watcherID := range t.WatcherIDs {
        if accountID == watcherID {
            return true
        }
    }
    return false
}

func (t *Task) IsEditor(accountID string) bool {
    for _, editorID := range t.EditorIDs {
        if accountID == editorID {
            return true
        }
    }
    return false
}

func (t *Task) IsCandidate(accountID string) bool {
    for _, candidateID := range t.CandidateIDs {
        if accountID == candidateID {
            return true
        }
    }
    return false
}

func (t *Task) GetAccess(accountID string) TaskAccess {
    a := TaskAccess{}
    switch {
    case t.IsAssignor(accountID):
        a[TASK_ACCESS_ADD_CANDIDATE] = true
        a[TASK_ACCESS_CHANGE_PRIORITY] = true
        a[TASK_ACCESS_DELETE] = true
        fallthrough
    case t.IsAssignee(accountID):
        a[TASK_ACCESS_CHANGE_ASSIGNEE] = true
        fallthrough
    case t.IsEditor(accountID):
        a[TASK_ACCESS_ADD_WATCHER] = true
        a[TASK_ACCESS_REMOVE_WATCHER] = true
        a[TASK_ACCESS_ADD_EDITOR] = true
        a[TASK_ACCESS_REMOVE_EDITOR] = true
        a[TASK_ACCESS_UPDATE] = true
        a[TASK_ACCESS_REMOVE_ATTACHMENT] = true
        fallthrough
    case t.IsWatcher(accountID):
        a[TASK_ACCESS_ADD_ATTACHMENT] = true
        a[TASK_ACCESS_COMMENT] = true
        a[TASK_ACCESS_ADD_LABEL] = true
        a[TASK_ACCESS_REMOVE_LABEL] = true
        a[TASK_ACCESS_READ] = true
    case t.IsCandidate(accountID):
        a[TASK_ACCESS_PICK_TASK] = true
        a[TASK_ACCESS_ADD_LABEL] = true
        a[TASK_ACCESS_REMOVE_LABEL] = true
        a[TASK_ACCESS_READ] = true
        a[TASK_ACCESS_COMMENT] = true
    }

    return a
}

func (t *Task) GetAccessArray(accountID string) []int {
    access := t.GetAccess(accountID)
    r := make([]int, 0, len(access))
    for k, v := range access {
        if v {
            r = append(r, k)
        }
    }
    return r
}

func (t *Task) GetTodo(todoID int) *TaskToDo {
    for _, v := range t.ToDos {
        if v.ID == todoID {
            return &v
        }
    }
    return nil
}

func (t *Task) HasAccess(accountID string, a int) bool {
    access := t.GetAccess(accountID)
    if access[a] {
        return true
    }
    return false
}

// HasActivity returns TRUE if the activityID is for t otherwise returns FALSE
func (t *Task) HasActivity(activityID bson.ObjectId) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if n, err := db.C(COLLECTION_TASKS_ACTIVITIES).Find(
        bson.M{"_id": activityID, "task_id": t.ID},
    ).Count(); err != nil {
        return false
    } else if n > 0 {
        return true
    }
    return false
}

// HasLabel returns TRUE if labelID is for t otherwise return FALSE
func (t *Task) HasLabel(labelID string) bool {
    for _, id := range t.LabelIDs {
        if labelID == id {
            return true
        }
    }
    return false
}

// HasAttachment returns TRUE if attachmentID is for t otherwise returns FALSE
func (t *Task) HasAttachment(attachmentID UniversalID) bool {
    for _, id := range t.AttachmentIDs {
        if attachmentID == id {
            return true
        }
    }
    return false
}

func (t *Task) Accept(accountID string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TASKS).UpdateId(
        t.ID,
        bson.M{
            "$set": bson.M{
                "assignee":    accountID,
                "status":      TASK_STATUS_ASSIGNED,
                "last_update": Timestamp(),
            },
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    _Manager.TaskActivity.StatusChanged(t.ID, accountID, TASK_STATUS_ASSIGNED)
    return true

}

func (t *Task) Reject(accountID, reason string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if t.Counters.Candidates > 1 {
        if err := db.C(COLLECTION_TASKS).Update(
            bson.M{
                "_id":        t.ID,
                "candidates": accountID,
            },
            bson.M{
                "$pull": bson.M{"candidates": accountID},
                "$inc":  bson.M{"counters.candidates": -1},
            },
        ); err != nil {
            _Log.Warn(err.Error())
            return false
        }

        _Manager.TaskActivity.CandidateRemoved(t.ID, accountID, []string{accountID})
    } else {
        if err := db.C(COLLECTION_TASKS).Update(
            bson.M{
                "_id":        t.ID,
                "candidates": accountID,
            },
            bson.M{
                "$set": bson.M{
                    "status":      TASK_STATUS_REJECTED,
                    "last_update": Timestamp(),
                },
                "$pull": bson.M{"candidates": accountID},
                "$inc":  bson.M{"counters.candidates": -1},
            },
        ); err != nil {
            _Log.Warn(err.Error())
            return false
        }
        _Manager.TaskActivity.CandidateRemoved(t.ID, accountID, []string{accountID})
        _Manager.TaskActivity.StatusChanged(t.ID, accountID, TASK_STATUS_REJECTED)
    }

    return true

}

func (t *Task) Resign(accountID, reason string) bool {
    defer _Manager.Task.removeCache(t.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{
            "_id":      t.ID,
            "assignee": accountID,
        },
        bson.M{
            "$set": bson.M{
                "assignee":    "",
                "status":      TASK_STATUS_REJECTED,
                "last_update": Timestamp(),
            },
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }

    _Manager.TaskActivity.StatusChanged(t.ID, accountID, TASK_STATUS_REJECTED)
    return true
}
