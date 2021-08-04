package nested

import (
	"fmt"

	"github.com/globalsign/mgo/bson"
)

const (
	TASK_ACTIVITY_WATCHER_ADDED      TaskAction = 0x0001
	TASK_ACTIVITY_WATCHER_REMOVED    TaskAction = 0x0002
	TASK_ACTIVITY_ATTACHMENT_ADDED   TaskAction = 0x0003
	TASK_ACTIVITY_ATTACHMENT_REMOVED TaskAction = 0x0004
	TASK_ACTIVITY_COMMENT            TaskAction = 0x0006
	TASK_ACTIVITY_TITLE_CHANGED      TaskAction = 0x0007
	TASK_ACTIVITY_DESC_CHANGED       TaskAction = 0x0008
	TASK_ACTIVITY_CANDIDATE_ADDED    TaskAction = 0x0011
	TASK_ACTIVITY_CANDIDATE_REMOVED  TaskAction = 0x0012
	TASK_ACTIVITY_TODO_ADDED         TaskAction = 0x0013
	TASK_ACTIVITY_TODO_REMOVED       TaskAction = 0x0014
	TASK_ACTIVITY_TODO_CHANGED       TaskAction = 0x0015
	TASK_ACTIVITY_TODO_DONE          TaskAction = 0x0016
	TASK_ACTIVITY_TODO_UNDONE        TaskAction = 0x0017
	TASK_ACTIVITY_STATUS_CHANGED     TaskAction = 0x0018
	TASK_ACTIVITY_LABEL_ADDED        TaskAction = 0x0019
	TASK_ACTIVITY_LABEL_REMOVED      TaskAction = 0x0020
	TASK_ACTIVITY_DUE_DATE_UPDATED   TaskAction = 0x0021
	TASK_ACTIVITY_DUE_DATE_REMOVED   TaskAction = 0x0022
	TASK_ACTIVITY_CREATED            TaskAction = 0x0023
	TASK_ACTIVITY_ASSIGNEE_CHANGED   TaskAction = 0x0024
	TASK_ACTIVITY_EDITOR_ADDED       TaskAction = 0x0025
	TASK_ACTIVITY_EDITOR_REMOVED     TaskAction = 0x0026
	TASK_ACTIVITY_UPDATED            TaskAction = 0x0100
)

type TaskActivityManager struct{}

func NewTaskActivityManager() *TaskActivityManager {
	return new(TaskActivityManager)
}
func (tm *TaskActivityManager) Remove(activityID bson.ObjectId) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).UpdateId(
		activityID,
		bson.M{"$set": bson.M{"_removed": true}},
	); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}
func (tm *TaskActivityManager) GetActivityByID(activityID bson.ObjectId) *TaskActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	taskActivity := new(TaskActivity)
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).FindId(activityID).One(taskActivity); err != nil {
		log.Warn(err.Error())
		return nil
	}
	return taskActivity
}
func (tm *TaskActivityManager) GetActivitiesByIDs(activityIDs []bson.ObjectId) []TaskActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	taskActivities := make([]TaskActivity, 0, len(activityIDs))
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Find(
		bson.M{"_id": bson.M{"$in": activityIDs}},
	).All(&taskActivities); err != nil {
		log.Warn(err.Error())
		return nil
	}
	return taskActivities
}
func (tm *TaskActivityManager) GetActivitiesByTaskID(taskID bson.ObjectId, pg Pagination, filter []TaskAction) []TaskActivity {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	taskActivities := make([]TaskActivity, pg.GetLimit())
	sortItem := "timestamp"
	sortDir := fmt.Sprintf("-%s", sortItem)
	q := bson.M{
		"task_id":  taskID,
		"_removed": false,
	}
	q, sortDir = pg.FillQuery(q, sortItem, sortDir)

	if len(filter) > 0 {
		q["action"] = bson.M{"$in": filter}
	}

	Q := db.C(global.COLLECTION_TASKS_ACTIVITIES).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
	// Log Explain Query

	if err := Q.All(&taskActivities); err != nil {
		log.Warn(err.Error())
		return []TaskActivity{}
	}
	return taskActivities
}

