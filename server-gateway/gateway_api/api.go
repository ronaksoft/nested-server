package api

import (
    "git.ronaksoftware.com/nested/server/model"
    "log"
    "os"
    "sync"
    "gopkg.in/fzerorubigd/onion.v3"
    "git.ronaksoftware.com/nested/server/server-gateway/client"
)

// AUTH_LEVEL Constants
const (
    _                       AuthLevel = iota
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
        log.Println("NewServer::Nested Manager Error::", err.Error())
        os.Exit(1)
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
