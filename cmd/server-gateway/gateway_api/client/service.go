package nestedServiceClient

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/model"
)

const (
	SERVICE_PREFIX = "client"
)
const (
	CMD_UPLOAD_CONTACTS    = "client/upload_contacts"
	CMD_GET_SERVER_DETAILS = "client/get_server_details"
	CMD_SAVE_KEY           = "client/save_key"
	CMD_READ_KEY           = "client/read_key"
	CMD_REMOVE_KEY         = "client/remove_key"
	CMD_GET_ALL_KEYS       = "client/get_all_keys"
)

var (
	_Model *nested.Manager
)

type ClientService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewClientService(worker *api.Worker) *ClientService {
	s := new(ClientService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_UPLOAD_CONTACTS:    {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.uploadContacts},
		CMD_READ_KEY:           {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.getKey},
		CMD_SAVE_KEY:           {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.saveKey},
		CMD_REMOVE_KEY:         {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.removeKey},
		CMD_GET_ALL_KEYS:       {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.getAllKeys},
		CMD_GET_SERVER_DETAILS: {MinAuthLevel: api.AUTH_LEVEL_UNAUTHORIZED, Execute: s.getServerDetails},
	}

	_Model = s.worker.Model()
	return s
}

func (s *ClientService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *ClientService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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

func (s *ClientService) Worker() *api.Worker {
	return s.worker
}
