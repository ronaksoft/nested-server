package nestedServiceClient

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix = "client"
)
const (
	CmdUploadContacts   = "client/upload_contacts"
	CmdGetServerDetails = "client/get_server_details"
	CmdSaveKey          = "client/save_key"
	CmdReadKey          = "client/read_key"
	CmdRemoveKey        = "client/remove_key"
	CmdGetAllKeys       = "client/get_all_keys"
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
		CmdUploadContacts:   {MinAuthLevel: api.AuthLevelUser, Execute: s.uploadContacts},
		CmdReadKey:          {MinAuthLevel: api.AuthLevelUser, Execute: s.getKey},
		CmdSaveKey:          {MinAuthLevel: api.AuthLevelUser, Execute: s.saveKey},
		CmdRemoveKey:        {MinAuthLevel: api.AuthLevelUser, Execute: s.removeKey},
		CmdGetAllKeys:       {MinAuthLevel: api.AuthLevelUser, Execute: s.getAllKeys},
		CmdGetServerDetails: {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getServerDetails},
	}

	_Model = s.worker.Model()
	return s
}

func (s *ClientService) GetServicePrefix() string {
	return ServicePrefix
}

func (s *ClientService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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
