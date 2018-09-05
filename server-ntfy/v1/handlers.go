package v1

import (
    "context"
    "encoding/json"
    "fmt"

    "firebase.google.com/go/messaging"
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server/server-ntfy/client"
    "git.ronaksoftware.com/ronak/toolbox/rpc"
    "go.uber.org/zap"
)

func registerDevice(in rpc.Message) rpc.Message {
    req := new(ntfy.CMDRegisterDevice)

    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Error(err.Error())
        return ResultErr()
    }

    _Log.Debug("Register Device",
        zap.String("DeviceID", req.DeviceID),
        zap.String("UserID", req.UserID),
    )

    if !_Model.Device.Update(req.DeviceID, req.DeviceToken, req.DeviceOS, req.UserID) {
        if !_Model.Device.Register(req.DeviceID, req.DeviceToken, req.DeviceOS, req.UserID) {
            _Log.Warn("register device was not successful")
        }
    }

    return ResultOk()
}
func unregisterDevice(in rpc.Message) rpc.Message {
    req := new(ntfy.CMDUnRegisterDevice)

    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Warn(err.Error())
        return ResultErr()
    }

    if !_Model.Device.Remove(req.DeviceID) {
        _Log.Warn("unregister device was not successful")
    }

    return ResultOk()
}
func registerWebsocket(in rpc.Message) rpc.Message {
    req := new(ntfy.CMDRegisterWebsocket)

    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Warn(err.Error())
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

    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Warn(err.Error())
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

    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Warn(err.Error())
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
                    _Log.Warn(err.Error())
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
                        _Log.Warn(err.Error())
                    }
                } else {
                    if err := _NatsConn.Publish("GATEWAY", b); err != nil {
                        _Log.Warn(err.Error())
                    }
                }
            }
        }
    }

    return ResultOk()
}
func pushExternal(in rpc.Message) rpc.Message {
    req := new(ntfy.CMDPushExternal)
    if err := in.Data.UnMarshal(req); err != nil {
        _Log.Warn(err.Error())
        return ResultErr()
    }
    _Log.Debug("Push External",
        zap.Strings("Targets", req.Targets),
    )

    for _, uid := range req.Targets {
        go func(uid string) {
            _Model.Device.IncrementBadge(uid)
            devices := _Model.Device.GetByAccountID(uid)
            for _, d := range devices {
                FCM(d, *req)
            }
        }(uid)

    }

    return ResultOk()
}

func FCM(d nested.Device, req ntfy.CMDPushExternal) {
    message := messaging.Message{
        Data:  req.Data,
        Token: d.Token,
        Android: &messaging.AndroidConfig{
            Priority: "high",
            Data:     req.Data,
        },
        APNS: &messaging.APNSConfig{
            Payload: &messaging.APNSPayload{
                Aps: &messaging.Aps{
                    Alert: &messaging.ApsAlert{
                        Title: req.Data["title"],
                        Body:  req.Data["msg"],
                    },
                    Badge:      &d.Badge,
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
        _Log.Warn(err.Error())
    } else if _, err := client.Send(ctx, &message); err != nil {
        _Log.Warn(err.Error())
        _Model.Device.Remove(d.ID)
    }
}
