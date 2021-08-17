package nestedServiceSystem

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	SERVICE_PREFIX = "system"
)
const (
	CMD_GET_COUNTERS         = "system/get_counters"
	CMD_GET_INT_CONSTANTS    = "system/get_int_constants"
	CMD_GET_STRING_CONSTANTS = "system/get_string_constants"
	CMD_SET_INT_CONSTANTS    = "system/set_int_constants"
	CMD_SET_STRING_CONSTANTS = "system/set_string_constants"
	CMD_MONITOR_STATS        = "system/stats"
	CMD_MONITOR_ONLINE_USERS = "system/online_users"
	CMD_MONITOR_ENABLE       = "system/mon_enable"
	CMD_MONITOR_DISABLE      = "system/mon_disable"
	CMD_MONITOR_ACTIVITY     = "system/mon_activity"
	CMD_LICENSE_SET          = "system/set_license"
	CMD_LICENSE_GET          = "system/get_license"
)

type SystemService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewSystemService(worker *api.Worker) api.Service {
	s := new(SystemService)
	s.worker = worker
	s.serviceCommands = api.ServiceCommands{
		CMD_GET_INT_CONSTANTS:    {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getSystemIntegerConstants},
		CMD_GET_STRING_CONSTANTS: {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getSystemStringConstants},
		CMD_SET_INT_CONSTANTS:    {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setSystemIntegerConstants},
		CMD_SET_STRING_CONSTANTS: {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setSystemStringConstants},
		CMD_GET_COUNTERS:         {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.getSystemCounters},
		CMD_MONITOR_ENABLE:       {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.enableMonitor},
		CMD_MONITOR_DISABLE:      {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.disableMonitor},
		CMD_MONITOR_STATS:        {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.getSystemStats},
		CMD_MONITOR_ONLINE_USERS: {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.onlineUsers},
		CMD_MONITOR_ACTIVITY:     {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.monitorActivity},
		CMD_LICENSE_SET:          {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setLicense},
		CMD_LICENSE_GET:          {MinAuthLevel: api.AuthLevelUser, Execute: s.getLicense},
	}
	return s
}

func (s *SystemService) GetServicePrefix() string {
	return SERVICE_PREFIX
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
