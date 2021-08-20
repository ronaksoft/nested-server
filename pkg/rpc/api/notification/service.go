package nestedServiceNotification

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix string = "notification"
)
const (
	CmdGet              = "notification/get"
	CmdGetAll           = "notification/get_all"
	CmdMarkAsRead       = "notification/mark_as_read"
	CmdMarkAsReadByPost = "notification/mark_as_read_by_post"
	CmdRemove           = "notification/remove"
	CmdResetCounter     = "notification/reset_counter"
	CmdGetCounter       = "notification/get_counter"
)

type NotificationService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewNotificationService(worker *api.Worker) api.Service {
	s := new(NotificationService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CmdGet:              {MinAuthLevel: api.AuthLevelUser, Execute: s.getNotificationByID},
		CmdGetAll:           {MinAuthLevel: api.AuthLevelUser, Execute: s.getNotificationsByAccountID},
		CmdMarkAsRead:       {MinAuthLevel: api.AuthLevelUser, Execute: s.markNotificationAsRead},
		CmdMarkAsReadByPost: {MinAuthLevel: api.AuthLevelUser, Execute: s.markNotificationAsReadByPost},
		CmdRemove:           {MinAuthLevel: api.AuthLevelUser, Execute: s.removeNotification},
		CmdResetCounter:     {MinAuthLevel: api.AuthLevelUser, Execute: s.resetNotificationCounter},
		CmdGetCounter:       {MinAuthLevel: api.AuthLevelUser, Execute: s.getNotificationCounter},
	}

	return s
}

func (s *NotificationService) GetServicePrefix() string {
	return ServicePrefix
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
