package nestedServiceApp

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"strings"

	"git.ronaksoft.com/nested/server/nested"
)

// @Command: app/exists
// @Input:  app_id          string  *
func (s *AppService) exists(_ *nested.Account, request *rpc.Request, response *rpc.Response) {
	var appID string
	if v, ok := request.Data["app_id"].(string); ok {
		appID = strings.TrimSpace(v)
	}

	app := s.Worker().Model().App.GetByID(appID)
	if app == nil {
		response.Error(global.ErrUnavailable, []string{"app_id"})
		return
	}
	response.OkWithData(tools.M{"exists": true})
	return
}

// @Command: app/create_token
// @Input:  app_id          string  *
func (s *AppService) generateAppToken(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var appID string
	if v, ok := request.Data["app_id"].(string); ok {
		appID = strings.TrimSpace(v)
	}

	app := s.Worker().Model().App.GetByID(appID)
	if app == nil {
		response.Error(global.ErrInvalid, []string{"app_id"})
		return
	}
	appToken := s.Worker().Model().Token.CreateAppToken(
		requester.ID,
		appID,
	)

	response.OkWithData(tools.M{
		"token": appToken,
	})
}

// @Command: app/revoke_token
// @Input:  token        string  *
func (s *AppService) revokeAppToken(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var appToken string
	if v, ok := request.Data["token"].(string); ok {
		appToken = v
	}
	if s.Worker().Model().Token.RevokeAppToken(requester.ID, appToken) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{"internal_error"})
	}
}

// @Command: app/get_tokens
func (s *AppService) getTokensByAccountID(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	appTokens := s.Worker().Model().Token.GetAppTokenByAccountID(requester.ID, s.Worker().Argument().GetPagination(request))
	r := make([]tools.M, 0, len(appTokens))
	for _, appToken := range appTokens {
		r = append(r, s.Worker().Map().AppToken(appToken))
	}
	response.OkWithData(tools.M{"app_tokens": r})
	return
}

// @Command: app/get_app
func (s *AppService) getTokenByAppID(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	// TODO:: implement it
}

// @Command: app/get_many
// @Input: app_id       string      *   (comma separated)
func (s *AppService) getManyApps(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var appIDs []string
	if v, ok := request.Data["app_id"].(string); ok {
		appIDs = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	} else {
		response.Error(global.ErrIncomplete, []string{"app_id"})
		return
	}
	apps := s.Worker().Model().App.GetManyByIDs(appIDs)
	response.OkWithData(tools.M{"apps": apps})
}

// @Command: app/register
// @Input:  app_id          string      *
// @Input:  app_name        string      *
// @Input:  homepage        string      *
// @Input:  developer       string      *
// @Input:  icon_large_url  string      +
// @Input:  icon_small_url  string      +
func (s *AppService) register(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var appID, appName, homepage, developer, iconLargeUrl, iconSmallUrl, callbackUrl string
	if v, ok := request.Data["app_id"].(string); ok {
		appID = v
	}
	if v, ok := request.Data["app_name"].(string); ok {
		appName = strings.TrimSpace(v)
	}
	if v, ok := request.Data["homepage"].(string); ok {
		homepage = v
	}
	if v, ok := request.Data["developer"].(string); ok {
		developer = v
	}
	if v, ok := request.Data["icon_large_url"].(string); ok {
		iconLargeUrl = v
	}
	if v, ok := request.Data["icon_small_url"].(string); ok {
		iconSmallUrl = v
	}
	if v, ok := request.Data["callback_url"].(string); ok {
		callbackUrl = v
	}

	if s.Worker().Model().App.Exists(appID) {
		response.Error(global.ErrDuplicate, []string{"app_id"})
		return
	}
	if s.Worker().Model().App.Register(appID, appName, homepage, callbackUrl, developer, iconSmallUrl, iconLargeUrl) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{"internal_error"})
	}
}

// @Command: app/remove
// @Input:  app_id      string      *
func (s *AppService) remove(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var appID string
	if v, ok := request.Data["app_id"].(string); ok {
		appID = v
	}
	app := s.Worker().Model().App.GetByID(appID)
	if app == nil {
		response.Error(global.ErrInvalid, []string{"app_id"})
		return
	}

	if s.Worker().Model().App.UnRegister(appID) {
		if requester.Authority.Admin {
			s.Worker().Model().Token.RemoveAppTokenForAll(appID)
		}
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{"internal_error"})
	}
}

// @Command: app/has_token
// @Input:  app_id      string      *
func (s *AppService) hasToken(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var appID string
	if v, ok := request.Data["app_id"].(string); ok {
		appID = v
	}

	if s.Worker().Model().Token.AppTokenExists(requester.ID, appID) {
		response.Ok()
	} else {
		response.Error(global.ErrInvalid, []string{})
	}
}

// @Command: app/set_fav_status
// @Input:  app_id      string      *
func (s *AppService) setFavStatus(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var status bool
	var appID string
	if v, ok := request.Data["status"].(bool); ok {
		status = v
	}
	if v, ok := request.Data["app_id"].(string); ok {
		appID = v
	}

	if s.Worker().Model().Token.SetAppFavoriteStatus(requester.ID, appID, status) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{"internal_error"})
	}
}
