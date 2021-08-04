package api

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/model"
	"git.ronaksoft.com/nested/server/pkg/log"
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
	Execute      func(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response)
}

type Service interface {
	GetServicePrefix() string
	ExecuteCommand(authLevel AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response)
	Worker() *Worker
}

type API struct {
	m      sync.Mutex
	wg     *sync.WaitGroup
	config *onion.Onion
	model  *nested.Manager

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

func NewServer(config *onion.Onion, wg *sync.WaitGroup) *API {
	s := new(API)
	s.config = config
	s.wg = wg

	// Instantiate Nested Model Manager
	if model, err := nested.NewManager(
		config.GetString("INSTANCE_ID"),
		config.GetString("MONGO_DSN"),
		config.GetString("REDIS_DSN"),
		config.GetInt("DEBUG_LEVEL"),
	); err != nil {
		log.Fatal(err.Error())
	} else {
		s.model = model
	}
	// Run Model Checkups
	nested.StartupCheckups()

	// Register Bundle
	s.model.RegisterBundle(config.GetString("BUNDLE_ID"))

	// Instantiate Worker
	s.requestWorker = NewWorker(s)

	return s
}

func (s *API) RegisterBackgroundJob(backgroundJob *BackgroundJob) {
	s.backgroundWorkers = append(s.backgroundWorkers, backgroundJob)
	go backgroundJob.Run(s.wg)

}

func (s *API) Shutdown() {
	// Shutdowns the DB and Cache Connections
	s.model.Shutdown()

	// Shutdown Worker
	s.requestWorker.Shutdown()

	// Shutdowns all the BackgroundJob
	for _, w := range s.backgroundWorkers {
		w.Shutdown()
	}
}

func (s *API) GetFlags() Flags {
	return s.flags
}

func (s *API) ResetLicense() {
	s.flags.LicenseSlowMode = 0
	s.flags.LicenseExpired = false
}

func (s *API) SetHealthCheckState(b bool) {
	s.m.Lock()
	s.flags.HealthCheckRunning = b
	s.m.Unlock()
}

func (s *API) Worker() *Worker {
	return s.requestWorker
}
