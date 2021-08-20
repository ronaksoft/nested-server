package nestedServiceSystem

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix = "system"
)
const (
	CmdGetCounters        = "system/get_counters"
	CmdGetIntConstants    = "system/get_int_constants"
	CmdGetStringConstants = "system/get_string_constants"
	CmdSetIntConstants    = "system/set_int_constants"
	CmdSetStringConstants = "system/set_string_constants"
	CmdMonitorStats       = "system/stats"
	CmdMonitorOnlineUsers = "system/online_users"
	CmdMonitorEnable      = "system/mon_enable"
	CmdMonitorDisable     = "system/mon_disable"
	CmdMonitorActivity    = "system/mon_activity"
	CmdLicenseSet         = "system/set_license"
	CmdLicenseGet         = "system/get_license"
)

type SystemService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewSystemService(worker *api.Worker) api.Service {
	s := new(SystemService)
	s.worker = worker
	s.serviceCommands = api.ServiceCommands{
		CmdGetIntConstants:    {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getSystemIntegerConstants},
		CmdGetStringConstants: {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getSystemStringConstants},
		CmdSetIntConstants:    {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setSystemIntegerConstants},
		CmdSetStringConstants: {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setSystemStringConstants},
		CmdGetCounters:        {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.getSystemCounters},
		CmdMonitorEnable:      {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.enableMonitor},
		CmdMonitorDisable:     {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.disableMonitor},
		CmdMonitorStats:       {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.getSystemStats},
		CmdMonitorOnlineUsers: {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.onlineUsers},
		CmdMonitorActivity:    {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.monitorActivity},
		CmdLicenseSet:         {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setLicense},
		CmdLicenseGet:         {MinAuthLevel: api.AuthLevelUser, Execute: s.getLicense},
	}
	return s
}

func (s *SystemService) GetServicePrefix() string {
	return ServicePrefix
}

func (s *SystemService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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

func (s *SystemService) Worker() *api.Worker {
	return s.worker
}
