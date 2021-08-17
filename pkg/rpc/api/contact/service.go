package nestedServiceContact

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix string = "contact"
)
const (
	CmdAdd            string = "contact/add"
	CmdAddFavorite    string = "contact/add_favorite"
	CmdRemove         string = "contact/remove"
	CmdRemoveFavorite string = "contact/remove_favorite"
	CmdGet            string = "contact/get"
	CmdGetAll         string = "contact/get_all"
)

var (
	_Model *nested.Manager
)

type ContactService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewContactService(worker *api.Worker) api.Service {
	s := new(ContactService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CmdAdd:            {MinAuthLevel: api.AuthLevelUser, Execute: s.addContact},
		CmdAddFavorite:    {MinAuthLevel: api.AuthLevelUser, Execute: s.addContactToFavorite},
		CmdGet:            {MinAuthLevel: api.AuthLevelUser, Execute: s.getContact},
		CmdGetAll:         {MinAuthLevel: api.AuthLevelUser, Execute: s.getAllContacts},
		CmdRemove:         {MinAuthLevel: api.AuthLevelUser, Execute: s.removeContact},
		CmdRemoveFavorite: {MinAuthLevel: api.AuthLevelUser, Execute: s.removeContactFromFavorite},
	}

	_Model = s.worker.Model()
	return s
}

func (s *ContactService) GetServicePrefix() string {
	return ServicePrefix
}

func (s *ContactService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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

func (s *ContactService) Worker() *api.Worker {
	return s.worker
}
