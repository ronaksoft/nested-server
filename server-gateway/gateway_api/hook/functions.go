package nestedServiceHook

import (
    "git.ronaksoftware.com/nested/server-model-nested"
    "strconv"
    "github.com/globalsign/mgo/bson"
    "git.ronaksoftware.com/nested/server-gateway/client"
)

// @Command:	hook/add_place_hook
// @Input: place_id		  string	    *
// @Input: hook_name       string     *
// @Input: event_type      int        *
// @Input: url             string     *
func (s *HookService) addPlaceHook(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    var url, hookName string
    var eventType int

    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }

    if !place.IsCreator(requester.ID) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    if v, ok := request.Data["url"].(string); ok {
        url = v
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"url"})
        return
    }
    if v, ok := request.Data["hook_name"].(string); ok {
        if len(v) == 0 {
            response.Error(nested.ERR_INVALID, []string{"event_type"})
            return
        }
        hookName = v
    }
    if v, ok := request.Data["event_type"].(float64); ok {
        eventType = int(v)
    } else if v, ok := request.Data["event_type"].(string); ok {
        eventType, _ = strconv.Atoi(v)
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"event_type"})
        return
    }

    if s.Worker().Model().Hook.AddHook(requester.ID, hookName, place.ID, eventType, url) {
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
    }
}

// @Command:	hook/add_account_hook
// @Input: account_id		  string	    *
// @Input: hook_name       string     *
// @Input: event_type      int        * (0x201)
// @Input: url             string     *
func (s *HookService) addAccountHook(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var account *nested.Account
    var url, hookName string
    var eventType int

    if account = s.Worker().Argument().GetAccount(request, response); account == nil {
        return
    }

    if account.ID != requester.ID && !account.Authority.Admin {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    if v, ok := request.Data["url"].(string); ok {
        url = v
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"url"})
        return
    }
    if v, ok := request.Data["hook_name"].(string); ok {
        if len(v) == 0 {
            response.Error(nested.ERR_INVALID, []string{"event_type"})
            return
        }
        hookName = v
    }
    if v, ok := request.Data["event_type"].(float64); ok {
        eventType = int(v)
    } else if v, ok := request.Data["event_type"].(string); ok {
        eventType, _ = strconv.Atoi(v)
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"event_type"})
        return
    }

    if s.Worker().Model().Hook.AddHook(requester.ID, hookName, account.ID, eventType, url) {
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
    }
}

// @Command:	hook/remove
// @Input: hook_id          string    *
func (s *HookService) removeHook(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var hookID bson.ObjectId
    if v, ok := request.Data["hook_id"].(string); ok {
        if !bson.IsObjectIdHex(v) {
            response.Error(nested.ERR_INVALID, []string{"hook_id"})
            return
        }
        hookID = bson.ObjectIdHex(v)
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"hook_id"})
        return
    }
    if s.Worker().Model().Hook.RemoveHook(hookID) {
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{"internal_error"})
        return
    }
}

// @Command:    hook/list
func (s *HookService) list(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    hooks := s.Worker().Model().Hook.GetHooksBySetterID(
        requester.ID,
        s.Worker().Argument().GetPagination(request),
    )
    response.OkWithData(nested.M{"hooks": hooks})
}
