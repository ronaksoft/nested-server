package nestedServiceNotification

import (
	"git.ronaksoftware.com/nested/server/model"
	"git.ronaksoftware.com/nested/server/server-gateway/client"
	"git.ronaksoftware.com/nested/server/server-gateway/gateway_api"
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
		CMD_GET:                  {api.AUTH_LEVEL_USER, s.getNotificationByID},
		CMD_GET_ALL:              {api.AUTH_LEVEL_USER, s.getNotificationsByAccountID},
		CMD_MARK_AS_READ:         {api.AUTH_LEVEL_USER, s.markNotificationAsRead},
		CMD_MARK_AS_READ_BY_POST: {api.AUTH_LEVEL_USER, s.markNotificationAsReadByPost},
		CMD_REMOVE:               {api.AUTH_LEVEL_USER, s.removeNotification},
		CMD_RESET_COUNTER:        {api.AUTH_LEVEL_USER, s.resetNotificationCounter},
		CMD_GET_COUNTER:          {api.AUTH_LEVEL_USER, s.getNotificationCounter},
	}

	_Model = s.worker.Model()
	return s
}

func (s *NotificationService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *NotificationService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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
