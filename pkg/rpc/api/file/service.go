package nestedServiceFile

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix string = "file"
)
const (
	FileCmdGetDownloadToken string = "file/get_download_token"
	FileCmdGetUploadToken   string = "file/get_upload_token"
	FileCmdGetFile          string = "file/get"
	FileCmdGetByToken       string = "file/get_by_token"
	FileCmdGetRecentFiles   string = "file/get_recent_files"
)

type FileService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewFileService(worker *api.Worker) api.Service {
	s := new(FileService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		FileCmdGetDownloadToken: {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getDownloadToken},
		FileCmdGetUploadToken:   {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getUploadToken},
		FileCmdGetFile:          {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getFileByID},
		FileCmdGetByToken:       {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getFileByToken},
		FileCmdGetRecentFiles:   {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getRecentFiles},
	}

	return s
}

func (s *FileService) GetServicePrefix() string {
	return ServicePrefix
}

func (s *FileService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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

func (s *FileService) Worker() *api.Worker {
	return s.worker
}
