package v1

import (
    "os"

    "firebase.google.com/go"
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/ronak/toolbox"
    "git.ronaksoftware.com/ronak/toolbox/rpc"
    "github.com/nats-io/go-nats"
    "go.uber.org/zap"
)

var (
    _Log      *zap.Logger
    _LogLevel zap.AtomicLevel
    _Model    *nested.Manager
    _BundleID string
    _FCM      *firebase.App
    _NatsConn *nats.Conn
)

func init() {
    _LogLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
    zap.NewProductionConfig()
    logConfig := zap.NewProductionConfig()
    logConfig.Encoding = "console"
    logConfig.Level = _LogLevel
    if v, err := logConfig.Build(); err != nil {
        os.Exit(1)
    } else {
        _Log = v
    }
}

func NewWorker(rateLimiter ronak.RateLimiter, model *nested.Manager, natsConn *nats.Conn, fcmClient *firebase.App,
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
