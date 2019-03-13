package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"firebase.google.com/go"
	"git.ronaksoftware.com/nested/server/cmd/server-ntfy/v1"
	"git.ronaksoftware.com/nested/server/model"
	"git.ronaksoftware.com/nested/server/pkg/rpc"
	"git.ronaksoftware.com/ronak/toolbox"
	"github.com/nats-io/go-nats"
	"google.golang.org/api/option"
	"gopkg.in/fzerorubigd/onion.v3"
)

var (
	exit_ch chan os.Signal
	_Config *onion.Onion
	_Model  *nested.Manager
)

func init() {
	exit_ch = make(chan os.Signal, 1)
	signal.Notify(exit_ch, syscall.SIGTERM)

	_Config = readConfig()

}

func main() {
	var natsConn *nats.Conn

	// initialize nested model
	if v, err := nested.NewManager(
		_Config.GetString("INSTANCE_ID"),
		_Config.GetString("MONGO_DSN"),
		_Config.GetString("REDIS_DSN"),
		_Config.GetInt("DEBUG_LEVEL"),
	); err != nil {
		log.Println("NTFY::Nested Model Initializing Error::", err.Error())
		os.Exit(1)
	} else {
		_Model = v
	}

	// Remove all the websockets on this bundle
	_Model.Websocket.RemoveByBundleID(_Config.GetString("BUNDLE_ID"))

	// Initialize NATS
	natsConfig := nats.GetDefaultOptions()
	natsConfig.Url = _Config.GetString("JOB_ADDRESS")
	natsConfig.User = _Config.GetString("JOB_USER")
	natsConfig.Password = _Config.GetString("JOB_PASS")
	if conn, err := natsConfig.Connect(); err != nil {
		log.Println("Unable to establish NATS connection:", natsConfig.Url, err.Error())
		os.Exit(1)
	} else {
		natsConn = conn
		defer natsConn.Close()
	}

	// Initialize FCM Client
	var fcmClient *firebase.App
	if c, err := firebase.NewApp(
		context.Background(),
		nil,
		option.WithCredentialsFile("/ronak/certs/firebase-cred.json"),
	); err != nil {
		log.Panic(err.Error())
	} else {
		fcmClient = c
	}
	// Initialize RPC Worker
	var rpcWorker *rpc.SimpleWorker
	if rateLimiter, err := ronak.NewSimpleRateLimiter(10*time.Second, _Config.GetInt("JOB_WORKERS_COUNT")); err != nil {
		log.Panic(err.Error())
	} else {
		rpcWorker = v1.NewWorker(
			rateLimiter, _Model, natsConn,
			fcmClient,
			_Config.GetString("BUNDLE_ID"),
		)
	}

	natsConn.Subscribe("NTFY.>", func(msg *nats.Msg) {
		in := rpc.Message{
			Constructor: rpc.MessageConstructor(msg.Subject),
			Data:        rpc.JsonMessageStream(msg.Data),
		}
		rpcWorker.Execute(in)
	})

	// Waiting for exit signal
	<-exit_ch

}
