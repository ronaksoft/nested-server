package nestedServiceReport

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	SERVICE_PREFIX string = "report"
)
const (
	CMD_GET_TS_SINGLE_VAL string = "report/get_ts_single_val"
)

var (
	_Model *nested.Manager
)

type ReportService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewReportService(worker *api.Worker) api.Service {
	s := new(ReportService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_GET_TS_SINGLE_VAL: {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.ReportTimeSeriesSingleValue},
	}

	_Model = s.worker.Model()
	return s
}

func (s *ReportService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *ReportService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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

func (s *ReportService) Worker() *api.Worker {
	return s.worker
}
