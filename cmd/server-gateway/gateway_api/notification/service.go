package nestedServiceNotification

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/model"
	"git.ronaksoft.com/nested/server/pkg/rpc"
)

const (
	SERVICE_PREFIX string = "notification"
)
const (
	CMD_GET                  = "notification/get"
	CMD_GET_ALL              = "notification/get_all"
	CMD_MARK_AS_READ         = "notification/mark_as_read"
	CMD_MARK_AS_READ_BY_POST = "notification/mark_as_read_by_post"
	CMD_REMOVE               = "notification/remove"
	CMD_RESET_COUNTER        = "notification/reset_counter"
	CMD_GET_COUNTER          = "notification/get_counter"
)

var (
	_Model *nested.Manager
)

type NotificationService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewNotificationService(worker *api.Worker) *NotificationService {
	s := new(NotificationService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_GET:                  {MinAuthLevel: api.AuthLevelUser, Execute: s.getNotificationByID},
		CMD_GET_ALL:              {MinAuthLevel: api.AuthLevelUser, Execute: s.getNotificationsByAccountID},
		CMD_MARK_AS_READ:         {MinAuthLevel: api.AuthLevelUser, Execute: s.markNotificationAsRead},
		CMD_MARK_AS_READ_BY_POST: {MinAuthLevel: api.AuthLevelUser, Execute: s.markNotificationAsReadByPost},
		CMD_REMOVE:               {MinAuthLevel: api.AuthLevelUser, Execute: s.removeNotification},
		CMD_RESET_COUNTER:        {MinAuthLevel: api.AuthLevelUser, Execute: s.resetNotificationCounter},
		CMD_GET_COUNTER:          {MinAuthLevel: api.AuthLevelUser, Execute: s.getNotificationCounter},
	}

	_Model = s.worker.Model()
	return s
}

func (s *NotificationService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *NotificationService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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

func (s *NotificationService) Worker() *api.Worker {
	return s.worker
}
