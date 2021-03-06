package nestedServiceSystem

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/model"
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

func NewSystemService(worker *api.Worker) *SystemService {
	s := new(SystemService)
	s.worker = worker
	s.serviceCommands = api.ServiceCommands{
		CMD_GET_INT_CONSTANTS:    {MinAuthLevel: api.AUTH_LEVEL_UNAUTHORIZED, Execute: s.getSystemIntegerConstants},
		CMD_GET_STRING_CONSTANTS: {MinAuthLevel: api.AUTH_LEVEL_UNAUTHORIZED, Execute: s.getSystemStringConstants},
		CMD_SET_INT_CONSTANTS:    {MinAuthLevel: api.AUTH_LEVEL_ADMIN_USER, Execute: s.setSystemIntegerConstants},
		CMD_SET_STRING_CONSTANTS: {MinAuthLevel: api.AUTH_LEVEL_ADMIN_USER, Execute: s.setSystemStringConstants},
		CMD_GET_COUNTERS:         {MinAuthLevel: api.AUTH_LEVEL_ADMIN_USER, Execute: s.getSystemCounters},
		CMD_MONITOR_ENABLE:       {MinAuthLevel: api.AUTH_LEVEL_ADMIN_USER, Execute: s.enableMonitor},
		CMD_MONITOR_DISABLE:      {MinAuthLevel: api.AUTH_LEVEL_ADMIN_USER, Execute: s.disableMonitor},
		CMD_MONITOR_STATS:        {MinAuthLevel: api.AUTH_LEVEL_ADMIN_USER, Execute: s.getSystemStats},
		CMD_MONITOR_ONLINE_USERS: {MinAuthLevel: api.AUTH_LEVEL_ADMIN_USER, Execute: s.onlineUsers},
		CMD_MONITOR_ACTIVITY:     {MinAuthLevel: api.AUTH_LEVEL_ADMIN_USER, Execute: s.monitorActivity},
		CMD_LICENSE_SET:          {MinAuthLevel: api.AUTH_LEVEL_ADMIN_USER, Execute: s.setLicense},
		CMD_LICENSE_GET:          {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.getLicense},
	}
	return s
}

func (s *SystemService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *SystemService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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
