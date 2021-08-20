package nestedServiceLabel

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix string = "label"
)
const (
	CmdCreate        string = "label/create"
	CmdRemove        string = "label/remove"
	CmdGetMany       string = "label/get_many"
	CmdUpdate        string = "label/update"
	CmdMemberAdd     string = "label/add_member"
	CmdMemberRemove  string = "label/remove_member"
	CmdMemberGetAll  string = "label/get_members"
	CmdRequest       string = "label/request"
	CmdRequestList   string = "label/get_requests"
	CmdRequestRemove string = "label/remove_request"
	CmdRequestUpdate string = "label/update_request"
)

type LabelService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewLabelService(worker *api.Worker) api.Service {
	s := new(LabelService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CmdCreate:        {MinAuthLevel: api.AuthLevelUser, Execute: s.createLabel},
		CmdGetMany:       {MinAuthLevel: api.AuthLevelUser, Execute: s.getManyLabels},
		CmdMemberGetAll:  {MinAuthLevel: api.AuthLevelUser, Execute: s.getLabelMembers},
		CmdMemberAdd:     {MinAuthLevel: api.AuthLevelUser, Execute: s.addMemberToLabel},
		CmdMemberRemove:  {MinAuthLevel: api.AuthLevelUser, Execute: s.removeMemberFromLabel},
		CmdRemove:        {MinAuthLevel: api.AuthLevelUser, Execute: s.removeLabel},
		CmdRequest:       {MinAuthLevel: api.AuthLevelUser, Execute: s.createLabelRequest},
		CmdRequestUpdate: {MinAuthLevel: api.AuthLevelUser, Execute: s.updateLabelRequest},
		CmdRequestList:   {MinAuthLevel: api.AuthLevelUser, Execute: s.listLabelRequests},
		CmdRequestRemove: {MinAuthLevel: api.AuthLevelUser, Execute: s.removeLabelRequest},
		CmdUpdate:        {MinAuthLevel: api.AuthLevelUser, Execute: s.updateLabel},
	}

	return s
}

func (s *LabelService) GetServicePrefix() string {
	return ServicePrefix
}

func (s *LabelService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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
