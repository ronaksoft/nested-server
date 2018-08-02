package v1

import (
    "git.ronaksoftware.com/ronak/toolbox/rpc"
    "git.ronaksoftware.com/ronak/toolbox/logger"
    "git.ronaksoftware.com/nested/server-model-nested"
    "github.com/nats-io/go-nats"
    "firebase.google.com/go"
    "git.ronaksoftware.com/ronak/toolbox"
)

var (
    _Log      *log.Logger
    _Model    *nested.Manager
    _BundleID string
    _FCM      *firebase.App
    _NatsConn *nats.Conn
)

func init() {
    _Log = log.NewTerminalLogger(log.LEVEL_DEBUG)
}

func NewWorker(rateLimiter ronak.RateLimiter, model *nested.Manager, natsConn *nats.Conn,  fcmClient *firebase.App,
    bundleID string) *rpc.SimpleWorker {
    worker := rpc.NewSimpleRPCWorker(rateLimiter)
    worker.AddHandler("NTFY.REGISTER.DEVICE", registerDevice)
    worker.AddHandler("NTFY.UNREGISTER.DEVICE", unregisterDevice)
    worker.AddHandler("NTFY.REGISTER.WEBSOCKET", registerWebsocket)
    worker.AddHandler("NTFY.UNREGISTER.WEBSOCKET", unregisterWebsocket)
    worker.AddHandler("NTFY.PUSH.INTERNAL", pushInternal)
    worker.AddHandler("NTFY.PUSH.EXTERNAL", pushExternal)

    _Model = model
    _BundleID = bundleID
    _NatsConn = natsConn
    _FCM = fcmClient



    return worker
}

func ResultOk() rpc.Message {
    return rpc.Message{
        Constructor: "OK",
    }
}

func ResultErr() rpc.Message {
    return rpc.Message{
        Constructor: "ERR",
    }
}
