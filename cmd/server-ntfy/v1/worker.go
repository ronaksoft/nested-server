package v1

import (
	"firebase.google.com/go"
	"git.ronaksoft.com/nested/server/model"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"github.com/nats-io/go-nats"
)

var (
	_Model    *nested.Manager
	_BundleID string
	_FCM      *firebase.App
	_NatsConn *nats.Conn
)

func NewWorker(rateLimiter tools.RateLimiter, model *nested.Manager, natsConn *nats.Conn, fcmClient *firebase.App,
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
