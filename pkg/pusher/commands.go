package pusher

/*
   Creation Time: 2021 - Aug - 04
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type CMDRegisterWebsocket struct {
	UserID      string `json:"uid"`
	WebsocketID string `json:"ws_id"`
	BundleID    string `json:"bundle_id"`
	DeviceID    string `json:"_did"`
}

type CMDRegisterDevice struct {
	DeviceID    string `json:"_did"`
	UserID      string `json:"uid"`
	DeviceToken string `json:"_dt"`
	DeviceOS    string `json:"_os"`
}

type CMDUnRegisterDevice struct {
	DeviceID    string `json:"_did"`
	DeviceToken string `json:"_dt"`
	UserID      string `json:"uid"`
}

type CMDUnRegisterWebsocket struct {
	WebsocketID string `json:"ws_id"`
	BundleID    string `json:"bundle_id"`
}

type CMDPushInternal struct {
	Targets   []string `json:"targets"`
	LocalOnly bool     `json:"local_only"`
	Message   string   `json:"msg"`
}

type CMDPushExternal struct {
	Targets []string          `json:"targets"`
	Data    map[string]string `json:"data"`
}
