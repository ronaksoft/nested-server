package nested

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"go.uber.org/zap"

	"github.com/globalsign/mgo/bson"
)

type TaskActivityManager struct{}

func NewTaskActivityManager() *TaskActivityManager {
	return new(TaskActivityManager)
}

func (tm *TaskActivityManager) Remove(activityID bson.ObjectId) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionTasksActivities).UpdateId(
		activityID,
		bson.M{"$set": bson.M{"_removed": true}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

func (tm *TaskActivityManager) GetActivityByID(activityID bson.ObjectId) *TaskActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	taskActivity := new(TaskActivity)
	if err := db.C(global.CollectionTasksActivities).FindId(activityID).One(taskActivity); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}
	return taskActivity
}

func (tm *TaskActivityManager) GetActivitiesByIDs(activityIDs []bson.ObjectId) []TaskActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	taskActivities := make([]TaskActivity, 0, len(activityIDs))
	if err := db.C(global.CollectionTasksActivities).Find(
		bson.M{"_id": bson.M{"$in": activityIDs}},
	).All(&taskActivities); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}
	return taskActivities
}

func (tm *TaskActivityManager) GetActivitiesByTaskID(taskID bson.ObjectId, pg Pagination, filter []global.TaskAction) []TaskActivity {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
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

	Q := db.C(global.CollectionTasksActivities).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
	// Log Explain Query

	if err := Q.All(&taskActivities); err != nil {
		log.Warn("Got error", zap.Error(err))
		return []TaskActivity{}
	}
	return taskActivities
}

func (tm *TaskActivityManager) Created(taskID bson.ObjectId, actorID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityCreated,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) WatcherAdded(taskID bson.ObjectId, actorID string, watcherIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:         bson.NewObjectId(),
		Action:     global.TaskActivityWatcherAdded,
		Timestamp:  Timestamp(),
		TaskID:     taskID,
		ActorID:    actorID,
		WatcherIDs: watcherIDs,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) WatcherRemoved(taskID bson.ObjectId, actorID string, watcherIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:         bson.NewObjectId(),
		Action:     global.TaskActivityWatcherRemoved,
		Timestamp:  Timestamp(),
		TaskID:     taskID,
		ActorID:    actorID,
		WatcherIDs: watcherIDs,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) EditorAdded(taskID bson.ObjectId, actorID string, editorIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityEditorAdded,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		EditorIDs: editorIDs,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) EditorRemoved(taskID bson.ObjectId, actorID string, editorIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityEditorRemoved,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		EditorIDs: editorIDs,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) AttachmentAdded(taskID bson.ObjectId, actorID string, attachmentIDs []UniversalID) *TaskActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:            bson.NewObjectId(),
		Action:        global.TaskActivityAttachmentAdded,
		Timestamp:     Timestamp(),
		TaskID:        taskID,
		ActorID:       actorID,
		AttachmentIDs: attachmentIDs,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}
	return &v
}

