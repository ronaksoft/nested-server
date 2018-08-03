package nestedServiceContact

import (
	"git.ronaksoftware.com/nested/server/model"
	"git.ronaksoftware.com/nested/server/server-gateway/client"
	"git.ronaksoftware.com/nested/server/server-gateway/gateway_api"
)

const (
	SERVICE_PREFIX string = "contact"
)
const (
	CONTACT_CMD_ADD                        string = "contact/add"
	CONTACT_CMD_ADD_FAVORITE               string = "contact/add_favorite"
	CONTACT_CMD_REMOVE                     string = "contact/remove"
	CONTACT_CMD_REMOVE_FAVORITE            string = "contact/remove_favorite"
	CONTACT_CMD_GET                        string = "contact/get"
	CONTACT_CMD_GET_ALL                    string = "contact/get_all"
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
		CONTACT_CMD_ADD:                        {api.AUTH_LEVEL_USER, s.addContact},
		CONTACT_CMD_ADD_FAVORITE:               {api.AUTH_LEVEL_USER, s.addContactToFavorite},
		CONTACT_CMD_GET:                        {api.AUTH_LEVEL_USER, s.getContact},
		CONTACT_CMD_GET_ALL:                    {api.AUTH_LEVEL_USER, s.getAllContacts},
		CONTACT_CMD_REMOVE:                     {api.AUTH_LEVEL_USER, s.removeContact},
		CONTACT_CMD_REMOVE_FAVORITE:            {api.AUTH_LEVEL_USER, s.removeContactFromFavorite},
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
