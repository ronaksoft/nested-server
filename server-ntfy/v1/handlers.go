package v1

import (
    "git.ronaksoftware.com/nested/server/server-ntfy/client"
    "encoding/json"
    "fmt"
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/ronak/toolbox/rpc"
    "context"
    "firebase.google.com/go/messaging"
)

func registerDevice(in rpc.Message) rpc.Message {
    req := new(ntfy.CMDRegisterDevice)

    _funcName := "RegisterDevice"
    _Log.Debug(_funcName, "started")
    defer _Log.Debug(_funcName, "exit", req.DeviceID, req.UserID)

    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Error(_funcName, err.Error())
        return ResultErr()
    }

    _Log.Debug(_funcName, req.DeviceID, req.UserID)

    if !_Model.Device.Update(req.DeviceID, req.DeviceToken, req.DeviceOS, req.UserID) {
        if !_Model.Device.Register(req.DeviceID, req.DeviceToken, req.DeviceOS, req.UserID) {
            _Log.Error(_funcName, "Did not success")
        }
    }

    return ResultOk()
}
func unregisterDevice(in rpc.Message) rpc.Message {
    req := new(ntfy.CMDUnRegisterDevice)

    _funcName := "UnregisterDevice"
    _Log.Debug(_funcName, "started")
    defer _Log.Debug(_funcName, "exit", req.DeviceID, req.UserID)

    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Error(_funcName, err.Error())
        return ResultErr()
    }

    if !_Model.Device.Remove(req.DeviceID) {
        _Log.Error(_funcName, "Did not success")
    }

    return ResultOk()
}
func registerWebsocket(in rpc.Message) rpc.Message {
    req := new(ntfy.CMDRegisterWebsocket)

    _funcName := "registerWebsocket"
    _Log.Debug(_funcName, "started")
    defer _Log.Debug(_funcName, "exit", req.DeviceID, req.UserID)

    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Error(_funcName, err.Error())
        return ResultErr()
    }

    // register websocket
    _Model.Websocket.Register(req.WebsocketID, req.BundleID, req.DeviceID, req.UserID)

    // Set device as connected and update the badges
    _Model.Device.SetAsConnected(req.DeviceID, req.UserID)

    return ResultOk()
}
func unregisterWebsocket(in rpc.Message) rpc.Message {
    req := new(ntfy.CMDUnRegisterWebsocket)

    _funcName := "unregisterWebsocket"
    _Log.Debug(_funcName, "started")
    defer _Log.Debug(_funcName, "exit", req.BundleID, req.WebsocketID)

    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Error(_funcName, err.Error())
        return ResultErr()
    }

    // Remove websocket object and set device as disconnected
    ws := _Model.Websocket.Remove(req.WebsocketID, req.BundleID)
    if ws != nil {
        _Model.Device.SetAsDisconnected(ws.DeviceID)
    }

    return ResultOk()
}
func pushInternal(in rpc.Message) rpc.Message {
    req := new(ntfy.CMDPushInternal)

    _funcName := "pushInternal"
    _Log.Debug(_funcName, "started")
    defer _Log.Debug(_funcName, "exit", req.LocalOnly)

    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Error(_funcName, err.Error())
        return ResultErr()
    }

    if req.LocalOnly {
        for _, uid := range req.Targets {
            websockets := _Model.Websocket.GetWebsocketsByAccountID(uid, _BundleID)
            for _, ws := range websockets {
                b, _ := json.Marshal(ntfy.WebsocketPush{
                    WebsocketID: ws.WebsocketID,
                    Payload:     req.Message,
                    BundleID:    ws.BundleID})

                if err := _NatsConn.Publish("GATEWAY", b); err != nil {
                    _Log.Error(_funcName, err.Error())
                }
            }
        }
    } else {
        for _, uid := range req.Targets {
            websockets := _Model.Websocket.GetWebsocketsByAccountID(uid, "")
            for _, ws := range websockets {
                b, _ := json.Marshal(ntfy.WebsocketPush{
                    WebsocketID: ws.WebsocketID,
                    Payload:     req.Message,
                    BundleID:    ws.BundleID})
                if _BundleID != ws.BundleID {
                    if err := _NatsConn.Publish(fmt.Sprintf("ROUTER.%s.GATEWAY", ws.BundleID), b); err != nil {
                        _Log.Error(_funcName, err.Error())
                    }
                } else {
                    if err := _NatsConn.Publish("GATEWAY", b); err != nil {
                        _Log.Error(_funcName, err.Error())
                    }
                }
            }
        }
    }

    return ResultOk()
}
func pushExternal(in rpc.Message) rpc.Message {
    _funcName := "pushExternal"
    _Log.Debug(_funcName, "started")
    defer _Log.Debug(_funcName, "exit")

    req := new(ntfy.CMDPushExternal)
    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Error(_funcName, err.Error())
        return ResultErr()
    }
    _Log.Debug(_funcName, "", req.Targets, req.Data)
    for _, uid := range req.Targets {
        go func(uid string) {
            _Model.Device.IncrementBadge(uid)
            devices := _Model.Device.GetByAccountID(uid)
            for _, d := range devices {
                FCM(d, *req)
                // switch d.OS {
                // case  nested.PLATFORM_IOS, nested.PLATFORM_SAFARI:
                //     apn_push_notification(d, *req)
                // case nested.PLATFORM_ANDROID, nested.PLATFORM_CHROME, nested.PLATFORM_FIREFOX, nested:
                //
                // }
            }
        }(uid)

    }

    return ResultOk()
}

func FCM(d nested.Device, req ntfy.CMDPushExternal) {
    _funcName := "FCM"
    _Log.Debug(_funcName, "started")
    defer _Log.Debug(_funcName, "exit")
    message := messaging.Message{
        Data: req.Data,
        Token: d.Token,
        Android: &messaging.AndroidConfig{
            Priority: "high",
            // Notification: &messaging.AndroidNotification{
            //     Title: req.Data["title"],
            //     Body: req.Data["msg"],
            // },
            Data: req.Data,
        },
        APNS: &messaging.APNSConfig{
            Payload: &messaging.APNSPayload{
                Aps: &messaging.Aps{
                    Alert: &messaging.ApsAlert{
                        Title: req.Data["title"],
                        Body: req.Data["msg"],
                    },
                    Badge: &d.Badge,
                    CustomData: make(map[string]interface{}),
                },
            },
        },
    }
    for k, v := range req.Data {
        message.APNS.Payload.Aps.CustomData[k] = v
    }


    ctx := context.Background()
    if client, err := _FCM.Messaging(ctx); err != nil {
        _Log.Error(_funcName, err.Error())
    } else if resp, err := client.Send(ctx, &message); err != nil {
        _Log.Error(_funcName, err.Error())
        _Model.Device.Remove(d.ID)
    } else {
        _Log.Debug(_funcName, resp)
    }
}