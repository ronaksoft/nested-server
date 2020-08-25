package nestedServiceFile

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/model"
)

const (
	SERVICE_PREFIX string = "file"
)
const (
	FILE_CMD_GET_DOWNLOAD_TOKEN string = "file/get_download_token"
	FILE_CMD_GET_UPLOAD_TOKEN   string = "file/get_upload_token"
	FILE_CMD_GET_FILE           string = "file/get"
	FILE_CMD_GET_BY_TOKEN       string = "file/get_by_token"
	FILE_CMD_GET_RECENT_FILES   string = "file/get_recent_files"
)

var (
	_Model *nested.Manager
)

type FileService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewFileService(worker *api.Worker) *FileService {
	s := new(FileService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		FILE_CMD_GET_DOWNLOAD_TOKEN: {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.getDownloadToken},
		FILE_CMD_GET_UPLOAD_TOKEN:   {MinAuthLevel: api.AUTH_LEVEL_APP_L1, Execute: s.getUploadToken},
		FILE_CMD_GET_FILE:           {MinAuthLevel: api.AUTH_LEVEL_APP_L1, Execute: s.getFileByID},
		FILE_CMD_GET_BY_TOKEN:       {MinAuthLevel: api.AUTH_LEVEL_UNAUTHORIZED, Execute: s.getFileByToken},
		FILE_CMD_GET_RECENT_FILES:   {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.getRecentFiles},
	}

	_Model = s.worker.Model()
	return s
}

func (s *FileService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *FileService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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
