package nestedServiceTask

import (
    "git.ronaksoftware.com/nested/server/model"
    "strings"
    "github.com/globalsign/mgo/bson"
    "strconv"
    "encoding/base64"
    "git.ronaksoftware.com/nested/server-gateway/client"
)

// @Command:	task/create
// @Input:	title				string		*
// @Input:	desc 				string		+
// @Input:	assignee_id			string		+
// @Input:	candidate_id		    string		+	(comma separated)
// @Input:	attachment_id		string		+	(comma separated)
// @Input:	related_to			string		+	(task_id of parent)
// @Input:  related_post        string          +   (post_id of the post this task is related to)
// @Input:	watcher_id 			string		+	(comma separated)
// @Input:  editor_id           string      +   (comma separated)
// @Input:  label_id              string       +    (comma separated)
// @Input:	todos				string		+	(base64(txt);weight[1-10],...)
// @Input:	due_date			    int 		+	(timestamp milli-seconds)
// @Input:	due_date_has_clock	bool	+	(compulsory if due_date is set)
func (s *TaskService) create(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    // Initialize Task Create Query
    tcr := nested.TaskCreateRequest{
        AssignorID: requester.ID,
    }

    if v, ok := request.Data["title"].(string); ok {
        v = strings.TrimSpace(v)
        if len(v) > 0 && len(v) < 128 {
            tcr.Title = v
        } else {
            response.Error(nested.ERR_INVALID, []string{"title"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"title"})
        return
    }
    if v, ok := request.Data["desc"].(string); ok {
        if len(v) > 0 && len(v) < 512 {
            tcr.Description = v
        }
    }
    if v, ok := request.Data["assignee_id"].(string); ok {
        assignee := s.Worker().Model().Account.GetByID(v, nil)
        if assignee != nil {
            tcr.AssigneeID = v
        }
    }
    if len(tcr.AssigneeID) == 0 {
        if vv, ok := request.Data["candidate_id"].(string); ok {
            accountIDs := strings.SplitN(vv, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
            candidates := s.Worker().Model().Account.GetAccountsByIDs(accountIDs)
            for _, c := range candidates {
                tcr.CandidateIDs = append(tcr.CandidateIDs, c.ID)
            }
            if len(candidates) == 0 {
                response.Error(nested.ERR_INVALID, []string{"candidate_id"})
                return
            }
        } else {
            response.Error(nested.ERR_INCOMPLETE, []string{"assignee_id", "candidate_id"})
            return
        }
    }
    if v, ok := request.Data["related_to"].(string); ok {
        if bson.IsObjectIdHex(v) {
            tcr.RelatedTo = bson.ObjectIdHex(v)
        }
    }
    if v, ok := request.Data["related_post"].(string); ok {
        if bson.IsObjectIdHex(v) {
            tcr.RelatedPost = bson.ObjectIdHex(v)
        }
    }
    if v, ok := request.Data["watcher_id"].(string); ok {
        accountIDs := strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
        watchers := s.Worker().Model().Account.GetAccountsByIDs(accountIDs)
        for _, watcher := range watchers {
            tcr.WatcherIDs = append(tcr.WatcherIDs, watcher.ID)
        }
    }
    if v, ok := request.Data["editor_id"].(string); ok {
        accountIDs := strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
        editors := s.Worker().Model().Account.GetAccountsByIDs(accountIDs)
        for _, editor := range editors {
            tcr.EditorIDs = append(tcr.EditorIDs, editor.ID)
        }
    }
    if v, ok := request.Data["attachment_id"].(string); ok {
        attachmentIDs := []nested.UniversalID{}
        for _, attachmentID := range strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT) {
            attachmentIDs = append(attachmentIDs, nested.UniversalID(attachmentID))
        }
        tcr.AttachmentIDs = attachmentIDs
    }
    if v, ok := request.Data["todos"].(string); ok {
        i := 0
        for _, todoRaw := range strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT) {
            i++
            todoParts := strings.SplitN(todoRaw, ";", 2)
            if vv, err := base64.StdEncoding.DecodeString(todoParts[0]); err != nil {
                continue
            } else {
                todoText := string(vv)
                todoWeight, _ := strconv.Atoi(todoParts[1])
                if todoWeight < 1 {
                    todoWeight = 1
                } else if todoWeight > 10 {
                    todoWeight = 10
                }
                tcr.Todos = append(tcr.Todos, nested.TaskToDo{
                    ID:     i,
                    Text:   todoText,
                    Weight: todoWeight,
                })
            }
        }
    }
    if v, ok := request.Data["due_date_has_clock"].(bool); ok {
        tcr.DueDateHasClock = v
    }
    if v, ok := request.Data["due_date"].(float64); ok {
        tcr.DueDate = uint64(v)
    }
    if v, ok := request.Data["label_id"].(string); ok {
        labels := s.Worker().Model().Label.GetByIDs(strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT))
        for _, l := range labels {
            if l.IsMember(requester.ID) || l.Public {
                tcr.LabelIDs = append(tcr.LabelIDs, l.ID)
            }
        }
    }
    task := s.Worker().Model().Task.CreateTask(tcr)
    if task == nil {
        response.Error(nested.ERR_UNKNOWN, []string{"could not create task"})
        return
    }

    switch task.Status {
    case nested.TASK_STATUS_ASSIGNED:
        go s.Worker().Pusher().TaskAssigned(task)
        go s.Worker().Pusher().TaskAddedToWatchers(task, requester.ID, task.WatcherIDs)
        go s.Worker().Pusher().TaskAddedToEditors(task, requester.ID, task.EditorIDs)
    case nested.TASK_STATUS_NOT_ASSIGNED:
        go s.Worker().Pusher().TaskAddedToCandidates(task, requester.ID, task.CandidateIDs)
        go s.Worker().Pusher().TaskAddedToWatchers(task, requester.ID, task.WatcherIDs)
        go s.Worker().Pusher().TaskAddedToEditors(task, requester.ID, task.EditorIDs)
    }

    response.OkWithData(nested.M{"task_id": task.ID.Hex()})
}

