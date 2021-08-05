package api

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/pusher"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"strconv"
	"strings"
	"time"

	"git.ronaksoft.com/nested/server/nested"
	"github.com/globalsign/mgo/bson"
	"gopkg.in/fzerorubigd/onion.v3"
)

/*
   Creation Time: 2018 - Jul - 02
   Created by:  (ehsan)
   Maintainers:
       1.  (ehsan)
   Auditor: Ehsan N. Moosa
   Copyright Ronak Software Group 2018
*/

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
	server   *Server
	mapper   *Mapper
	model    *nested.Manager
	argument *ArgumentHandler
	mailer   *Mailer
	services map[string]Service
}

func NewWorker(server *Server) *Worker {
	sw := new(Worker)
	sw.server = server
	sw.services = map[string]Service{}
	sw.model = server.model
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
	if sw.server.flags.LicenseExpired {
		time.Sleep(time.Duration(sw.server.flags.LicenseSlowMode) * time.Second)
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

func (sw *Worker) RegisterService(services ...Service) {
	for _, s := range services {
		sw.services[s.GetServicePrefix()] = s
	}

}

func (sw *Worker) Argument() *ArgumentHandler {
	return sw.argument
}

func (sw *Worker) Config() *onion.Onion {
	return sw.server.config
}

func (sw *Worker) GetService(prefix string) Service {
	if _, ok := sw.services[prefix]; ok {
		return sw.services[prefix]
	}
	return nil
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
	return sw.server.pusher
}

func (sw *Worker) Server() *Server {
	return sw.server
}

func (sw *Worker) Shutdown() {
	sw.model.Shutdown()
}

// ArgumentHandler provides functions for easy argument extraction
type ArgumentHandler struct {
	worker *Worker
}

func NewArgumentHandler(worker *Worker) *ArgumentHandler {
	ah := new(ArgumentHandler)
	ah.worker = worker
	return ah
}

func (ae *ArgumentHandler) GetAccount(request *rpc.Request, response *rpc.Response) *nested.Account {
	var account *nested.Account
	if accountID, ok := request.Data["account_id"].(string); ok {
		account = ae.worker.Model().Account.GetByID(accountID, nil)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return nil
	}
	return account
}

func (ae *ArgumentHandler) GetAccounts(request *rpc.Request, response *rpc.Response) []nested.Account {
	var uniqueAccountIDs []string
	var accounts []nested.Account
	if csAccountIDs, ok := request.Data["account_id"].(string); ok {
		accountIDs := strings.SplitN(csAccountIDs, ",", global.DefaultMaxResultLimit)
		mapAccountIDs := tools.MB{}
		for _, accountID := range accountIDs {
			mapAccountIDs[accountID] = true
		}
		for accountID := range mapAccountIDs {
			uniqueAccountIDs = append(uniqueAccountIDs, accountID)
		}
		accounts = ae.worker.Model().Account.GetAccountsByIDs(uniqueAccountIDs)
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
	}
	return accounts
}

func (ae *ArgumentHandler) GetAccountIDs(request *rpc.Request, response *rpc.Response) []string {
	var uniqueAccountIDs []string
	if csAccountIDs, ok := request.Data["account_id"].(string); ok {
		accountIDs := strings.SplitN(csAccountIDs, ",", global.DefaultMaxResultLimit)
		mapAccountIDs := tools.MB{}
		for _, accountID := range accountIDs {
			mapAccountIDs[accountID] = true
		}
		for accountID := range mapAccountIDs {
			uniqueAccountIDs = append(uniqueAccountIDs, accountID)
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
	}
	return uniqueAccountIDs
}

func (ae *ArgumentHandler) GetComment(request *rpc.Request, response *rpc.Response) *nested.Comment {
	var comment *nested.Comment
	if commentID, ok := request.Data["comment_id"].(string); ok {
		if bson.IsObjectIdHex(commentID) {
			comment = ae.worker.Model().Post.GetCommentByID(bson.ObjectIdHex(commentID))
			if comment == nil {
				response.Error(global.ErrUnavailable, []string{"comment_id"})
				return nil
			}
		} else {
			response.Error(global.ErrInvalid, []string{"comment_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"comment_id"})
		return nil
	}
	return comment
}

func (ae *ArgumentHandler) GetLabel(request *rpc.Request, response *rpc.Response) *nested.Label {
	var label *nested.Label
	if labelID, ok := request.Data["label_id"].(string); ok {
		label = ae.worker.Model().Label.GetByID(labelID)
		if label == nil {
			response.Error(global.ErrInvalid, []string{"label_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"label_id"})
		return nil
	}
	return label
}

func (ae *ArgumentHandler) GetLabelRequest(request *rpc.Request, response *rpc.Response) *nested.LabelRequest {
	var labelRequest *nested.LabelRequest

	if labelRequestID, ok := request.Data["request_id"].(string); ok {
		if !bson.IsObjectIdHex(labelRequestID) {
			response.Error(global.ErrInvalid, []string{"request_id"})
			return nil
		}
		labelRequest = ae.worker.Model().Label.GetRequestByID(bson.ObjectIdHex(labelRequestID))
		if labelRequest == nil {
			response.Error(global.ErrInvalid, []string{"request_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"request_id"})
		return nil
	}
	return labelRequest
}

func (ae *ArgumentHandler) GetPlace(request *rpc.Request, response *rpc.Response) *nested.Place {
	var place *nested.Place
	if placeID, ok := request.Data["place_id"].(string); ok {
		place = ae.worker.Model().Place.GetByID(placeID, nil)
		if place == nil {
			response.Error(global.ErrInvalid, []string{"place_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_id"})
		return nil
	}
	return place
}

func (ae *ArgumentHandler) GetPost(request *rpc.Request, response *rpc.Response) *nested.Post {
	var post *nested.Post
	if postID, ok := request.Data["post_id"].(string); ok {
		if bson.IsObjectIdHex(postID) {
			post = ae.worker.Model().Post.GetPostByID(bson.ObjectIdHex(postID))
			if post == nil {
				response.Error(global.ErrUnavailable, []string{"post_id"})
				return nil
			}
		} else {
			response.Error(global.ErrInvalid, []string{"post_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"post_id"})
		return nil
	}
	return post
}

func (ae *ArgumentHandler) GetPagination(request *rpc.Request) nested.Pagination {
	pg := nested.NewPagination(0, 0, 0, 0)
	if v, ok := request.Data["skip"].(float64); ok {
		pg.SetSkip(int(v))
	} else if v, ok := request.Data["skip"].(string); ok {
		skip, _ := strconv.Atoi(v)
		pg.SetSkip(skip)
	}
	if v, ok := request.Data["limit"].(float64); ok {
		pg.SetLimit(int(v))
	} else if v, ok := request.Data["limit"].(string); ok {
		limit, _ := strconv.Atoi(v)
		pg.SetLimit(limit)
	}
	if v, ok := request.Data["after"].(float64); ok {
		pg.After = int64(v)
	} else if v, ok := request.Data["after"].(string); ok {
		after, _ := strconv.Atoi(v)
		pg.After = int64(after)
	}
	if v, ok := request.Data["before"].(float64); ok {
		pg.Before = int64(v)
	} else if v, ok := request.Data["before"].(string); ok {
		before, _ := strconv.Atoi(v)
		pg.Before = int64(before)
	}
	return pg
}

func (ae *ArgumentHandler) GetTask(request *rpc.Request, response *rpc.Response) *nested.Task {
	var task *nested.Task
	if taskID, ok := request.Data["task_id"].(string); ok {
		if bson.IsObjectIdHex(taskID) {
			task = ae.worker.Model().Task.GetByID(bson.ObjectIdHex(taskID))
			if task == nil {
				response.Error(global.ErrUnavailable, []string{"task_id"})
				return nil
			}
		} else {
			response.Error(global.ErrInvalid, []string{"task_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"task_id"})
		return nil
	}
	return task
}