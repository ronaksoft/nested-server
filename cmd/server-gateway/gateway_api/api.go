package api

import (
	"os"
	"sync"

	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/model"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/fzerorubigd/onion.v3"
)

var (
	_Log      *zap.Logger
	_LogLevel zap.AtomicLevel
)

func init() {
	// Initialize Logger
	_LogLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	zap.NewProductionConfig()
	config := zap.NewProductionConfig()
	config.Encoding = "console"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.Level = _LogLevel
	if v, err := config.Build(); err != nil {
		os.Exit(1)
	} else {
		_Log = v
	}
}

// AUTH_LEVEL Constants
const (
	_ AuthLevel = iota
	AUTH_LEVEL_UNAUTHORIZED
	AUTH_LEVEL_APP_L1
	AUTH_LEVEL_APP_L2
	AUTH_LEVEL_APP_L3
	AUTH_LEVEL_USER
	AUTH_LEVEL_ADMIN_USER
)

type AuthLevel byte
type ServiceCommands map[string]ServiceCommand
type ServiceCommand struct {
	MinAuthLevel AuthLevel
	Execute      func(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response)
}
type Service interface {
	GetServicePrefix() string
	ExecuteCommand(authLevel AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response)
	Worker() *Worker
}

// API
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

// Flags
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
		_Log.Fatal(err.Error())
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

// RegisterBackgroundJob
func (s *API) RegisterBackgroundJob(backgroundJob *BackgroundJob) {
	s.backgroundWorkers = append(s.backgroundWorkers, backgroundJob)
	go backgroundJob.Run(s.wg)

}

// Shutdown
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

// GetFlags
func (s *API) GetFlags() Flags {
	return s.flags
}

// Reset License
func (s *API) ResetLicense() {
	s.flags.LicenseSlowMode = 0
	s.flags.LicenseExpired = false
}

// SetHealthCheckState
func (s *API) SetHealthCheckState(b bool) {
	s.m.Lock()
	s.flags.HealthCheckRunning = b
	s.m.Unlock()
}

// Worker
func (s *API) Worker() *Worker {
	return s.requestWorker
}