// @Command:	task/add_comment
// @Input:	task_id		string	*
// @Input:	txt			string	*
func (s *TaskService) addComment(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var commentText string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["txt"].(string); ok {
        commentText = strings.TrimSpace(v)
        if len(commentText) == 0 {
            response.Error(nested.ERR_INVALID, []string{"txt_length"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"txt"})
        return
    }
    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_COMMENT) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    taskActivity := task.AddComment(requester.ID, commentText)
    if taskActivity != nil {
        go s.Worker().Pusher().TaskCommentAdded(task, requester.ID, taskActivity.ID, taskActivity.CommentText)
        response.OkWithData(nested.M{"activity_id": taskActivity.ID})
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
    }
}

// @Command:	task/add_attachment
// @Input:	task_id				string	*
// @Input:	universal_id			string 	*	(comma separated)
func (s *TaskService) addAttachment(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var universalIDs []nested.UniversalID
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        response.Error(nested.ERR_INVALID, []string{"task_id"})
        return
    }
    if v, ok := request.Data["universal_id"].(string); ok {
        attachIDs := strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
        for _, attachID := range attachIDs {
            if s.Worker().Model().File.Exists(nested.UniversalID(attachID)) {
                universalIDs = append(universalIDs, nested.UniversalID(attachID))
            }
        }
        if len(universalIDs) == 0 {
            response.Error(nested.ERR_INVALID, []string{"universal_id"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"universal_id"})
        return
    }
    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_ADD_ATTACHMENT) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    if task.AddAttachments(requester.ID, universalIDs) {
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
    }
}

// @Command:	task/add_label
// @Input:	task_id				string	*
// @Input:	label_id				string	+	(comma separated)
func (s *TaskService) addLabel(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var labelIDs []string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["label_id"].(string); ok {
        labels := s.Worker().Model().Label.GetByIDs(strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT))
        for _, l := range labels {
            if l.IsMember(requester.ID) || l.Public {
                labelIDs = append(labelIDs, l.ID)
            }
        }
    }

    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_ADD_LABEL) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    if task.AddLabels(requester.ID, labelIDs) {
        go s.Worker().Pusher().TaskNewActivity(task, nested.TASK_ACTIVITY_LABEL_ADDED)
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
    }
}

