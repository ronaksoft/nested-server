package nestedServiceSession

import (
	"git.ronaksoftware.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoftware.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoftware.com/nested/server/model"
)

const (
	SERVICE_PREFIX = "session"
)
const (
	CMD_CLOSE             = "session/close"
	CMD_CLOSE_ACTIVE      = "session/close_active"
	CMD_CLOSE_ALL_ACTIVES = "session/close_all_actives"
	CMD_RECALL            = "session/recall"
	CMD_REGISTER          = "session/register"
	CMD_GET_ACTIVES       = "session/get_actives"
)

type SessionService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewSessionService(worker *api.Worker) *SessionService {
	s := new(SessionService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_CLOSE:             {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.close},
		CMD_CLOSE_ACTIVE:      {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.closeActive},
		CMD_CLOSE_ALL_ACTIVES: {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.closeAllActives},
		CMD_RECALL:            {MinAuthLevel: api.AUTH_LEVEL_UNAUTHORIZED,Execute: s.recall},
		CMD_REGISTER:          {MinAuthLevel: api.AUTH_LEVEL_UNAUTHORIZED,Execute: s.register},
		CMD_GET_ACTIVES:       {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.getAllActives},
	}

	return s
}

func (s *SessionService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *SessionService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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
