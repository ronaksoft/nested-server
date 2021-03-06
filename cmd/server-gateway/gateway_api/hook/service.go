package nestedServiceHook

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/model"
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

func NewHookService(worker *api.Worker) *HookService {
	s := new(HookService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_ADD_PLACE_HOOK:   {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.addPlaceHook},
		CMD_ADD_ACCOUNT_HOOK: {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.addAccountHook},
		CMD_REMOVE_HOOK:      {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.removeHook},
		CMD_LIST:             {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.list},
	}

	return s
}

func (s *HookService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *HookService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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
