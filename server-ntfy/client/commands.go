package ntfy

// easyjson:json
type CMDRegisterWebsocket struct {
    UserID      string `json:"uid"`
    WebsocketID string `json:"ws_id"`
    BundleID    string `json:"bundle_id"`
    DeviceID    string `json:"_did"`
}

// easyjson:json
type CMDRegisterDevice struct {
    DeviceID    string `json:"_did"`
    UserID      string `json:"uid"`
    DeviceToken string `json:"_dt"`
    DeviceOS    string `json:"_os"`
}

// easyjson:json
type CMDUnRegisterDevice struct {
    DeviceID    string `json:"_did"`
    DeviceToken string `json:"_dt"`
    UserID      string `json:"uid"`
}

// easyjson:json
type CMDUnRegisterWebsocket struct {
    WebsocketID string `json:"ws_id"`
    BundleID    string `json:"bundle_id"`
}

// easyjson:json
type CMDPushInternal struct {
    Targets   []string `json:"targets"`
    LocalOnly bool     `json:"local_only"`
    Message   string   `json:"msg"`
}

// easyjson:json
type CMDPushExternal struct {
    Targets []string          `json:"targets"`
    Data    map[string]string `json:"data"`
}
