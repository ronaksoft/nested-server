package api

import (
    "strconv"
    "strings"
    "time"

    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server/server-gateway/client"
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
// API
// 	|------>(n) Worker
// 	|			|----------> (n)	Service
// 	|			|----------> (1)	Mapper
// 	|			|----------> (1)	ArgumentHandler
// 	|			|----------> (1)	ResponseHandler
// 	|------>(n) ResponseWorker
type Worker struct {
    server   *API
    mapper   *Mapper
    model    *nested.Manager
    pusher   *PushManager
    argument *ArgumentHandler
    mailer   *Mailer
    services map[string]Service
}

func NewWorker(server *API) *Worker {
    sw := new(Worker)
    sw.server = server
    sw.services = map[string]Service{}
    sw.model = server.model
    sw.mapper = NewMapper(sw)
    sw.argument = NewArgumentHandler(sw)
    sw.pusher = NewPushManager(sw)
    sw.mailer = NewMailer(sw)

    return sw
}
func (sw *Worker) Execute(request *nestedGateway.Request, response *nestedGateway.Response) {
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
    authLevel := AUTH_LEVEL_UNAUTHORIZED
    if len(request.SessionSec) > 0 && request.SessionKey.Valid() {
        if sw.Model().Session.Verify(request.SessionKey, request.SessionSec) {
            requester = sw.Model().Session.GetAccount(request.SessionKey)
            if requester == nil {
                response.Error(nested.ERR_UNKNOWN, []string{"internal error"})
                return
            }
            if requester.Authority.Admin {
                authLevel = AUTH_LEVEL_ADMIN_USER
            } else {
                authLevel = AUTH_LEVEL_USER
            }
        } else {
            // response with ERR_SESSION  and go to next request
            response.Error(nested.ERR_INVALID, []string{"session invalid"})
            return
        }
    } else if len(request.AppToken) > 0 {
        appToken := sw.Model().Token.GetAppToken(request.AppToken)
        if appToken != nil && !appToken.Expired {
            app := sw.Model().App.GetByID(appToken.AppID)
            if app != nil && appToken.AppID == app.ID {
                requester = sw.Model().Account.GetByID(appToken.AccountID, nil)
                // TODO (Ehsan):: app levels must be set here
                authLevel = AUTH_LEVEL_APP_L3
            }
        }
    }

    if requester != nil && requester.Disabled {
        response.Error(nested.ERR_ACCESS, []string{"account_is_disabled"})
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
    // sw.request.ResponseChannel <- *sw.response
}
func (sw *Worker) RegisterService(service Service) {
    sw.services[service.GetServicePrefix()] = service
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
func (sw *Worker) Pusher() *PushManager {
    return sw.pusher
}
func (sw *Worker) Server() *API {
    return sw.server
}
func (sw *Worker) Shutdown() {
    sw.model.Shutdown()
    sw.pusher.CloseConnection()
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

func (ae *ArgumentHandler) GetAccount(request *nestedGateway.Request, response *nestedGateway.Response) *nested.Account {
    var account *nested.Account
    if accountID, ok := request.Data["account_id"].(string); ok {
        account = ae.worker.Model().Account.GetByID(accountID, nil)
        if account == nil {
            response.Error(nested.ERR_INVALID, []string{"account_id"})
            return nil
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"account_id"})
        return nil
    }
    return account
}
func (ae *ArgumentHandler) GetAccounts(request *nestedGateway.Request, response *nestedGateway.Response) []nested.Account {
    var uniqueAccountIDs []string
    var accounts []nested.Account
    if csAccountIDs, ok := request.Data["account_id"].(string); ok {
        accountIDs := strings.SplitN(csAccountIDs, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
        mapAccountIDs := nested.MB{}
        for _, accountID := range accountIDs {
            mapAccountIDs[accountID] = true
        }
        for accountID := range mapAccountIDs {
            uniqueAccountIDs = append(uniqueAccountIDs, accountID)
        }
        accounts = ae.worker.Model().Account.GetAccountsByIDs(uniqueAccountIDs)
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"account_id"})
    }
    return accounts
}
func (ae *ArgumentHandler) GetAccountIDs(request *nestedGateway.Request, response *nestedGateway.Response) []string {
    var uniqueAccountIDs []string
    if csAccountIDs, ok := request.Data["account_id"].(string); ok {
        accountIDs := strings.SplitN(csAccountIDs, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
        mapAccountIDs := nested.MB{}
        for _, accountID := range accountIDs {
            mapAccountIDs[accountID] = true
        }
        for accountID := range mapAccountIDs {
            uniqueAccountIDs = append(uniqueAccountIDs, accountID)
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"account_id"})
    }
    return uniqueAccountIDs
}
func (ae *ArgumentHandler) GetComment(request *nestedGateway.Request, response *nestedGateway.Response) *nested.Comment {
    var comment *nested.Comment
    if commentID, ok := request.Data["comment_id"].(string); ok {
        if bson.IsObjectIdHex(commentID) {
            comment = ae.worker.Model().Post.GetCommentByID(bson.ObjectIdHex(commentID))
            if comment == nil {
                response.Error(nested.ERR_UNAVAILABLE, []string{"comment_id"})
                return nil
            }
        } else {
            response.Error(nested.ERR_INVALID, []string{"comment_id"})
            return nil
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"comment_id"})
        return nil
    }
    return comment
}
func (ae *ArgumentHandler) GetLabel(request *nestedGateway.Request, response *nestedGateway.Response) *nested.Label {
    var label *nested.Label
    if labelID, ok := request.Data["label_id"].(string); ok {
        label = ae.worker.Model().Label.GetByID(labelID)
        if label == nil {
            response.Error(nested.ERR_INVALID, []string{"label_id"})
            return nil
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"label_id"})
        return nil
    }
    return label
}
func (ae *ArgumentHandler) GetLabelRequest(request *nestedGateway.Request, response *nestedGateway.Response) *nested.LabelRequest {
    var labelRequest *nested.LabelRequest

    if labelRequestID, ok := request.Data["request_id"].(string); ok {
        if !bson.IsObjectIdHex(labelRequestID) {
            response.Error(nested.ERR_INVALID, []string{"request_id"})
            return nil
        }
        labelRequest = ae.worker.Model().Label.GetRequestByID(bson.ObjectIdHex(labelRequestID))
        if labelRequest == nil {
            response.Error(nested.ERR_INVALID, []string{"request_id"})
            return nil
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"request_id"})
        return nil
    }
    return labelRequest
}
func (ae *ArgumentHandler) GetPlace(request *nestedGateway.Request, response *nestedGateway.Response) *nested.Place {
    var place *nested.Place
    if placeID, ok := request.Data["place_id"].(string); ok {
        place = ae.worker.Model().Place.GetByID(placeID, nil)
        if place == nil {
            response.Error(nested.ERR_INVALID, []string{"place_id"})
            return nil
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"place_id"})
        return nil
    }
    return place
}
func (ae *ArgumentHandler) GetPost(request *nestedGateway.Request, response *nestedGateway.Response) *nested.Post {
    var post *nested.Post
    if postID, ok := request.Data["post_id"].(string); ok {
        if bson.IsObjectIdHex(postID) {
            post = ae.worker.Model().Post.GetPostByID(bson.ObjectIdHex(postID))
            if post == nil {
                response.Error(nested.ERR_UNAVAILABLE, []string{"post_id"})
                return nil
            }
        } else {
            response.Error(nested.ERR_INVALID, []string{"post_id"})
            return nil
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"post_id"})
        return nil
    }
    return post
}
func (ae *ArgumentHandler) GetPagination(request *nestedGateway.Request) nested.Pagination {
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
func (ae *ArgumentHandler) GetTask(request *nestedGateway.Request, response *nestedGateway.Response) *nested.Task {
    var task *nested.Task
    if taskID, ok := request.Data["task_id"].(string); ok {
        if bson.IsObjectIdHex(taskID) {
            task = ae.worker.Model().Task.GetByID(bson.ObjectIdHex(taskID))
            if task == nil {
                response.Error(nested.ERR_UNAVAILABLE, []string{"task_id"})
                return nil
            }
        } else {
            response.Error(nested.ERR_INVALID, []string{"task_id"})
            return nil
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"task_id"})
        return nil
    }
    return task
}
