package nestedServiceSession

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"strings"

	"git.ronaksoft.com/nested/server/nested"
	"github.com/globalsign/mgo/bson"
)

// @Command:	session/close
// @CommandInfo:	terminates the current session.
func (s *SessionService) close(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	session := s.Worker().Model().Session.GetByID(request.SessionKey)
	if s == nil {
		response.Error(global.ErrInvalid, []string{"_sk"})
		return
	}
	if request.SessionKey != session.ID || request.SessionSec != session.SessionSecret {
		response.Error(global.ErrAccess, []string{"_sk", "_ss"})
		return
	}
	s.Worker().Model().Session.Expire(request.SessionKey)
	if session.DeviceID != "" {
		s.Worker().Pusher().UnregisterDevice(session.DeviceID, session.DeviceToken, session.AccountID)
	}
	response.Ok()

	return
}

// @Command:	session/recall
// @CommandInfo:	recall the session on new connections. clients must call this function when they reconnect to
// @CommandInfo:	websocket server, otherwise they will not receive push notifications
// @Input:	_sk			string		*	(session key)
// @Input:	_ss			string		*	(session secret)
// @Input:	_did			string		+
// @Input:	_dt			string		+
// @Input:	_os			string		+
func (s *SessionService) recall(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var sk bson.ObjectId
	var ss, did, dt, os string
	if v, ok := request.Data["_sk"].(string); ok {
		if bson.IsObjectIdHex(v) {
			sk = bson.ObjectIdHex(v)
		} else {
			response.Error(global.ErrInvalid, []string{"_sk"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"_sk"})
		return
	}
	if v, ok := request.Data["_ss"].(string); ok {
		ss = v
	} else {
		response.Error(global.ErrIncomplete, []string{"_ss"})
		return
	}
	if v, ok := request.Data["_did"].(string); ok {
		did = v
	}
	if v, ok := request.Data["_dt"].(string); ok {
		dt = v
	}
	if v, ok := request.Data["_os"].(string); ok {
		os = strings.ToLower(v)
	}

	// If session key and secret do not match expire the session
	if !s.Worker().Model().Session.Verify(sk, ss) {
		s.Worker().Model().Session.Expire(sk)
		response.Error(global.ErrSession, []string{"_sk", "_ss"})
		return
	}
	session := s.Worker().Model().Session.GetByID(sk)
	s.Worker().Model().Session.UpdateLastAccess(sk)

	if session.AccountID == "" {
		s.Worker().Model().Session.Expire(sk)
		response.Error(global.ErrSession, []string{"uid"})
		return
	}

	// Count session recalls
	s.Worker().Model().Report.CountSessionRecall()

	// Register device in NTFY
	if did != "" && dt != "" && os != "" {
		s.Worker().Pusher().RegisterDevice(did, dt, os, session.AccountID)
	}

	// Update Session Document
	s.Worker().Model().Session.Set(
		sk,
		bson.M{
			"security.last_ip": request.ClientIP,
			"_cver":            request.ClientVersion,
		},
	)

	// Register websocket in NTFY
	if len(request.WebsocketID) > 0 {
		s.Worker().Pusher().RegisterWebsocket(session.AccountID, did, s.Worker().Config().GetString("BUNDLE_ID"), request.WebsocketID)
	}

	account := s.Worker().Model().Account.GetByID(
		session.AccountID,
		tools.M{
			"fname": 1, "lname": 1, "email": 1, "gender": 1, "phone": 1,
			"registered": 1, "disabled": 1, "counters": 1, "picture": 1,
			"dob": 1, "admin": 1, "flags": 1,
		},
	)
	if account == nil {
		response.Error(global.ErrUnknown, []string{})
		return
	}
	r := tools.M{
		"_sk":              sk,
		"_ss":              ss,
		"server_timestamp": nested.Timestamp(),
		"license_expired":  s.Worker().Server().GetFlags().LicenseExpired,
		"account":          s.Worker().Map().Account(*account, true),
	}
	switch os {
	case global.PlatformAndroid:
		r["update"] = tools.M{
			"os":          global.PlatformAndroid,
			"cur_version": global.AndroidCurrentSdkVersion,
			"min_version": global.AndroidMinSdkVersion,
		}
	case global.PlatformIOS:
		r["update"] = tools.M{
			"os":          global.PlatformIOS,
			"cur_version": global.IosCurrentSdkVersion,
			"min_version": global.IosMinSdkVersion,
		}
	}
	response.OkWithData(r)
	return
}

// @Command:	session/register
// @Input:	uid		string	*
// @Input:	pass		string	*
// @Input:	_did		string	+
// @Input:	_dt		string	+
// @Input:	_os		string	+
func (s *SessionService) register(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var uid, pass, did, dt, os string
	if v, ok := request.Data["uid"].(string); ok {
		uid = strings.ToLower(v)
	} else {
		response.Error(global.ErrIncomplete, []string{"uid"})
		return
	}
	if v, ok := request.Data["pass"].(string); ok {
		pass = v
	} else {
		response.Error(global.ErrIncomplete, []string{"pass"})
		return
	}
	if v, ok := request.Data["_did"].(string); ok {
		did = v
	}
	if v, ok := request.Data["_dt"].(string); ok {
		dt = v
	}
	if v, ok := request.Data["_os"].(string); ok {
		os = strings.ToLower(v)
	}

	// if user already logged in close the session before creating new session
	if requester != nil {
		sessionKey := request.SessionKey
		if sessionKey.Valid() {
			s.Worker().Model().Session.Expire(request.SessionKey)
		}
	}

	account := s.Worker().Model().Account.GetByID(uid, nil)
	if account == nil {
		response.Error(global.ErrInvalid, []string{"uid", "pass"})
		return
	}

	if account.Disabled {
		response.Error(global.ErrAccess, []string{"disabled"})
		return
	}

	// verify if uid & pass are matched and found in our database
	if !s.Worker().Model().Account.Verify(uid, pass) {
		response.Error(global.ErrInvalid, []string{"uid", "pass"})
		return
	}

	// increase the number of logins for the user account
	s.Worker().Model().Account.IncreaseLogins(uid)

	// Init the new session
	info := nested.MS{
		"ip": request.ClientIP,
		"ua": request.UserAgent,
	}
	sk, err := s.Worker().Model().Session.Create(info)
	if err != nil {
		response.Error(global.ErrUnknown, []string{"_sk"})
		return
	}
	ss := nested.RandomID(64)
	session := s.Worker().Model().Session.GetByID(sk)
	session.DeviceID = did
	session.DeviceOS = os
	session.SessionSecret = ss
	session.AccountID = uid
	session.DeviceToken = dt
	session.ClientID = request.ClientID
	session.ClientVersion = request.ClientVersion
	session.Login()

	// Register device in Pusher
	if did != "" && dt != "" && os != "" {
		_ = s.Worker().Pusher().RegisterDevice(did, dt, os, uid)
	}

	// Register websocket in Pusher
	if len(request.WebsocketID) > 0 {
		_ = s.Worker().Pusher().RegisterWebsocket(uid, did, s.Worker().Config().GetString("BUNDLE_ID"), request.WebsocketID)
	}

	r := tools.M{
		"_sk":              sk,
		"_ss":              ss,
		"server_timestamp": nested.Timestamp(),
		"license_expired":  s.Worker().Server().GetFlags().LicenseExpired,
		"account":          s.Worker().Map().Account(*account, true),
	}
	switch os {
	case global.PlatformAndroid:
		r["update"] = tools.M{
			"os":          global.PlatformAndroid,
			"cur_version": global.AndroidCurrentSdkVersion,
			"min_version": global.AndroidMinSdkVersion,
			"description": "",
		}
	case global.PlatformIOS:
		r["update"] = tools.M{
			"os":          global.PlatformIOS,
			"cur_version": global.IosCurrentSdkVersion,
			"min_version": global.IosMinSdkVersion,
			"description": "",
		}
	}
	// Notification Handling
	s.Worker().Pusher().NewSession(session.AccountID, request.ClientID)

	response.OkWithData(r)
}

// @Command:	session/get_actives
func (s *SessionService) getAllActives(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	sessions := s.Worker().Model().Session.GetByUser(requester.ID, s.Worker().Argument().GetPagination(request))
	r := make([]tools.M, 0, len(sessions))
	for _, s := range sessions {
		r = append(r, tools.M{
			"_sk":         s.ID,
			"ua":          s.Security.UserAgent,
			"creation_ip": s.Security.CreatorIP,
			"last_ip":     s.Security.LastIP,
			"last_access": s.LastAccess,
			"_cid":        s.ClientID,
			"_cver":       s.ClientVersion,
		})
	}
	response.OkWithData(tools.M{"sessions": r})
	return
}

// @Command:	session/close_active
// @Input:	_sk		string		*	(session key)
func (s *SessionService) closeActive(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var sessionKey bson.ObjectId
	if v, ok := request.Data["_sk"].(string); ok {
		if bson.IsObjectIdHex(v) {
			sessionKey = bson.ObjectIdHex(v)
		} else {
			response.Error(global.ErrInvalid, []string{"_sk"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"_sk"})
		return
	}
	// if session was not found return error
	session := s.Worker().Model().Session.GetByID(sessionKey)
	if session == nil {
		response.Error(global.ErrInvalid, []string{"_sk"})
		return
	}

	// users can only close their own sessions
	if session.AccountID != requester.ID {
		response.Error(global.ErrAccess, []string{})
		return
	}
	// users cannot close their current session using this function
	if session.ID == request.SessionKey {
		response.Error(global.ErrAccess, []string{"current session"})
		return
	}
	s.Worker().Model().Session.Expire(sessionKey)
	response.Ok()
}

// @Command: session/close_all_actives
func (s *SessionService) closeAllActives(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	session := s.Worker().Model().Session.GetByID(request.SessionKey)
	if session == nil {
		response.Error(global.ErrUnknown, []string{})
		return
	}
	if nested.Timestamp()-session.CreatedOn < 3600000 {
		response.Error(global.ErrAccess, []string{"session_just_created"})
		return
	}
	session.CloseOtherActives()
	response.Ok()
}