// @Command:	task/add_todo
// @Input:	task_id			string	*
// @Input:	txt 				string	*
// @Input:	weight			int		+	(between 1 - 10)
func (s *TaskService) addTodo(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var todoText string
    var todoWeight int = 1
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["txt"].(string); ok {
        v = strings.TrimSpace(v)
        if len(v) == 0 {
            response.Error(nested.ERR_INVALID, []string{"txt"})
            return
        }
        todoText = v
    }
    if v, ok := request.Data["weight"].(float64); ok {
        intV := int(v)
        if intV >= 1 && intV <= 10 {
            todoWeight = intV
        }
    }
    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_UPDATE) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    taskToDo := task.AddToDo(requester.ID, todoText, todoWeight)
    if taskToDo == nil {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
        return
    }
    go s.Worker().Pusher().TaskNewActivity(task, nested.TASK_ACTIVITY_TODO_ADDED)
    response.OkWithData(nested.M{
        "todo_id": taskToDo.ID,
        "done":    taskToDo.Done,
    })
}

// @Command: task/add_watcher
// @Input:	task_id		string		*
// @Input:	watcher_id	string		*	(comma separated)
func (s *TaskService) addWatcher(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var watcherIDs []string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["watcher_id"].(string); ok {
        // only add account_ids which are exists in system and are not already in the watchers list
        accounts := s.Worker().Model().Account.GetAccountsByIDs(strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT))
        for _, account := range accounts {
            if !task.IsWatcher(account.ID) {
                watcherIDs = append(watcherIDs, account.ID)
            }
        }
    }
    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_ADD_WATCHER) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    if task.AddWatchers(requester.ID, watcherIDs) {
        go s.Worker().Pusher().TaskAddedToWatchers(task, requester.ID, watcherIDs)
        response.OkWithData(nested.M{
            "accepted_watchers": watcherIDs,
        })
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{})
    }
}

// @Command: task/add_editor
// @Input:	task_id		string		*
// @Input:	editor_id	string		*	(comma separated)
func (s *TaskService) addEditor(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var editorIDs []string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["editor_id"].(string); ok {
        // only add account_ids which are exists in system and are not already in the watchers list
        accounts := s.Worker().Model().Account.GetAccountsByIDs(strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT))
        for _, account := range accounts {
            if !task.IsEditor(account.ID) {
                editorIDs = append(editorIDs, account.ID)
            }
        }
    }
    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_ADD_EDITOR) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    if task.AddEditors(requester.ID, editorIDs) {
        go s.Worker().Pusher().TaskAddedToEditors(task, requester.ID, editorIDs)
        response.OkWithData(nested.M{
            "accepted_editors": editorIDs,
        })
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{})
    }
}

// @Command: task/update_assignee
// @Input:	task_id		string		*
// @Input:	account_id	string		* (comma separated)
func (s *TaskService) updateAssignee(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var accountIDs []string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["account_id"].(string); ok {
        // only add account_ids which are exists in system and are not already in the watchers list
        accounts := s.Worker().Model().Account.GetAccountsByIDs(strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT))
        for _, account := range accounts {
            accountIDs = append(accountIDs, account.ID)
        }
    }

    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_UPDATE) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    if task.UpdateAssignee(requester.ID, accountIDs) {
        response.Ok()
        go s.Worker().Pusher().TaskNewActivity(task, nested.TASK_ACTIVITY_ASSIGNEE_CHANGED)
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
    }
}

// @Command: task/add_candidate
// @Input:	task_id			string		*
// @Input:	candidate_id	string		*	(comma separated)
func (s *TaskService) addCandidate(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var candidateIDs []string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["candidate_id"].(string); ok {
        // only add account_ids which are exists in system and are not already in the watchers list
        accounts := s.Worker().Model().Account.GetAccountsByIDs(strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT))
        for _, account := range accounts {
            if !task.IsCandidate(account.ID) {
                candidateIDs = append(candidateIDs, account.ID)
            }
        }
    }
    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_ADD_CANDIDATE) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    if task.AddCandidates(requester.ID, candidateIDs) {
        go s.Worker().Pusher().TaskAddedToCandidates(task, requester.ID, candidateIDs)
        response.OkWithData(nested.M{
            "accepted_candidates": candidateIDs,
        })
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{})
    }
}

