package nestedServiceSession

import (
    "git.ronaksoftware.com/nested/server/model"
    "github.com/globalsign/mgo/bson"
    "log"
    "strings"
    "git.ronaksoftware.com/nested/server-gateway/client"
)

// @Command:	session/close
// @CommandInfo:	terminates the current session.
func (s *SessionService) close(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    session := s.Worker().Model().Session.GetByID(request.SessionKey)
    if s == nil {
        response.Error(nested.ERR_INVALID, []string{"_sk"})
        return
    }
    if request.SessionKey != session.ID || request.SessionSec != session.SessionSecret {
        response.Error(nested.ERR_ACCESS, []string{"_sk", "_ss"})
        return
    }
    log.Println("Session Expired because of CloseSession::", request.SessionKey.Hex())
    s.Worker().Model().Session.Expire(request.SessionKey)
    if session.DeviceID != "" {
        s.Worker().Pusher().Notification.UnregisterDevice(session.DeviceID, session.DeviceToken, session.AccountID)
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
func (s *SessionService) recall(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var sk bson.ObjectId
    var ss, did, dt, os string
    if v, ok := request.Data["_sk"].(string); ok {
        if bson.IsObjectIdHex(v) {
            sk = bson.ObjectIdHex(v)
        } else {
            response.Error(nested.ERR_INVALID, []string{"_sk"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"_sk"})
        return
    }
    if v, ok := request.Data["_ss"].(string); ok {
        ss = v
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"_ss"})
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
        response.Error(nested.ERR_SESSION, []string{"_sk", "_ss"})
        return
    }
    session := s.Worker().Model().Session.GetByID(sk)
    s.Worker().Model().Session.UpdateLastAccess(sk)

    if session.AccountID == "" {
        log.Println("SESSION::EXPIRED::ACCOUNT NOT EXIST IN SESSION", request.SessionKey)
        s.Worker().Model().Session.Expire(sk)
        response.Error(nested.ERR_SESSION, []string{"uid"})
        return
    }

    // Count session recalls
    s.Worker().Model().Report.CountSessionRecall()

    // Register device in NTFY
    if did != "" && dt != "" && os != "" {
        s.Worker().Pusher().Notification.RegisterDevice(did, dt, os, session.AccountID)
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
        s.Worker().Pusher().Notification.RegisterWebsocket(session.AccountID, did, s.Worker().Config().GetString("BUNDLE_ID"), request.WebsocketID)
    }

    account := s.Worker().Model().Account.GetByID(
        session.AccountID,
        nested.M{
            "fname":      1, "lname": 1, "email": 1, "gender": 1, "phone": 1,
            "registered": 1, "disabled": 1, "counters": 1, "picture": 1,
            "dob":        1, "admin": 1, "flags": 1,
        },
    )
    if account == nil {
        log.Println("USERAPI::RecallSession::Account is nil::", sk.Hex())
        response.Error(nested.ERR_UNKNOWN, []string{})
        return
    }
    r := nested.M{
        "_sk":              sk,
        "_ss":              ss,
        "server_timestamp": nested.Timestamp(),
        "license_expired":  s.Worker().Server().GetFlags().LicenseExpired,
        "account":          s.Worker().Map().Account(*account, true),
    }
    switch os {
    case nested.PLATFORM_ANDROID:
        r["update"] = nested.M{
            "os":          nested.PLATFORM_ANDROID,
            "cur_version": nested.ANDROID_CURRENT_SDK_VERSION,
            "min_version": nested.ANDROID_MIN_SDK_VERSION,
        }
    case nested.PLATFORM_IOS:
        r["update"] = nested.M{
            "os":          nested.PLATFORM_IOS,
            "cur_version": nested.IOS_CURRENT_SDK_VERSION,
            "min_version": nested.IOS_MIN_SDK_VERSION,
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
func (s *SessionService) register(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var uid, pass, did, dt, os string
    if v, ok := request.Data["uid"].(string); ok {
        uid = strings.ToLower(v)
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"uid"})
        return
    }
    if v, ok := request.Data["pass"].(string); ok {
        pass = v
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"pass"})
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
        log.Println("SESSION::EXPIRED::ALREADY_LOGGED_IN", uid, sessionKey)
        if sessionKey.Valid() {
            s.Worker().Model().Session.Expire(request.SessionKey)
        }
    }

    account := s.Worker().Model().Account.GetByID(uid, nil)
    if account == nil {
        response.Error(nested.ERR_INVALID, []string{"uid", "pass"})
        return
    }

    if account.Disabled {
        response.Error(nested.ERR_ACCESS, []string{"disabled"})
        return
    }

    // verify if uid & pass are matched and found in our database
    if !s.Worker().Model().Account.Verify(uid, pass) {
        response.Error(nested.ERR_INVALID, []string{"uid", "pass"})
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
        response.Error(nested.ERR_UNKNOWN, []string{"_sk"})
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

    // Register device in NTFY
    if did != "" && dt != "" && os != "" {
        s.Worker().Pusher().Notification.RegisterDevice(did, dt, os, uid)
    }

    // Register websocket in NTFY
    if len(request.WebsocketID) > 0 {
        s.Worker().Pusher().Notification.RegisterWebsocket(uid, did, s.Worker().Config().GetString("BUNDLE_ID"), request.WebsocketID)
    }

    r := nested.M{
        "_sk":              sk,
        "_ss":              ss,
        "server_timestamp": nested.Timestamp(),
        "license_expired":  s.Worker().Server().GetFlags().LicenseExpired,
        "account":          s.Worker().Map().Account(*account, true),
    }
    switch os {
    case nested.PLATFORM_ANDROID:
        r["update"] = nested.M{
            "os":          nested.PLATFORM_ANDROID,
            "cur_version": nested.ANDROID_CURRENT_SDK_VERSION,
            "min_version": nested.ANDROID_MIN_SDK_VERSION,
            "description": "",
        }
    case nested.PLATFORM_IOS:
        r["update"] = nested.M{
            "os":          nested.PLATFORM_IOS,
            "cur_version": nested.IOS_CURRENT_SDK_VERSION,
            "min_version": nested.IOS_MIN_SDK_VERSION,
            "description": "",
        }
    }
    // Notification Handling
    s.Worker().Pusher().NewSession(session.AccountID, request.ClientID)

    response.OkWithData(r)
}

// @Command:	session/get_actives
func (s *SessionService) getAllActives(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    sessions := s.Worker().Model().Session.GetByUser(requester.ID, s.Worker().Argument().GetPagination(request))
    r := make([]nested.M, 0, len(sessions))
    for _, s := range sessions {
        r = append(r, nested.M{
            "_sk":         s.ID,
            "ua":          s.Security.UserAgent,
            "creation_ip": s.Security.CreatorIP,
            "last_ip":     s.Security.LastIP,
            "last_access": s.LastAccess,
            "_cid":        s.ClientID,
            "_cver":       s.ClientVersion,
        })
    }
    response.OkWithData(nested.M{"sessions": r})
    return
}

// @Command:	session/close_active
// @Input:	_sk		string		*	(session key)
func (s *SessionService) closeActive(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var sessionKey bson.ObjectId
    if v, ok := request.Data["_sk"].(string); ok {
        if bson.IsObjectIdHex(v) {
            sessionKey = bson.ObjectIdHex(v)
        } else {
            response.Error(nested.ERR_INVALID, []string{"_sk"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"_sk"})
        return
    }
    // if session was not found return error
    session := s.Worker().Model().Session.GetByID(sessionKey)
    if session == nil {
        response.Error(nested.ERR_INVALID, []string{"_sk"})
        return
    }

    // users can only close their own sessions
    if session.AccountID != requester.ID {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    // users cannot close their current session using this function
    if session.ID == request.SessionKey {
        response.Error(nested.ERR_ACCESS, []string{"current session"})
        return
    }

    log.Println("Session Expired because of CloseActiveSession::", sessionKey.Hex())
    s.Worker().Model().Session.Expire(sessionKey)
    response.Ok()
}

// @Command: session/close_all_actives
func (s *SessionService) closeAllActives(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    session := s.Worker().Model().Session.GetByID(request.SessionKey)
    if session == nil {
        response.Error(nested.ERR_UNKNOWN, []string{})
        return
    }
    if nested.Timestamp()-session.CreatedOn < 3600000 {
        response.Error(nested.ERR_ACCESS, []string{"session_just_created"})
        return
    }
    session.CloseOtherActives()
    response.Ok()
}
