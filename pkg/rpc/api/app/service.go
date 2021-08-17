package nestedServiceApp

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix = "app"
)
const (
	CmdExists       = "app/exists"
	CmdRegisterApp  = "app/register"
	CmdRemoveApp    = "app/remove"
	CmdGetMany      = "app/get_many"
	CmdCreateToken  = "app/create_token"
	CmdRevokeToken  = "app/revoke_token"
	CmdGetTokens    = "app/get_tokens"
	CmdHasToken     = "app/has_token"
	CmdSetFavStatus = "app/set_fav_status"
)

type AppService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewAppService(worker *api.Worker) api.Service {
	s := new(AppService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CmdCreateToken:  {MinAuthLevel: api.AuthLevelUser, Execute: s.generateAppToken},
		CmdSetFavStatus: {MinAuthLevel: api.AuthLevelUser, Execute: s.setFavStatus},
		CmdRevokeToken:  {MinAuthLevel: api.AuthLevelUser, Execute: s.revokeAppToken},
		CmdGetTokens:    {MinAuthLevel: api.AuthLevelUser, Execute: s.getTokensByAccountID},
		CmdRemoveApp:    {MinAuthLevel: api.AuthLevelAppL3, Execute: s.remove},
		CmdGetMany:      {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getManyApps},
		CmdExists:       {MinAuthLevel: api.AuthLevelAppL3, Execute: s.exists},
		CmdRegisterApp:  {MinAuthLevel: api.AuthLevelAppL3, Execute: s.register},
		CmdHasToken:     {MinAuthLevel: api.AuthLevelAppL1, Execute: s.hasToken},
	}
	return s
}

func (s *AppService) GetServicePrefix() string {
	return ServicePrefix
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