// @Command:	task/get_by_filter
// @Input:	filter 			string 	*	("assigned_to_me" | "created_by_me" | "watched" | "candidate" | "upcoming")
// @Input:	status_filter	int		+	(comma separated)[Max. 10 TASK_STATE]
// @Pagination
func (s *TaskService) getByFilter(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var filter string
    var statusFilter []nested.TaskStatus
    var tasks []nested.Task
    if v, ok := request.Data["filter"].(string); ok {
        filter = strings.ToLower(v)
    }
    if v, ok := request.Data["status_filter"].(string); ok {
        for _, status := range strings.SplitN(v, ",", 10) {
            s, _ := strconv.Atoi(status)
            switch nested.TaskStatus(s) {
            case nested.TASK_STATUS_COMPLETED, nested.TASK_STATUS_NOT_ASSIGNED, nested.TASK_STATUS_ASSIGNED,
                nested.TASK_STATUS_REJECTED, nested.TASK_STATUS_HOLD, nested.TASK_STATUS_CANCELED,
                nested.TASK_STATUS_FAILED, nested.TASK_STATUS_OVERDUE:
                statusFilter = append(statusFilter, nested.TaskStatus(s))
            }
        }
    }
    switch filter {
    case "assigned_to_me":
        tasks = s.Worker().Model().Task.GetByAssigneeID(
            requester.ID,
            s.Worker().Argument().GetPagination(request),
            statusFilter,
        )
    case "created_by_me":
        tasks = s.Worker().Model().Task.GetByAssignorID(
            requester.ID,
            s.Worker().Argument().GetPagination(request),
            statusFilter,
        )
    case "watched":
        tasks = s.Worker().Model().Task.GetByWatcherEditorID(
            requester.ID,
            s.Worker().Argument().GetPagination(request),
            statusFilter,
        )
    case "candidate":
        tasks = s.Worker().Model().Task.GetByCandidateID(
            requester.ID,
            s.Worker().Argument().GetPagination(request),
            statusFilter,
        )
    case "upcoming":
        tasks = s.Worker().Model().Task.GetUpcomingTasks(
            requester.ID,
            s.Worker().Argument().GetPagination(request),
        )
    default:
        response.Error(nested.ERR_INVALID, []string{"filter"})
        return
    }
    r := make([]nested.M, 0, len(tasks))
    for _, task := range tasks {
        r = append(r, s.Worker().Map().Task(requester, task, true))
    }
    response.OkWithData(nested.M{"tasks": r})
}

// @Command:    task/get_by_custom_filter
// @Input:      assignee_id         string      (comma separated)
// @Input:      assignor_id         string      (comma separated)
// @Input:      label.logic         string      ("and" | "or")
// @Input:      label_id            string      (comma separated)
// @Input:      label_title         string      (comma separated)
// @Input:      status_filter       int
// @Input:      keyword             string
// @Input:      due_date            string      (relative timestamp)
func (s *TaskService) getByCustomFilter(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var assignorIDs, assigneeIDs, labelIDs []string
    var labelLogic = "and"
    var statusFilter []nested.TaskStatus
    var keyword string
    var dueDate uint64

    if v, ok := request.Data["assignor_id"].(string); ok && len(v) > 0 {
        assignorIDs = strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
    }
    if v, ok := request.Data["assignee_id"].(string); ok && len(v) > 0 {
        assigneeIDs = strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
    }
    if v, ok := request.Data["label_id"].(string); ok && len(v) > 0 {
        labelIDs = strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
    }
    if v, ok := request.Data["label_title"].(string); ok && len(v) > 0 {
        labelTitles := strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
        labels := s.Worker().Model().Label.GetByTitles(labelTitles)
        for _, label := range labels {
            labelIDs = append(labelIDs, label.ID)
        }
    }
    if v, ok := request.Data["keyword"].(string); ok {
        keyword = v
    }
    if v, ok := request.Data["label.logic"].(string); ok {
        v = strings.ToLower(v)
        switch v {
        case "and", "or":
            labelLogic = v
        }
    }
    if v, ok := request.Data["status_filter"].(string); ok {
        for _, status := range strings.SplitN(v, ",", 10) {
            s, _ := strconv.Atoi(status)
            switch nested.TaskStatus(s) {
            case nested.TASK_STATUS_COMPLETED, nested.TASK_STATUS_NOT_ASSIGNED, nested.TASK_STATUS_ASSIGNED,
                nested.TASK_STATUS_REJECTED, nested.TASK_STATUS_HOLD, nested.TASK_STATUS_CANCELED,
                nested.TASK_STATUS_FAILED, nested.TASK_STATUS_OVERDUE:
                statusFilter = append(statusFilter, nested.TaskStatus(s))
            }
        }
    }
    if v, ok := request.Data["due_date"].(float64); ok {
        dueDate = nested.Timestamp() + uint64(v)
    }

    tasks := s.Worker().Model().Task.GetByCustomFilter(
        requester.ID,
        assignorIDs,
        assigneeIDs,
        labelIDs,
        labelLogic,
        keyword,
        s.Worker().Argument().GetPagination(request),
        statusFilter,
        dueDate,
    )
    r := make([]nested.M, 0, len(tasks))
    for _, task := range tasks {
        r = append(r, s.Worker().Map().Task(requester, task, true))
    }
    response.OkWithData(nested.M{"tasks": r})

}

