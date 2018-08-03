package nestedServiceTask

import (
    "git.ronaksoftware.com/nested/server/server-gateway/client"
    "git.ronaksoftware.com/nested/server/server-gateway/gateway_api"
    "git.ronaksoftware.com/nested/server/model"
)

const (
    SERVICE_PREFIX string = "task"
)
const (
    CMD_CREATE               = "task/create"
    CMD_REMOVE               = "task/remove"
    CMD_ADD_ATTACHMENT       = "task/add_attachment"
    CMD_REMOVE_ATTACHMENT    = "task/remove_attachment"
    CMD_ADD_TODO             = "task/add_todo"
    CMD_REMOVE_TODO          = "task/remove_todo"
    CMD_UPDATE_TODO          = "task/update_todo"
    CMD_ADD_LABEL            = "task/add_label"
    CMD_REMOVE_LABEL         = "task/remove_label"
    CMD_ADD_COMMENT          = "task/add_comment"
    CMD_REMOVE_COMMENT       = "task/remove_comment"
    CMD_GET_MANY             = "task/get_many"
    CMD_GET_BY_FILTER        = "task/get_by_filter"
    CMD_GET_BY_CUSTOM_FILTER = "task/get_by_custom_filter"
    CMD_ADD_WATCHER          = "task/add_watcher"
    CMD_REMOVE_WATCHER       = "task/remove_watcher"
    CMD_ADD_EDITOR           = "task/add_editor"
    CMD_REMOVE_EDITOR        = "task/remove_editor"
    CMD_ADD_CANDIDATE        = "task/add_candidate"
    CMD_REMOVE_CANDIDATE     = "task/remove_candidate"
    CMD_GET_ACTIVITIES       = "task/get_activities"
    CMD_GET_MANY_ACTIVITIES  = "task/get_many_activities"
    CMD_UPDATE               = "task/update"
    CMD_UPDATE_ASSIGNEE      = "task/update_assignee"
    CMD_RESPOND              = "task/respond"
    CMD_SET_STATUS           = "task/set_status"
    CMD_SET_STATE            = "task/set_state"
)

type TaskService struct {
    worker          *api.Worker
    serviceCommands api.ServiceCommands
}

func NewTaskService(worker *api.Worker) *TaskService {
    s := new(TaskService)
    s.worker = worker

    s.serviceCommands = api.ServiceCommands{
        CMD_CREATE:               {api.AUTH_LEVEL_APP_L1, s.create},
        CMD_REMOVE:               {api.AUTH_LEVEL_APP_L1, s.remove},
        CMD_ADD_COMMENT:          {api.AUTH_LEVEL_APP_L1, s.addComment},
        CMD_ADD_ATTACHMENT:       {api.AUTH_LEVEL_APP_L1, s.addAttachment},
        CMD_ADD_LABEL:            {api.AUTH_LEVEL_APP_L1, s.addLabel},
        CMD_ADD_TODO:             {api.AUTH_LEVEL_APP_L1, s.addTodo},
        CMD_ADD_WATCHER:          {api.AUTH_LEVEL_APP_L1, s.addWatcher},
        CMD_ADD_CANDIDATE:        {api.AUTH_LEVEL_APP_L1, s.addCandidate},
        CMD_ADD_EDITOR:           {api.AUTH_LEVEL_APP_L1, s.addEditor},
        CMD_REMOVE_ATTACHMENT:    {api.AUTH_LEVEL_APP_L1, s.removeAttachment},
        CMD_REMOVE_LABEL:         {api.AUTH_LEVEL_APP_L1, s.removeLabel},
        CMD_REMOVE_TODO:          {api.AUTH_LEVEL_APP_L1, s.removeTodo},
        CMD_REMOVE_WATCHER:       {api.AUTH_LEVEL_APP_L1, s.removeWatcher},
        CMD_REMOVE_CANDIDATE:     {api.AUTH_LEVEL_APP_L1, s.removeCandidate},
        CMD_REMOVE_EDITOR:        {api.AUTH_LEVEL_APP_L1, s.removeEditor},
        CMD_REMOVE_COMMENT:       {api.AUTH_LEVEL_APP_L1, s.removeComment},
        CMD_GET_MANY:             {api.AUTH_LEVEL_APP_L1, s.getMany},
        CMD_GET_BY_FILTER:        {api.AUTH_LEVEL_APP_L1, s.getByFilter},
        CMD_GET_BY_CUSTOM_FILTER: {api.AUTH_LEVEL_APP_L1, s.getByCustomFilter},
        CMD_GET_ACTIVITIES:       {api.AUTH_LEVEL_APP_L1, s.getActivities},
        CMD_GET_MANY_ACTIVITIES:  {api.AUTH_LEVEL_APP_L1, s.getManyActivities},
        CMD_UPDATE:               {api.AUTH_LEVEL_APP_L1, s.update},
        CMD_UPDATE_TODO:          {api.AUTH_LEVEL_APP_L1, s.updateTodo},
        CMD_UPDATE_ASSIGNEE:      {api.AUTH_LEVEL_APP_L1, s.updateAssignee},
        CMD_RESPOND:              {api.AUTH_LEVEL_APP_L1, s.respond},
        CMD_SET_STATUS:           {api.AUTH_LEVEL_APP_L1, s.setStatus},
        CMD_SET_STATE:            {api.AUTH_LEVEL_APP_L1, s.setState},
    }
    return s
}

func (s *TaskService) GetServicePrefix() string {
    return SERVICE_PREFIX
}

func (s *TaskService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    commandName := request.Command
    if cmd, ok := s.serviceCommands[commandName]; ok {
        if authLevel >= cmd.MinAuthLevel {
            cmd.Execute(requester, request, response)
        } else {
            response.NotAuthorized()
        }
    } else {
        response.NotImplemented()
    }
}

func (s *TaskService) Worker() *api.Worker {
    return s.worker
}
