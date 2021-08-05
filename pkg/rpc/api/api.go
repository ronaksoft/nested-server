package api

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/log"
	"git.ronaksoft.com/nested/server/pkg/pusher"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"go.uber.org/zap"
	"gopkg.in/fzerorubigd/onion.v3"
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
	config *onion.Onion
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

func NewServer(config *onion.Onion, wg *sync.WaitGroup, pushFunc pusher.PushFunc) *Server {
	s := new(Server)
	s.config = config
	s.wg = wg

	// Instantiate Nested Model Manager
	if model, err := nested.NewManager(
		config.GetString("INSTANCE_ID"),
		config.GetString("MONGO_DSN"),
		config.GetString("REDIS_DSN"),
		config.GetInt("DEBUG_LEVEL"),
	); err != nil {
		log.Fatal("we got error on initializing nested.Manager", zap.Error(err))
	} else {
		s.model = model
	}
	// Run Model Checkups
	nested.StartupCheckups()

	// Register Bundle
	s.model.RegisterBundle(config.GetString("BUNDLE_ID"))

	// Initialize Pusher
	s.pusher = pusher.New(
		s.model,
		config.GetString("BUNDLE_ID"), config.GetString("SENDER_DOMAIN"),
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