// @Command:	task/get_activities
// @Input:	task_id			string	*
// @Input:	only_comments	bool		+
// @Input:	details			bool		+
// @Pagination
func (s *TaskService) getActivities(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var onlyComments, details bool
    var activities []nested.TaskActivity

    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["only_comments"].(bool); ok {
        onlyComments = v
    }
    if v, ok := request.Data["details"].(bool); ok {
        details = v
    }

    if onlyComments {
        activities = s.Worker().Model().TaskActivity.GetActivitiesByTaskID(
            task.ID,
            s.Worker().Argument().GetPagination(request),
            []nested.TaskAction{nested.TASK_ACTIVITY_COMMENT},
        )
    } else {
        activities = s.Worker().Model().TaskActivity.GetActivitiesByTaskID(
            task.ID,
            s.Worker().Argument().GetPagination(request),
            []nested.TaskAction{},
        )
    }
    r := []nested.M{}
    for _, activity := range activities {
        r = append(r, s.Worker().Map().TaskActivity(requester, activity, details))
    }
    response.OkWithData(nested.M{"activities": r})
}

// @Command:	task/get_many
// @Input:	task_id		string	*	(comma separated)
func (s *TaskService) getMany(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var taskIDs []bson.ObjectId
    if v, ok := request.Data["task_id"].(string); ok {
        ids := strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
        for _, taskID := range ids {
            if bson.IsObjectIdHex(taskID) {
                taskIDs = append(taskIDs, bson.ObjectIdHex(taskID))
            }
        }
    }
    r := make([]nested.M, 0)
    tasks := s.Worker().Model().Task.GetTasksByIDs(taskIDs)
    for _, task := range tasks {
        if task.HasAccess(requester.ID, nested.TASK_ACCESS_READ) {
            r = append(r, s.Worker().Map().Task(requester, task, true))
        }
    }
    response.OkWithData(nested.M{
        "tasks": r,
    })
}

// @Command: task/get_many_activities
// @Input:	activity_id		string	*	(comma separated)
// @Input:	details			bool		+
func (s *TaskService) getManyActivities(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var taskActivityIDs []bson.ObjectId
    var details bool
    if v, ok := request.Data["activity_id"].(string); ok {
        ids := strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
        for _, taskActivityID := range ids {
            if bson.IsObjectIdHex(taskActivityID) {
                taskActivityIDs = append(taskActivityIDs, bson.ObjectIdHex(taskActivityID))
            }
        }
    }
    if v, ok := request.Data["details"].(bool); ok {
        details = v
    }
    r := []nested.M{}
    taskActivities := s.Worker().Model().TaskActivity.GetActivitiesByIDs(taskActivityIDs)
    for _, activity := range taskActivities {
        r = append(r, s.Worker().Map().TaskActivity(requester, activity, details))
    }
    response.OkWithData(nested.M{
        "activities": r,
    })
}

// @Command:	task/remove_attachment
// @Input:	task_id			string	*
// @Input:	universal_id		string	*	(comma separated)
func (s *TaskService) removeAttachment(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var attachmentIDs []nested.UniversalID
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["universal_id"].(string); ok {

        for _, id := range strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT) {
            attachmentIDs = append(attachmentIDs, nested.UniversalID(id))
        }
    }
    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_REMOVE_ATTACHMENT) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    if task.RemoveAttachments(requester.ID, attachmentIDs) {
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
    }
}

