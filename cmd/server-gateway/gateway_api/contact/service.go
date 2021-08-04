package nestedServiceContact

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/model"
)

const (
	SERVICE_PREFIX string = "contact"
)
const (
	CONTACT_CMD_ADD             string = "contact/add"
	CONTACT_CMD_ADD_FAVORITE    string = "contact/add_favorite"
	CONTACT_CMD_REMOVE          string = "contact/remove"
	CONTACT_CMD_REMOVE_FAVORITE string = "contact/remove_favorite"
	CONTACT_CMD_GET             string = "contact/get"
	CONTACT_CMD_GET_ALL         string = "contact/get_all"
)

var (
	_Model *nested.Manager
)

type ContactService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewContactService(worker *api.Worker) *ContactService {
	s := new(ContactService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CONTACT_CMD_ADD:             {MinAuthLevel: api.AuthLevelUser,Execute: s.addContact},
		CONTACT_CMD_ADD_FAVORITE:    {MinAuthLevel: api.AuthLevelUser,Execute: s.addContactToFavorite},
		CONTACT_CMD_GET:             {MinAuthLevel: api.AuthLevelUser,Execute: s.getContact},
		CONTACT_CMD_GET_ALL:         {MinAuthLevel: api.AuthLevelUser,Execute: s.getAllContacts},
		CONTACT_CMD_REMOVE:          {MinAuthLevel: api.AuthLevelUser,Execute: s.removeContact},
		CONTACT_CMD_REMOVE_FAVORITE: {MinAuthLevel: api.AuthLevelUser,Execute: s.removeContactFromFavorite},
	}

	_Model = s.worker.Model()
	return s
}

func (s *ContactService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *ContactService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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
