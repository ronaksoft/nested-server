package nestedServiceApp

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
)

const (
	SERVICE_PREFIX = "app"
)
const (
	CMD_EXISTS         = "app/exists"
	CMD_REGISTER_APP   = "app/register"
	CMD_REMOVE_APP     = "app/remove"
	CMD_GET_MANY       = "app/get_many"
	CMD_CREATE_TOKEN   = "app/create_token"
	CMD_REVOKE_TOKEN   = "app/revoke_token"
	CMD_GET_TOKENS     = "app/get_tokens"
	CMD_HAS_TOKEN      = "app/has_token"
	CMD_SET_FAV_STATUS = "app/set_fav_status"
)

type AppService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewAppService(worker *api.Worker) *AppService {
	s := new(AppService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_CREATE_TOKEN:   {MinAuthLevel: api.AuthLevelUser, Execute: s.generateAppToken},
		CMD_SET_FAV_STATUS: {MinAuthLevel: api.AuthLevelUser, Execute: s.setFavStatus},
		CMD_REVOKE_TOKEN:   {MinAuthLevel: api.AuthLevelUser, Execute: s.revokeAppToken},
		CMD_GET_TOKENS:     {MinAuthLevel: api.AuthLevelUser, Execute: s.getTokensByAccountID},
		CMD_REMOVE_APP:     {MinAuthLevel: api.AuthLevelAppL3, Execute: s.remove},
		CMD_GET_MANY:       {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getManyApps},
		CMD_EXISTS:         {MinAuthLevel: api.AuthLevelAppL3, Execute: s.exists},
		CMD_REGISTER_APP:   {MinAuthLevel: api.AuthLevelAppL3, Execute: s.register},
		CMD_HAS_TOKEN:      {MinAuthLevel: api.AuthLevelAppL1, Execute: s.hasToken},
	}
	return s
}

func (s *AppService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *AppService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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