// @Command:	task/remove
// @Input:	task_id		string	*
func (s *TaskService) remove(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }

    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_DELETE) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    if s.Worker().Model().Task.RemoveTask(task.ID) {
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
    }
}

// @Command:	task/remove_comment
// @Input:	task_id			string	*
// @Input:	activity_id		string	*
func (s *TaskService) removeComment(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var activityID bson.ObjectId
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["activity_id"].(string); ok {
        if bson.IsObjectIdHex(v) {
            activityID = bson.ObjectIdHex(v)
        } else {
            response.Error(nested.ERR_INVALID, []string{"activity_id"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"activity_id"})
        return
    }

    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_UPDATE) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    if task.HasActivity(activityID) {
        if s.Worker().Model().TaskActivity.Remove(activityID) {
            response.Ok()
        } else {
            response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
        }
    } else {
        response.Error(nested.ERR_INVALID, []string{"activity_id"})
    }
}

// @Command:	task/remove_label
// @Input:	task_id			string	*
// @Input:	label_id			string	*	(comma separated)
func (s *TaskService) removeLabel(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var labelIDs []string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["label_id"].(string); ok {
        for _, id := range strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT) {
            labelIDs = append(labelIDs, id)
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"label_id"})
        return
    }
    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_REMOVE_LABEL) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    if task.RemoveLabels(requester.ID, labelIDs) {
        go s.Worker().Pusher().TaskNewActivity(task, nested.TASK_ACTIVITY_LABEL_REMOVED)
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
    }

}

// @Command:	task/remove_todo
// @Input:	task_id			string	*
// @Input:	todo_id			int		*		(comma separated)
func (s *TaskService) removeTodo(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var todoIDs []int
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["todo_id"].(string); ok {
        for _, id := range strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT) {
            s, _ := strconv.Atoi(id)
            todoIDs = append(todoIDs, s)
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"todo_id"})
        return
    }

    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_UPDATE) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    for _, todoID := range todoIDs {
        task.RemoveToDo(requester.ID, todoID)
    }
    go s.Worker().Pusher().TaskNewActivity(task, nested.TASK_ACTIVITY_TODO_REMOVED)
    response.Ok()
}

// @Command:	task/remove_watcher
// @Input:	task_id		string		*
// @Input:	watcher_id	string		*		(comma separated)
func (s *TaskService) removeWatcher(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var watcherIDs []string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["watcher_id"].(string); ok {
        for _, id := range strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT) {
            watcherIDs = append(watcherIDs, id)
        }
    }
    if !(len(watcherIDs) == 1 && watcherIDs[0] == requester.ID) {
        if !task.HasAccess(requester.ID, nested.TASK_ACCESS_REMOVE_WATCHER) {
            response.Error(nested.ERR_ACCESS, []string{})
            return
        }
    }

    if task.RemoveWatchers(requester.ID, watcherIDs) {
        go s.Worker().Pusher().TaskNewActivity(task, nested.TASK_ACTIVITY_WATCHER_REMOVED)
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{})
    }
}

// @Command:	task/remove_editor
// @Input:	task_id		string		*
// @Input:	editor_id	string		*		(comma separated)
func (s *TaskService) removeEditor(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var editorIDs []string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["editor_id"].(string); ok {
        for _, id := range strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT) {
            editorIDs = append(editorIDs, id)
        }
    }
    if !(len(editorIDs) == 1 && editorIDs[0] == requester.ID) {
        if !task.HasAccess(requester.ID, nested.TASK_ACCESS_REMOVE_EDITOR) {
            response.Error(nested.ERR_ACCESS, []string{})
            return
        }
    }

    if task.RemoveEditors(requester.ID, editorIDs) {
        go s.Worker().Pusher().TaskNewActivity(task, nested.TASK_ACTIVITY_EDITOR_REMOVED)
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{})
    }
}

