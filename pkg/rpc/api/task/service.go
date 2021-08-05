package nestedServiceTask

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix string = "task"
)
const (
	CmdCreate            = "task/create"
	CmdRemove            = "task/remove"
	CmdAddAttachment     = "task/add_attachment"
	CmdRemoveAttachment  = "task/remove_attachment"
	CmdAddTodo           = "task/add_todo"
	CmdRemoveTodo        = "task/remove_todo"
	CmdUpdateTodo        = "task/update_todo"
	CmdAddLabel          = "task/add_label"
	CmdRemoveLabel       = "task/remove_label"
	CmdAddComment        = "task/add_comment"
	CmdRemoveComment     = "task/remove_comment"
	CmdGetMany           = "task/get_many"
	CmdGetByFilter       = "task/get_by_filter"
	CmdGetByCustomFilter = "task/get_by_custom_filter"
	CmdAddWatcher        = "task/add_watcher"
	CmdRemoveWatcher     = "task/remove_watcher"
	CmdAddEditor         = "task/add_editor"
	CmdRemoveEditor      = "task/remove_editor"
	CmdAddCandidate      = "task/add_candidate"
	CmdRemoveCandidate   = "task/remove_candidate"
	CmdGetActivities     = "task/get_activities"
	CmdGetManyActivities = "task/get_many_activities"
	CmdUpdate            = "task/update"
	CmdUpdateAssignee    = "task/update_assignee"
	CmdRespond           = "task/respond"
	CmdSetStatus         = "task/set_status"
	CmdSetState          = "task/set_state"
)

type TaskService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewTaskService(worker *api.Worker) *TaskService {
	s := new(TaskService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CmdCreate:            {MinAuthLevel: api.AuthLevelAppL1, Execute: s.create},
		CmdRemove:            {MinAuthLevel: api.AuthLevelAppL1, Execute: s.remove},
		CmdAddComment:        {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addComment},
		CmdAddAttachment:     {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addAttachment},
		CmdAddLabel:          {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addLabel},
		CmdAddTodo:           {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addTodo},
		CmdAddWatcher:        {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addWatcher},
		CmdAddCandidate:      {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addCandidate},
		CmdAddEditor:         {MinAuthLevel: api.AuthLevelAppL1, Execute: s.addEditor},
		CmdRemoveAttachment:  {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeAttachment},
		CmdRemoveLabel:       {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeLabel},
		CmdRemoveTodo:        {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeTodo},
		CmdRemoveWatcher:     {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeWatcher},
		CmdRemoveCandidate:   {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeCandidate},
		CmdRemoveEditor:      {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeEditor},
		CmdRemoveComment:     {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeComment},
		CmdGetMany:           {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getMany},
		CmdGetByFilter:       {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getByFilter},
		CmdGetByCustomFilter: {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getByCustomFilter},
		CmdGetActivities:     {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getActivities},
		CmdGetManyActivities: {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getManyActivities},
		CmdUpdate:            {MinAuthLevel: api.AuthLevelAppL1, Execute: s.update},
		CmdUpdateTodo:        {MinAuthLevel: api.AuthLevelAppL1, Execute: s.updateTodo},
		CmdUpdateAssignee:    {MinAuthLevel: api.AuthLevelAppL1, Execute: s.updateAssignee},
		CmdRespond:           {MinAuthLevel: api.AuthLevelAppL1, Execute: s.respond},
		CmdSetStatus:         {MinAuthLevel: api.AuthLevelAppL1, Execute: s.setStatus},
		CmdSetState:          {MinAuthLevel: api.AuthLevelAppL1, Execute: s.setState},
	}
	return s
}

func (s *TaskService) GetServicePrefix() string {
	return ServicePrefix
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
