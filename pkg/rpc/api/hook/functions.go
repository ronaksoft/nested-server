package nestedServiceHook

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"strconv"

	"git.ronaksoft.com/nested/server/nested"
	"github.com/globalsign/mgo/bson"
)

// @Command:	hook/add_place_hook
// @Input: place_id		  string	    *
// @Input: hook_name       string     *
// @Input: event_type      int        *
// @Input: url             string     *
func (s *HookService) addPlaceHook(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	var url, hookName string
	var eventType int

	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}

	if !place.IsCreator(requester.ID) {
		response.Error(global.ErrAccess, []string{})
		return
	}

	if v, ok := request.Data["url"].(string); ok {
		url = v
	} else {
		response.Error(global.ErrIncomplete, []string{"url"})
		return
	}
	if v, ok := request.Data["hook_name"].(string); ok {
		if len(v) == 0 {
			response.Error(global.ErrInvalid, []string{"event_type"})
			return
		}
		hookName = v
	}
	if v, ok := request.Data["event_type"].(float64); ok {
		eventType = int(v)
	} else if v, ok := request.Data["event_type"].(string); ok {
		eventType, _ = strconv.Atoi(v)
	} else {
		response.Error(global.ErrIncomplete, []string{"event_type"})
		return
	}

	if s.Worker().Model().Hook.AddHook(requester.ID, hookName, place.ID, eventType, url) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{"internal_error"})
	}
}

// @Command:	hook/add_account_hook
// @Input: account_id		  string	    *
// @Input: hook_name       string     *
// @Input: event_type      int        * (0x201)
// @Input: url             string     *
func (s *HookService) addAccountHook(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var account *nested.Account
	var url, hookName string
	var eventType int

	if account = s.Worker().Argument().GetAccount(request, response); account == nil {
		return
	}

	if account.ID != requester.ID && !account.Authority.Admin {
		response.Error(global.ErrAccess, []string{})
		return
	}

	if v, ok := request.Data["url"].(string); ok {
		url = v
	} else {
		response.Error(global.ErrIncomplete, []string{"url"})
		return
	}
	if v, ok := request.Data["hook_name"].(string); ok {
		if len(v) == 0 {
			response.Error(global.ErrInvalid, []string{"event_type"})
			return
		}
		hookName = v
	}
	if v, ok := request.Data["event_type"].(float64); ok {
		eventType = int(v)
	} else if v, ok := request.Data["event_type"].(string); ok {
		eventType, _ = strconv.Atoi(v)
	} else {
		response.Error(global.ErrIncomplete, []string{"event_type"})
		return
	}

	if s.Worker().Model().Hook.AddHook(requester.ID, hookName, account.ID, eventType, url) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{"internal_error"})
	}
}

// @Command:	hook/remove
// @Input: hook_id          string    *
func (s *HookService) removeHook(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var hookID bson.ObjectId
	if v, ok := request.Data["hook_id"].(string); ok {
		if !bson.IsObjectIdHex(v) {
			response.Error(global.ErrInvalid, []string{"hook_id"})
			return
		}
		hookID = bson.ObjectIdHex(v)
	} else {
		response.Error(global.ErrIncomplete, []string{"hook_id"})
		return
	}
	if s.Worker().Model().Hook.RemoveHook(hookID) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{"internal_error"})
		return
	}
}

// @Command:    hook/list
func (s *HookService) list(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	hooks := s.Worker().Model().Hook.GetHooksBySetterID(
		requester.ID,
		s.Worker().Argument().GetPagination(request),
	)
	response.OkWithData(tools.M{"hooks": hooks})
}
