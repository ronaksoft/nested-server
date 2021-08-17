package nestedServiceHook

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	SERVICE_PREFIX = "hook"
)
const (
	CMD_ADD_PLACE_HOOK   = "hook/add_place_hook"
	CMD_ADD_ACCOUNT_HOOK = "hook/add_account_hook"
	CMD_REMOVE_HOOK      = "hook/remove"
	CMD_LIST             = "hook/list"
)

type HookService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewHookService(worker *api.Worker) api.Service {
	s := new(HookService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_ADD_PLACE_HOOK:   {MinAuthLevel: api.AuthLevelUser, Execute: s.addPlaceHook},
		CMD_ADD_ACCOUNT_HOOK: {MinAuthLevel: api.AuthLevelUser, Execute: s.addAccountHook},
		CMD_REMOVE_HOOK:      {MinAuthLevel: api.AuthLevelUser, Execute: s.removeHook},
		CMD_LIST:             {MinAuthLevel: api.AuthLevelUser, Execute: s.list},
	}

	return s
}

func (s *HookService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *HookService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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

func (s *HookService) Worker() *api.Worker {
	return s.worker
}
