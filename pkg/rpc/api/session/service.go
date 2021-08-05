package nestedServiceSession

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix = "session"
)
const (
	CmdClose           = "session/close"
	CmdCloseActive     = "session/close_active"
	CmdCloseAllActives = "session/close_all_actives"
	CmdRecall          = "session/recall"
	CmdRegister        = "session/register"
	CmdGetActives      = "session/get_actives"
)

type SessionService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewSessionService(worker *api.Worker) *SessionService {
	s := new(SessionService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CmdClose:           {MinAuthLevel: api.AuthLevelUser, Execute: s.close},
		CmdCloseActive:     {MinAuthLevel: api.AuthLevelUser, Execute: s.closeActive},
		CmdCloseAllActives: {MinAuthLevel: api.AuthLevelUser, Execute: s.closeAllActives},
		CmdRecall:          {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.recall},
		CmdRegister:        {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.register},
		CmdGetActives:      {MinAuthLevel: api.AuthLevelUser, Execute: s.getAllActives},
	}

	return s
}

func (s *SessionService) GetServicePrefix() string {
	return ServicePrefix
}

func (s *SessionService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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

func (s *SessionService) Worker() *api.Worker {
	return s.worker
}