// @Command:	task/remove_candidate
// @Input:	task_id				string		*
// @Input:	candidate_id		string		*		(comma separated)
func (s *TaskService) removeCandidate(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var candidateIDs []string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["candidate_id"].(string); ok {
        for _, id := range strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT) {
            candidateIDs = append(candidateIDs, id)
        }
    }
    if !(len(candidateIDs) == 1 && candidateIDs[0] == requester.ID) {
        if !task.HasAccess(requester.ID, nested.TASK_ACCESS_REMOVE_WATCHER) {
            response.Error(nested.ERR_ACCESS, []string{})
            return
        }
    }

    if task.RemoveCandidates(requester.ID, candidateIDs) {
        go s.Worker().Pusher().TaskNewActivity(task, nested.TASK_ACTIVITY_CANDIDATE_REMOVED)
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{})
    }
}

// @Command: task/respond
// @Input:	task_id		string		*
// @Input:	response		string		*	(accept | reject | resign)
// @Input:	reason		string		+
func (s *TaskService) respond(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var respond, reason string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["response"].(string); ok {
        respond = v
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"response"})
    }
    if v, ok := request.Data["reason"].(string); ok {
        reason = v
    }
    switch  respond {
    case "accept":
        // Only candidates can accept the task if the task was not already assigned
        if !task.IsCandidate(requester.ID) {
            response.Error(nested.ERR_ACCESS, []string{"not_candidate"})
            return
        }
        if task.Status == nested.TASK_STATUS_ASSIGNED {
            response.Error(nested.ERR_ACCESS, []string{"already_assigned"})
            return
        }
        if task.Accept(requester.ID) {
            go s.Worker().Pusher().TaskAccepted(task, requester.ID)
            response.Ok()
            return
        }
    case "reject":
        // Only candidates can reject a task if the task was not already assigned
        if !task.IsCandidate(requester.ID) {
            response.Error(nested.ERR_ACCESS, []string{"not_candidate"})
            return
        }
        if task.Status == nested.TASK_STATUS_ASSIGNED {
            response.Error(nested.ERR_ACCESS, []string{"already_assigned"})
            return
        }
        if task.Reject(requester.ID, reason) {
            go s.Worker().Pusher().TaskRejected(task, requester.ID)
            response.Ok()
            return
        }
    case "resign":
        // Only assignee of the task can resign the task
        if !task.IsAssignee(requester.ID) {
            response.Error(nested.ERR_ACCESS, []string{"not_assignee"})
            return
        }
        if task.Resign(requester.ID, reason) {
            response.Ok()
            return
        }
    default:
        response.Error(nested.ERR_INVALID, []string{"response"})
        return
    }

    response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
}

// @Command: task/set_status
// @Input:	task_id		string		*
// @Input:	status		int			*		(TASK_STATUS_COMPLETED | TASK_STATUS_HOLD | TASK_STATUS_CANCELED | TASK_STATUS_FAILED)
// @Deprecated
func (s *TaskService) setStatus(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var status nested.TaskStatus
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["status"].(float64); ok {
        status = nested.TaskStatus(v)
        switch status {
        case nested.TASK_STATUS_COMPLETED, nested.TASK_STATUS_HOLD, nested.TASK_STATUS_CANCELED, nested.TASK_STATUS_FAILED:
        default:
            response.Error(nested.ERR_INVALID, []string{"status"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"status"})
        return
    }
    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_UPDATE) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    task.UpdateStatus(requester.ID, status)
    response.Ok()
}

