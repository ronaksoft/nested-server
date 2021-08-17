package api

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
)

// AUTH_LEVEL Constants
const (
	_ AuthLevel = iota
	AuthLevelUnauthorized
	AuthLevelAppL1
	AuthLevelAppL2
	AuthLevelAppL3
	AuthLevelUser
	AuthLevelAdminUser
)

type (
	AuthLevel       byte
	ServiceCommands map[string]ServiceCommand
)

type ServiceCommand struct {
	MinAuthLevel AuthLevel
	Execute      func(requester *nested.Account, request *rpc.Request, response *rpc.Response)
}

type Service interface {
	GetServicePrefix() string
	ExecuteCommand(authLevel AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response)
	Worker() *Worker
}

type Flags struct {
	HealthCheckRunning bool
	LicenseExpired     bool
	LicenseSlowMode    int
}
