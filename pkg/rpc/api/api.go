package api

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/config"
	"git.ronaksoft.com/nested/server/pkg/pusher"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"sync"
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

type Server struct {
	m      sync.Mutex
	wg     *sync.WaitGroup
	model  *nested.Manager
	pusher *pusher.Pusher

	requestWorker     *Worker
	backgroundWorkers []*BackgroundJob
	flags             Flags

	// License
	license *nested.License
}

type Flags struct {
	HealthCheckRunning bool
	LicenseExpired     bool
	LicenseSlowMode    int
}

func NewServer(wg *sync.WaitGroup, model *nested.Manager, pushFunc pusher.PushFunc) *Server {
	s := new(Server)
	s.wg = wg
	s.model = model

	// Run Model Checkups
	nested.StartupCheckups()

	// Register Bundle
	s.model.RegisterBundle(config.GetString(config.BundleID))

	// Initialize Pusher
	s.pusher = pusher.New(
		s.model,
		config.GetString(config.BundleID), config.GetString(config.SenderDomain),
		pushFunc,
	)
	// Instantiate Worker
	s.requestWorker = NewWorker(s)

	return s
}

func (s *Server) RegisterBackgroundJob(backgroundJobs ...*BackgroundJob) {
	for _, bg := range backgroundJobs {
		s.backgroundWorkers = append(s.backgroundWorkers, bg)
		go bg.Run(s.wg)
	}
}

func (s *Server) Shutdown() {
	// Shutdowns the DB and Cache Connections
	s.model.Shutdown()

	// Shutdown Worker
	s.requestWorker.Shutdown()

	// Shutdowns all the BackgroundJob
	for _, w := range s.backgroundWorkers {
		w.Shutdown()
	}

	// Wait for all the background jobs to finish
	s.wg.Wait()
}

func (s *Server) GetFlags() Flags {
	return s.flags
}

func (s *Server) ResetLicense() {
	s.flags.LicenseSlowMode = 0
	s.flags.LicenseExpired = false
}

func (s *Server) SetHealthCheckState(b bool) {
	s.m.Lock()
	s.flags.HealthCheckRunning = b
	s.m.Unlock()
}

func (s *Server) Worker() *Worker {
	return s.requestWorker
}