// @Command: task/set_state
// @Input:	task_id		string		*
// @Input:	state		string		*		("complete" | "hold" | "in_progress" | "failed")
func (s *TaskService) setState(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var state string
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["state"].(string); ok {
        state = v
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"state"})
        return
    }

    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_UPDATE) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    switch strings.ToLower(state) {
    case "complete":
        task.UpdateStatus(requester.ID, nested.TASK_STATUS_COMPLETED)
        go s.Worker().Pusher().TaskCompleted(task, requester.ID)
        response.OkWithData(nested.M{"new_status": nested.TASK_STATUS_COMPLETED})
        return
    case "failed":
        task.UpdateStatus(requester.ID, nested.TASK_STATUS_FAILED)
        go s.Worker().Pusher().TaskFailed(task, requester.ID)
        response.OkWithData(nested.M{"new_status": nested.TASK_STATUS_FAILED})
        return
    case "hold":
        task.UpdateStatus(requester.ID, nested.TASK_STATUS_HOLD)
        go s.Worker().Pusher().TaskHold(task, requester.ID)
        response.OkWithData(nested.M{"new_status": nested.TASK_STATUS_HOLD})
        return
    case "in_progress":
        if task.Status != nested.TASK_STATUS_OVERDUE {
            if len(task.AssigneeID) > 0 {
                task.UpdateStatus(requester.ID, nested.TASK_STATUS_ASSIGNED)
                response.OkWithData(nested.M{"new_status": nested.TASK_STATUS_ASSIGNED})
            } else {
                task.UpdateStatus(requester.ID, nested.TASK_STATUS_NOT_ASSIGNED)
                response.OkWithData(nested.M{"new_status": nested.TASK_STATUS_NOT_ASSIGNED})
            }
            go s.Worker().Pusher().TaskInProgress(task, requester.ID)
            return
        } else {
            response.Error(nested.ERR_ACCESS, []string{"task_overdue"})
            return
        }
    default:
        response.Error(nested.ERR_INVALID, []string{"state"})
        return
    }
    response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
}

// @Command: task/update
// @Input:	task_id					string	*
// @Input:	title					string	+
// @Input:	desc 					string	+
// @Input:	due_date				int 	+	(timestamp milli-seconds)
// @Input:	due_date_has_clock		bool	+	(compulsory if due_date is set)
func (s *TaskService) update(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var title, desc string
    var dueDate uint64
    var dueDateHasClock bool

    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["title"].(string); ok {
        v = strings.TrimSpace(v)
        if len(v) > 0 && len(v) < 128 {
            title = v
        } else {
            response.Error(nested.ERR_INVALID, []string{"title"})
            return
        }
    } else {
        title = task.Title
    }
    desc = task.Description
    if v, ok := request.Data["desc"].(string); ok {
        if len(v) > 0 && len(v) <= 512 {
            desc = v
        } else {
            response.Error(nested.ERR_LIMIT, []string{"description_length"})
            return
        }
    }
    if v, ok := request.Data["due_date_has_clock"].(bool); ok {
        dueDateHasClock = v
    }
    if v, ok := request.Data["due_date"].(string); ok {
        if i, err := strconv.Atoi(v); err == nil {
            dueDate = uint64(i)
        } else {
            response.Error(nested.ERR_INVALID, []string{"due_date"})
            return
        }
    } else if v, ok := request.Data["due_date"].(float64); ok {
        dueDate = uint64(v)
    } else {
        dueDate = task.DueDate
    }

    if task.Update(requester.ID, title, desc, dueDate, dueDateHasClock) {
        response.Ok()
        go s.Worker().Pusher().TaskNewActivity(task, nested.TASK_ACTIVITY_UPDATED)
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
    }
}

// @Command:	task/update_todo
// @Input:	task_id			string	*
// @Input:	todo_id			int		*
// @Input:	txt 				string	*
// @Input:	weight			int		+	(between 1 - 10)
// @Input:	done				bool		+
func (s *TaskService) updateTodo(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var todoID int
    var todo *nested.TaskToDo
    task := s.Worker().Argument().GetTask(request, response)
    if task == nil {
        return
    }
    if v, ok := request.Data["todo_id"].(float64); ok {
        todoID = int(v)
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"todo_id"})
        return
    }
    if todo = task.GetTodo(todoID); todo == nil {
        response.Error(nested.ERR_INVALID, []string{"todo_id"})
        return
    }
    if v, ok := request.Data["txt"].(string); ok {
        v = strings.TrimSpace(v)
        if len(v) == 0 {
            response.Error(nested.ERR_INVALID, []string{"txt"})
            return
        }
        todo.Text = v
    }
    if v, ok := request.Data["weight"].(float64); ok {
        intV := int(v)
        if intV >= 1 && intV <= 10 {
            todo.Weight = intV
        }
    }
    if v, ok := request.Data["done"].(bool); ok {
        todo.Done = v
    }

    // Check requester has the right permission
    if !task.HasAccess(requester.ID, nested.TASK_ACCESS_UPDATE) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    if task.UpdateTodo(requester.ID, todo.ID, todo.Text, todo.Weight, todo.Done) {
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
    }
}
