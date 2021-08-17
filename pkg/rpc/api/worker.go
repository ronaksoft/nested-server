package api

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/pusher"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"strings"
	"sync"
	"time"

	"git.ronaksoft.com/nested/server/nested"
)

/*
   Creation Time: 2018 - Jul - 02
   Created by:  (ehsan)
   Maintainers:
       1.  (ehsan)
   Auditor: Ehsan N. Moosa
   Copyright Ronak Software Group 2018
*/

type ServiceInitiator func(worker *Worker) Service

// Worker
// are runnable structures which handle input requests
// services are registered with Worker
// Server
// 	|------>(n) Worker
// 	|			|----------> (n)	Service
// 	|			|----------> (1)	Mapper
// 	|			|----------> (1)	ArgumentHandler
// 	|			|----------> (1)	ResponseHandler
// 	|------>(n) ResponseWorker
type Worker struct {
	m              sync.Mutex
	wg             sync.WaitGroup
	mapper         *Mapper
	model          *nested.Manager
	argument       *ArgumentHandler
	mailer         *Mailer
	services       map[string]Service
	pusher         *pusher.Pusher
	backgroundJobs []*BackgroundJob
	flags          Flags

	// License
	license *nested.License
}

func NewWorker(model *nested.Manager, pusher *pusher.Pusher) *Worker {
	sw := new(Worker)
	sw.services = map[string]Service{}
	sw.model = model
	sw.pusher = pusher
	sw.mapper = NewMapper(sw)
	sw.argument = NewArgumentHandler(sw)
	sw.mailer = NewMailer(sw)

	return sw
}

func (sw *Worker) Execute(request *rpc.Request, response *rpc.Response) {
	var requester *nested.Account = nil
	response.RequestID = request.RequestID
	response.Format = request.Format
	response.NotImplemented()

	// Slow down the system if license has been expired
	if sw.flags.LicenseExpired {
		time.Sleep(time.Duration(sw.flags.LicenseSlowMode) * time.Second)
	}

	// authLevel initialized to UNAUTHORIZED, and if SessionSecret and SessionKey checked
	// and at the last step AppToken will be checked.
	authLevel := AuthLevelUnauthorized
	if len(request.SessionSec) > 0 && request.SessionKey.Valid() {
		if sw.Model().Session.Verify(request.SessionKey, request.SessionSec) {
			requester = sw.Model().Session.GetAccount(request.SessionKey)
			if requester == nil {
				response.Error(global.ErrUnknown, []string{"internal error"})
				return
			}
			if requester.Authority.Admin {
				authLevel = AuthLevelAdminUser
			} else {
				authLevel = AuthLevelUser
			}
		} else {
			// response with ErrSession  and go to next request
			response.Error(global.ErrInvalid, []string{"session invalid"})
			return
		}
	} else if len(request.AppToken) > 0 {
		appToken := sw.Model().Token.GetAppToken(request.AppToken)
		if appToken != nil && !appToken.Expired {
			app := sw.Model().App.GetByID(appToken.AppID)
			if app != nil && appToken.AppID == app.ID {
				requester = sw.Model().Account.GetByID(appToken.AccountID, nil)
				// TODO (Ehsan):: app levels must be set here
				authLevel = AuthLevelAppL3
			}
		}
	}

	if requester != nil && requester.Disabled {
		response.Error(global.ErrAccess, []string{"account_is_disabled"})
		return
	}

	// Increment Query Counter
	sw.Model().Report.CountRequests()
	sw.Model().Report.CountAPI(request.Command)

	// Refresh MongoDB Connection
	sw.Model().RefreshDbConnection()

	// Pass the authLevel to the appropriate service for execution
	prefix := strings.SplitN(request.Command, "/", 2)[0]
	startTime := time.Now()

	if service := sw.GetService(prefix); service != nil {
		service.ExecuteCommand(authLevel, requester, request, response)
	}
	processTime := int(time.Now().Sub(startTime).Nanoseconds() / 1000000)

	// Collect data for system report
	sw.Model().Report.CountProcessTime(processTime)
	sw.Model().Report.CountDataIn(request.PacketSize)

	return
}

func (sw *Worker) RegisterService(serviceInitiators ...ServiceInitiator) {
	for _, si := range serviceInitiators {
		s := si(sw)
		sw.services[s.GetServicePrefix()] = s
	}

}

func (sw *Worker) Argument() *ArgumentHandler {
	return sw.argument
}

func (sw *Worker) GetService(prefix string) Service {
	return sw.services[prefix]
}

func (sw *Worker) Map() *Mapper {
	return sw.mapper
}

func (sw *Worker) Mailer() *Mailer {
	return sw.mailer
}

func (sw *Worker) Model() *nested.Manager {
	return sw.model
}

func (sw *Worker) Pusher() *pusher.Pusher {
	return sw.pusher
}

func (sw *Worker) RegisterBackgroundJob(backgroundJobs ...*BackgroundJob) {
	for _, bg := range backgroundJobs {
		sw.backgroundJobs = append(sw.backgroundJobs, bg)
		go bg.Run(&sw.wg)
	}
}

func (sw *Worker) Shutdown() {
	// Shutdowns the DB and Cache Connections
	sw.model.Shutdown()

	// Shutdowns all the BackgroundJob
	for _, w := range sw.backgroundJobs {
		w.Shutdown()
	}

	// Wait for all the background jobs to finish
	sw.wg.Wait()
}

func (sw *Worker) GetFlags() Flags {
	return sw.flags
}

func (sw *Worker) ResetLicense() {
	sw.flags.LicenseSlowMode = 0
	sw.flags.LicenseExpired = false
}

func (sw *Worker) SetHealthCheckState(b bool) {
	sw.m.Lock()
	sw.flags.HealthCheckRunning = b
	sw.m.Unlock()
}
