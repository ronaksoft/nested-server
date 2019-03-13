package nestedServiceFile

import (
	"git.ronaksoftware.com/nested/server/model"
	"git.ronaksoftware.com/nested/server/server-gateway/client"
	"git.ronaksoftware.com/nested/server/server-gateway/gateway_api"
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
		FILE_CMD_GET_DOWNLOAD_TOKEN: {api.AUTH_LEVEL_APP_L3, s.getDownloadToken},
		FILE_CMD_GET_UPLOAD_TOKEN:   {api.AUTH_LEVEL_APP_L1, s.getUploadToken},
		FILE_CMD_GET_FILE:           {api.AUTH_LEVEL_APP_L1, s.getFileByID},
		FILE_CMD_GET_BY_TOKEN:       {api.AUTH_LEVEL_UNAUTHORIZED, s.getFileByToken},
		FILE_CMD_GET_RECENT_FILES:   {api.AUTH_LEVEL_APP_L3, s.getRecentFiles},
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