func (tm *TaskActivityManager) Created(taskID bson.ObjectId, actorID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_CREATED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) WatcherAdded(taskID bson.ObjectId, actorID string, watcherIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:         bson.NewObjectId(),
		Action:     TASK_ACTIVITY_WATCHER_ADDED,
		Timestamp:  Timestamp(),
		TaskID:     taskID,
		ActorID:    actorID,
		WatcherIDs: watcherIDs,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) WatcherRemoved(taskID bson.ObjectId, actorID string, watcherIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:         bson.NewObjectId(),
		Action:     TASK_ACTIVITY_WATCHER_REMOVED,
		Timestamp:  Timestamp(),
		TaskID:     taskID,
		ActorID:    actorID,
		WatcherIDs: watcherIDs,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) EditorAdded(taskID bson.ObjectId, actorID string, editorIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_EDITOR_ADDED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		EditorIDs: editorIDs,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) EditorRemoved(taskID bson.ObjectId, actorID string, editorIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_EDITOR_REMOVED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		EditorIDs: editorIDs,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) AttachmentAdded(taskID bson.ObjectId, actorID string, attachmentIDs []UniversalID) *TaskActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:            bson.NewObjectId(),
		Action:        TASK_ACTIVITY_ATTACHMENT_ADDED,
		Timestamp:     Timestamp(),
		TaskID:        taskID,
		ActorID:       actorID,
		AttachmentIDs: attachmentIDs,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
		return nil
	}
	return &v
}
func (tm *TaskActivityManager) AttachmentRemoved(taskID bson.ObjectId, actorID string, attachmentIDs []UniversalID) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:            bson.NewObjectId(),
		Action:        TASK_ACTIVITY_ATTACHMENT_REMOVED,
		Timestamp:     Timestamp(),
		TaskID:        taskID,
		ActorID:       actorID,
		AttachmentIDs: attachmentIDs,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) TaskTitleChanged(taskID bson.ObjectId, actorID, title string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_TITLE_CHANGED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		Title:     title,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) TaskDescriptionChanged(taskID bson.ObjectId, actorID, desc string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_DESC_CHANGED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		Title:     desc,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) CandidateAdded(taskID bson.ObjectId, actorID string, candidateIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:           bson.NewObjectId(),
		Action:       TASK_ACTIVITY_CANDIDATE_ADDED,
		Timestamp:    Timestamp(),
		TaskID:       taskID,
		ActorID:      actorID,
		CandidateIDs: candidateIDs,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) CandidateRemoved(taskID bson.ObjectId, actorID string, candidateIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:           bson.NewObjectId(),
		Action:       TASK_ACTIVITY_CANDIDATE_REMOVED,
		Timestamp:    Timestamp(),
		TaskID:       taskID,
		ActorID:      actorID,
		CandidateIDs: candidateIDs,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) StatusChanged(taskID bson.ObjectId, actorID string, newStatus TaskStatus) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_STATUS_CHANGED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		Status:    newStatus,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) ToDoAdded(taskID bson.ObjectId, actorID, todoText string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_TODO_ADDED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		ToDoText:  todoText,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) ToDoRemoved(taskID bson.ObjectId, actorID, todoText string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_TODO_REMOVED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		ToDoText:  todoText,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) ToDoChanged(taskID bson.ObjectId, actorID, todoText string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_TODO_CHANGED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		ToDoText:  todoText,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) ToDoDone(taskID bson.ObjectId, actorID, todoText string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_TODO_DONE,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		ToDoText:  todoText,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) ToDoUndone(taskID bson.ObjectId, actorID, todoText string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_TODO_UNDONE,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		ToDoText:  todoText,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) AssigneeChanged(taskID bson.ObjectId, actorID, assigneeID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:         bson.NewObjectId(),
		Action:     TASK_ACTIVITY_ASSIGNEE_CHANGED,
		Timestamp:  Timestamp(),
		TaskID:     taskID,
		ActorID:    actorID,
		AssigneeID: assigneeID,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) Comment(taskID bson.ObjectId, actorID string, commentText string) *TaskActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:          bson.NewObjectId(),
		Action:      TASK_ACTIVITY_COMMENT,
		Timestamp:   Timestamp(),
		TaskID:      taskID,
		ActorID:     actorID,
		CommentText: commentText,
	}

	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
		return nil
	}

	if err := db.C(global.COLLECTION_TASKS).UpdateId(
		taskID,
		bson.M{
			"$inc": bson.M{"counters.comments": 1},
		},
	); err != nil {
		log.Warn(err.Error())
	}
	return &v
}
func (tm *TaskActivityManager) LabelAdded(taskID bson.ObjectId, actorID string, labelIDs []string) *TaskActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_LABEL_ADDED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		LabelIDs:  labelIDs,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
		return nil
	}
	return &v
}
func (tm *TaskActivityManager) LabelRemoved(taskID bson.ObjectId, actorID string, labelIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_LABEL_REMOVED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		LabelIDs:  labelIDs,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) DueDateUpdated(taskID bson.ObjectId, actorID string, dueDate uint64, dueDateHasClock bool) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:              bson.NewObjectId(),
		Action:          TASK_ACTIVITY_DUE_DATE_UPDATED,
		Timestamp:       Timestamp(),
		TaskID:          taskID,
		ActorID:         actorID,
		DueDate:         dueDate,
		DueDateHasClock: dueDateHasClock,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}
func (tm *TaskActivityManager) DueDateRemoved(taskID bson.ObjectId, actorID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    TASK_ACTIVITY_DUE_DATE_REMOVED,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
	}
	if err := db.C(global.COLLECTION_TASKS_ACTIVITIES).Insert(v); err != nil {
		log.Warn(err.Error())
	}
	return
}

type TaskAction int
type TaskActivity struct {
	ID              bson.ObjectId `bson:"_id" json:"_id"`
	Timestamp       uint64        `bson:"timestamp" json:"timestamp"`
	TaskID          bson.ObjectId `bson:"task_id" json:"task_id"`
	Action          TaskAction    `bson:"action" json:"action"`
	ActorID         string        `bson:"actor_id" json:"actor_id"`
	AssigneeID      string        `bson:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	WatcherIDs      []string      `bson:"watcher_id,omitempty" json:"watcher_id,omitempty"`
	EditorIDs       []string      `bson:"editor_id,omitempty" json:"editor_id,omitempty"`
	CandidateIDs    []string      `bson:"candidate_id,omitempty" json:"candidate_id,omitempty"`
	AttachmentIDs   []UniversalID `bson:"attachment_id,omitempty" json:"attachment_id"`
	LabelIDs        []string      `bson:"label_id,omitempty" json:"label_id"`
	ToDoText        string        `bson:"todo_text,omitempty" json:"todo_text,omitempty"`
	Title           string        `bson:"title,omitempty" json:"title,omitempty"`
	Desc            string        `bson:"description,omitempty" json:"description,omitempty"`
	Status          TaskStatus    `bson:"status,omitempty" json:"status,omitempty"`
	CommentText     string        `bson:"comment,omitempty" json:"comment,omitempty"`
	DueDate         uint64        `bson:"due_date,omitempty" json:"due_date,omitempty"`
	DueDateHasClock bool          `bson:"due_date_has_clock,omitempty" json:"due_date_has_clock,omitempty"`
	Removed         bool          `bson:"_removed" json:"-"`
}
