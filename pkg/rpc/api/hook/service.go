package nestedServiceHook

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix = "hook"
)
const (
	CmdAddPlaceHook   = "hook/add_place_hook"
	CmdAddAccountHook = "hook/add_account_hook"
	CmdRemoveHook     = "hook/remove"
	CmdList           = "hook/list"
)

type HookService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewHookService(worker *api.Worker) api.Service {
	s := new(HookService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CmdAddPlaceHook:   {MinAuthLevel: api.AuthLevelUser, Execute: s.addPlaceHook},
		CmdAddAccountHook: {MinAuthLevel: api.AuthLevelUser, Execute: s.addAccountHook},
		CmdRemoveHook:     {MinAuthLevel: api.AuthLevelUser, Execute: s.removeHook},
		CmdList:           {MinAuthLevel: api.AuthLevelUser, Execute: s.list},
	}

	return s
}

func (s *HookService) GetServicePrefix() string {
	return ServicePrefix
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
