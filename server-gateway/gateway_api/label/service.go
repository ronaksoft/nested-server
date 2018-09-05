package nestedServiceLabel

import (
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server/server-gateway/client"
    "git.ronaksoftware.com/nested/server/server-gateway/gateway_api"
)

const (
	SERVICE_PREFIX string = "label"
)
const (
	CMD_CREATE                       string = "label/create"
	CMD_REMOVE                       string = "label/remove"
	CMD_GET_MANY                     string = "label/get_many"
	CMD_UPDATE                       string = "label/update"
	CMD_MEMBER_ADD                   string = "label/add_member"
	CMD_MEMBER_REMOVE                string = "label/remove_member"
	CMD_MEMBER_GET_ALL               string = "label/get_members"
	CMD_REQUEST                      string = "label/request"
	CMD_REQUEST_LIST                 string = "label/get_requests"
	CMD_REQUEST_REMOVE               string = "label/remove_request"
	CMD_REQUEST_UPDATE               string = "label/update_request"

)

var (
	_Model *nested.Manager
)

type LabelService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewLabelService(worker *api.Worker) *LabelService {
	s := new(LabelService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_CREATE:                       {api.AUTH_LEVEL_USER, s.CreateLabel},
		CMD_GET_MANY:                     {api.AUTH_LEVEL_USER, s.GetManyLabels},
		CMD_MEMBER_GET_ALL:               {api.AUTH_LEVEL_USER, s.GetLabelMembers},
		CMD_MEMBER_ADD:                   {api.AUTH_LEVEL_USER, s.AddMemberToLabel},
		CMD_MEMBER_REMOVE:                {api.AUTH_LEVEL_USER, s.RemoveMemberFromLabel},
		CMD_REMOVE:                       {api.AUTH_LEVEL_USER, s.RemoveLabel},
		CMD_REQUEST:                      {api.AUTH_LEVEL_USER, s.CreateLabelRequest},
		CMD_REQUEST_UPDATE:               {api.AUTH_LEVEL_USER, s.UpdateLabelRequest},
		CMD_REQUEST_LIST:                 {api.AUTH_LEVEL_USER, s.ListLabelRequests},
		CMD_REQUEST_REMOVE:               {api.AUTH_LEVEL_USER, s.RemoveLabelRequest},
		CMD_UPDATE:                       {api.AUTH_LEVEL_USER, s.UpdateLabel},

	}

	_Model = s.worker.Model()
	return s
}

func (s *LabelService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *LabelService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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

func (s *LabelService) Worker() *api.Worker {
	return s.worker
}
