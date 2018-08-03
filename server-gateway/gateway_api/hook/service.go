package nestedServiceHook

import (
    "git.ronaksoftware.com/nested/server-gateway/client"
    "git.ronaksoftware.com/nested/server-gateway/gateway_api"
    "git.ronaksoftware.com/nested/server/model"
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
        CMD_ADD_PLACE_HOOK:   {api.AUTH_LEVEL_USER, s.addPlaceHook},
        CMD_ADD_ACCOUNT_HOOK: {api.AUTH_LEVEL_USER, s.addAccountHook},
        CMD_REMOVE_HOOK:      {api.AUTH_LEVEL_USER, s.removeHook},
        CMD_LIST:             {api.AUTH_LEVEL_USER, s.list},
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
