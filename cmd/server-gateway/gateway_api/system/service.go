package nestedServiceSystem

import (
	"git.ronaksoftware.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoftware.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoftware.com/nested/server/model"
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
		CMD_GET_INT_CONSTANTS:    {api.AUTH_LEVEL_UNAUTHORIZED, s.getSystemIntegerConstants},
		CMD_GET_STRING_CONSTANTS: {api.AUTH_LEVEL_UNAUTHORIZED, s.getSystemStringConstants},
		CMD_SET_INT_CONSTANTS:    {api.AUTH_LEVEL_ADMIN_USER, s.setSystemIntegerConstants},
		CMD_SET_STRING_CONSTANTS: {api.AUTH_LEVEL_ADMIN_USER, s.setSystemStringConstants},
		CMD_GET_COUNTERS:         {api.AUTH_LEVEL_ADMIN_USER, s.getSystemCounters},
		CMD_MONITOR_ENABLE:       {api.AUTH_LEVEL_ADMIN_USER, s.enableMonitor},
		CMD_MONITOR_DISABLE:      {api.AUTH_LEVEL_ADMIN_USER, s.disableMonitor},
		CMD_MONITOR_STATS:        {api.AUTH_LEVEL_ADMIN_USER, s.getSystemStats},
		CMD_MONITOR_ONLINE_USERS: {api.AUTH_LEVEL_ADMIN_USER, s.onlineUsers},
		CMD_MONITOR_ACTIVITY:     {api.AUTH_LEVEL_ADMIN_USER, s.monitorActivity},
		CMD_LICENSE_SET:          {api.AUTH_LEVEL_ADMIN_USER, s.setLicense},
		CMD_LICENSE_GET:          {api.AUTH_LEVEL_USER, s.getLicense},
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
