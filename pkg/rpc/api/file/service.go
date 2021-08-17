package nestedServiceFile

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
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

func NewFileService(worker *api.Worker) api.Service {
	s := new(FileService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		FILE_CMD_GET_DOWNLOAD_TOKEN: {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getDownloadToken},
		FILE_CMD_GET_UPLOAD_TOKEN:   {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getUploadToken},
		FILE_CMD_GET_FILE:           {MinAuthLevel: api.AuthLevelAppL1, Execute: s.getFileByID},
		FILE_CMD_GET_BY_TOKEN:       {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getFileByToken},
		FILE_CMD_GET_RECENT_FILES:   {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getRecentFiles},
	}

	_Model = s.worker.Model()
	return s
}

func (s *FileService) GetServicePrefix() string {
	return SERVICE_PREFIX
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
