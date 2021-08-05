package nestedServiceTask

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
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
		CMD_CREATE:               {MinAuthLevel: api.AuthLevelAppL1, Execute: s.create},
		CMD_REMOVE:               {MinAuthLevel: api.AuthLevelAppL1, Execute: s.remove},
		CMD_ADD_COMMENT:          {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addComment},
		CMD_ADD_ATTACHMENT:       {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addAttachment},
		CMD_ADD_LABEL:            {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addLabel},
		CMD_ADD_TODO:             {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addTodo},
		CMD_ADD_WATCHER:          {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addWatcher},
		CMD_ADD_CANDIDATE:        {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addCandidate},
		CMD_ADD_EDITOR:           {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addEditor},
		CMD_REMOVE_ATTACHMENT:    {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeAttachment},
		CMD_REMOVE_LABEL:         {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeLabel},
		CMD_REMOVE_TODO:          {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeTodo},
		CMD_REMOVE_WATCHER:       {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeWatcher},
		CMD_REMOVE_CANDIDATE:     {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeCandidate},
		CMD_REMOVE_EDITOR:        {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeEditor},
		CMD_REMOVE_COMMENT:       {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeComment},
		CMD_GET_MANY:             {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getMany},
		CMD_GET_BY_FILTER:        {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getByFilter},
		CMD_GET_BY_CUSTOM_FILTER: {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getByCustomFilter},
		CMD_GET_ACTIVITIES:       {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getActivities},
		CMD_GET_MANY_ACTIVITIES:  {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getManyActivities},
		CMD_UPDATE:               {MinAuthLevel: api.AuthLevelAppL1, Execute: s.update},
		CMD_UPDATE_TODO:          {MinAuthLevel: api.AuthLevelAppL1, Execute: s.updateTodo},
		CMD_UPDATE_ASSIGNEE:      {MinAuthLevel: api.AuthLevelAppL1, Execute: s.updateAssignee},
		CMD_RESPOND:              {MinAuthLevel: api.AuthLevelAppL1, Execute: s.respond},
		CMD_SET_STATUS:           {MinAuthLevel: api.AuthLevelAppL1, Execute: s.setStatus},
		CMD_SET_STATE:            {MinAuthLevel: api.AuthLevelAppL1, Execute: s.setState},
	}
	return s
}

func (s *TaskService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *TaskService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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
