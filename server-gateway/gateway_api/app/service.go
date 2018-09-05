package nestedServiceApp

import (
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server/server-gateway/client"
    "git.ronaksoftware.com/nested/server/server-gateway/gateway_api"
)

const (
    SERVICE_PREFIX = "app"
)
const (
    CMD_EXISTS       = "app/exists"
    CMD_REGISTER_APP = "app/register"
    CMD_REMOVE_APP   = "app/remove"
    CMD_GET_MANY     = "app/get_many"
    CMD_CREATE_TOKEN = "app/create_token"
    CMD_REVOKE_TOKEN = "app/revoke_token"
    CMD_GET_TOKENS   = "app/get_tokens"
    CMD_HAS_TOKEN    = "app/has_token"
    CMD_SET_FAV_STATUS    = "app/set_fav_status"
)

type AppService struct {
    worker          *api.Worker
    serviceCommands api.ServiceCommands
}

func NewAppService(worker *api.Worker) *AppService {
    s := new(AppService)
    s.worker = worker

    s.serviceCommands = api.ServiceCommands{
        CMD_CREATE_TOKEN: {api.AUTH_LEVEL_USER, s.generateAppToken},
		CMD_SET_FAV_STATUS: {api.AUTH_LEVEL_USER, s.setFavStatus},
        CMD_REVOKE_TOKEN: {api.AUTH_LEVEL_USER, s.revokeAppToken},
        CMD_GET_TOKENS:   {api.AUTH_LEVEL_USER, s.getTokensByAccountID},
        CMD_REMOVE_APP:   {api.AUTH_LEVEL_APP_L3, s.remove},
        CMD_GET_MANY:     {api.AUTH_LEVEL_APP_L3, s.getManyApps},
        CMD_EXISTS:       {api.AUTH_LEVEL_APP_L3, s.exists},
        CMD_REGISTER_APP: {api.AUTH_LEVEL_APP_L3, s.register},
        CMD_HAS_TOKEN:    {api.AUTH_LEVEL_APP_L1, s.hasToken},
    }
    return s
}

func (s *AppService) GetServicePrefix() string {
    return SERVICE_PREFIX
}

func (s *AppService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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

func (s *AppService) Worker() *api.Worker {
    return s.worker
}
