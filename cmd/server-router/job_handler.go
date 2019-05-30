package main

import (
	"github.com/nats-io/go-nats"
	"go.uber.org/zap"
	"gopkg.in/fzerorubigd/onion.v3"
)

type JobHandler struct {
	conf *onion.Onion

	iconn *nats.Conn
	xconn *nats.Conn

	router *RouterWorker
}

func (jh *JobHandler) RegisterWorkers() error {
	// TODO: Add ability to disable some workers
	if err := jh.router.RegisterWorker(); err != nil {
		return err
	}

	return nil
}

func NewJobHandler(conf *onion.Onion) (*JobHandler, error) {
	jh := &JobHandler{
		conf: conf,
	}

	// Connecting to Internal NATs
	_Log.Debug("Connecting to Internal NATS",
		zap.String("Address", conf.GetString("JOB_INT_ADDRESS")),
	)
	if conn, err := nats.Connect(conf.GetString("JOB_INT_ADDRESS")); err != nil {
		_Log.Error("Unable to establish Internal NATS connection")

		return nil, err
	} else {
		jh.iconn = conn
	}

	_Log.Info("iNATS Connected")
	_Log.Debug("Connecting to External NATS",
		zap.String("Address", conf.GetString("JOB_EXT_ADDRESS")),
	)

	if conn, err := nats.Connect(conf.GetString("JOB_EXT_ADDRESS")); err != nil {
		_Log.Error("Unable to establish External NATS connection")

		return nil, err
	} else {
		jh.xconn = conn
	}

	_Log.Info("xNATS Connected")

	jh.router, _ = NewRouterWorker(jh)

	return jh, nil
}