func (tm *TaskActivityManager) AttachmentRemoved(taskID bson.ObjectId, actorID string, attachmentIDs []UniversalID) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:            bson.NewObjectId(),
		Action:        global.TaskActivityAttachmentRemoved,
		Timestamp:     Timestamp(),
		TaskID:        taskID,
		ActorID:       actorID,
		AttachmentIDs: attachmentIDs,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) TaskTitleChanged(taskID bson.ObjectId, actorID, title string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityTitleChanged,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		Title:     title,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) TaskDescriptionChanged(taskID bson.ObjectId, actorID, desc string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityDescChanged,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		Title:     desc,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) CandidateAdded(taskID bson.ObjectId, actorID string, candidateIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:           bson.NewObjectId(),
		Action:       global.TaskActivityCandidateAdded,
		Timestamp:    Timestamp(),
		TaskID:       taskID,
		ActorID:      actorID,
		CandidateIDs: candidateIDs,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) CandidateRemoved(taskID bson.ObjectId, actorID string, candidateIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:           bson.NewObjectId(),
		Action:       global.TaskActivityCandidateRemoved,
		Timestamp:    Timestamp(),
		TaskID:       taskID,
		ActorID:      actorID,
		CandidateIDs: candidateIDs,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) StatusChanged(taskID bson.ObjectId, actorID string, newStatus TaskStatus) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityStatusChanged,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		Status:    newStatus,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) ToDoAdded(taskID bson.ObjectId, actorID, todoText string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityTodoAdded,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		ToDoText:  todoText,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) ToDoRemoved(taskID bson.ObjectId, actorID, todoText string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityTodoRemoved,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		ToDoText:  todoText,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) ToDoChanged(taskID bson.ObjectId, actorID, todoText string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityTodoChanged,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		ToDoText:  todoText,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) ToDoDone(taskID bson.ObjectId, actorID, todoText string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityTodoDone,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		ToDoText:  todoText,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) ToDoUndone(taskID bson.ObjectId, actorID, todoText string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityTodoUndone,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		ToDoText:  todoText,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) AssigneeChanged(taskID bson.ObjectId, actorID, assigneeID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:         bson.NewObjectId(),
		Action:     global.TaskActivityAssigneeChanged,
		Timestamp:  Timestamp(),
		TaskID:     taskID,
		ActorID:    actorID,
		AssigneeID: assigneeID,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) Comment(taskID bson.ObjectId, actorID string, commentText string) *TaskActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:          bson.NewObjectId(),
		Action:      global.TaskActivityComment,
		Timestamp:   Timestamp(),
		TaskID:      taskID,
		ActorID:     actorID,
		CommentText: commentText,
	}

	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}

	if err := db.C(global.CollectionTasks).UpdateId(
		taskID,
		bson.M{
			"$inc": bson.M{"counters.comments": 1},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return &v
}

func (tm *TaskActivityManager) LabelAdded(taskID bson.ObjectId, actorID string, labelIDs []string) *TaskActivity {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityLabelAdded,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		LabelIDs:  labelIDs,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}
	return &v
}

func (tm *TaskActivityManager) LabelRemoved(taskID bson.ObjectId, actorID string, labelIDs []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityLabelRemoved,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
		LabelIDs:  labelIDs,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) DueDateUpdated(taskID bson.ObjectId, actorID string, dueDate uint64, dueDateHasClock bool) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:              bson.NewObjectId(),
		Action:          global.TaskActivityDueDateUpdated,
		Timestamp:       Timestamp(),
		TaskID:          taskID,
		ActorID:         actorID,
		DueDate:         dueDate,
		DueDateHasClock: dueDateHasClock,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

func (tm *TaskActivityManager) DueDateRemoved(taskID bson.ObjectId, actorID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := TaskActivity{
		ID:        bson.NewObjectId(),
		Action:    global.TaskActivityDueDateRemoved,
		Timestamp: Timestamp(),
		TaskID:    taskID,
		ActorID:   actorID,
	}
	if err := db.C(global.CollectionTasksActivities).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

type TaskActivity struct {
	ID              bson.ObjectId     `bson:"_id" json:"_id"`
	Timestamp       uint64            `bson:"timestamp" json:"timestamp"`
	TaskID          bson.ObjectId     `bson:"task_id" json:"task_id"`
	Action          global.TaskAction `bson:"action" json:"action"`
	ActorID         string            `bson:"actor_id" json:"actor_id"`
	AssigneeID      string            `bson:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	WatcherIDs      []string          `bson:"watcher_id,omitempty" json:"watcher_id,omitempty"`
	EditorIDs       []string          `bson:"editor_id,omitempty" json:"editor_id,omitempty"`
	CandidateIDs    []string          `bson:"candidate_id,omitempty" json:"candidate_id,omitempty"`
	AttachmentIDs   []UniversalID     `bson:"attachment_id,omitempty" json:"attachment_id"`
	LabelIDs        []string          `bson:"label_id,omitempty" json:"label_id"`
	ToDoText        string            `bson:"todo_text,omitempty" json:"todo_text,omitempty"`
	Title           string            `bson:"title,omitempty" json:"title,omitempty"`
	Desc            string            `bson:"description,omitempty" json:"description,omitempty"`
	Status          TaskStatus        `bson:"status,omitempty" json:"status,omitempty"`
	CommentText     string            `bson:"comment,omitempty" json:"comment,omitempty"`
	DueDate         uint64            `bson:"due_date,omitempty" json:"due_date,omitempty"`
	DueDateHasClock bool              `bson:"due_date_has_clock,omitempty" json:"due_date_has_clock,omitempty"`
	Removed         bool              `bson:"_removed" json:"-"`
}
