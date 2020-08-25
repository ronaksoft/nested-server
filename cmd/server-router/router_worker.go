package main

import (
	"bytes"
	"errors"
	"fmt"
	router "git.ronaksoft.com/nested/server/cmd/server-router/client"
	"go.uber.org/zap"
	"strings"
	"time"

	"github.com/nats-io/go-nats"
)

type RouterWorker struct {
	jh *JobHandler

	BundleGroup string
	BundleIndex string

	subjInt              string
	subjIntPrefix        string
	subjExtAnycast       string
	subjExtAnycastPrefix string
	subjExtUnicast       string
	subjExtUnicastPrefix string

	chIntToExt chan *nats.Msg
	chExtToInt chan *nats.Msg
}

func NewRouterWorker(jh *JobHandler) (*RouterWorker, error) {
	rw := new(RouterWorker)
	rw.jh = jh

	rw.chIntToExt = make(chan *nats.Msg, jh.conf.GetInt("JOB_INT_BUFFER_SIZE"))
	rw.chExtToInt = make(chan *nats.Msg, jh.conf.GetInt("JOB_EXT_BUFFER_SIZE"))

	_Log.Info("Internal Job Buffer Size",
		zap.Int("BufferSize", jh.conf.GetInt("JOB_INT_BUFFER_SIZE")),
	)
	_Log.Info("External Job Buffer Size",
		zap.Int("BufferSize", jh.conf.GetInt("JOB_EXT_BUFFER_SIZE")),
	)

	var subjInt bytes.Buffer
	subjInt.WriteString(router.ROUTER_SUBJECT_PREFIX)
	subjInt.WriteRune('.')
	rw.subjIntPrefix = subjInt.String()
	subjInt.WriteRune('>')
	rw.subjInt = subjInt.String()

	bundleId := jh.conf.GetString("BUNDLE_ID")
	if s := strings.SplitN(bundleId, "-", 2); len(s) != 2 {
		return nil, errors.New(fmt.Sprintf("Invalid Bundle ID. Expected <BUNDLE GROUP>-<BUNDLE INDEX> Got: %s", bundleId))
	} else {
		rw.BundleGroup = strings.ToUpper(s[0])
		rw.BundleIndex = strings.ToUpper(s[1])
	}

	var subjUniCast bytes.Buffer
	subjUniCast.WriteString(rw.BundleGroup)
	subjUniCast.WriteRune('-')
	subjUniCast.WriteString(rw.BundleIndex)
	subjUniCast.WriteRune('.')
	rw.subjExtUnicastPrefix = subjUniCast.String()
	subjUniCast.WriteRune('>')
	rw.subjExtUnicast = subjUniCast.String()

	var subjAnyCast bytes.Buffer
	subjAnyCast.WriteString(rw.BundleGroup)
	subjAnyCast.WriteRune('.')
	rw.subjExtAnycastPrefix = subjAnyCast.String()
	subjAnyCast.WriteRune('>')
	rw.subjExtAnycast = subjAnyCast.String()

	return rw, nil
}

func (rw *RouterWorker) RegisterWorker() error {
	for i := 0; i < rw.jh.conf.GetInt("JOB_INT_WORKERS_COUNT"); i++ {
		go func() {
			for {
				msg := <-rw.chIntToExt
				rw.toExtern(msg)
			}
		}()
	}

	for i := 0; i < rw.jh.conf.GetInt("JOB_EXT_WORKERS_COUNT"); i++ {
		go func() {
			for {
				msg := <-rw.chExtToInt
				rw.toIntern(msg)
			}
		}()
	}

	if _, err := rw.jh.iconn.ChanSubscribe(rw.subjInt, rw.chIntToExt); err != nil {
		return err
	}

	if _, err := rw.jh.xconn.ChanSubscribe(rw.subjExtUnicast, rw.chExtToInt); err != nil {
		return err
	}

	if _, err := rw.jh.xconn.ChanSubscribe(rw.subjExtAnycast, rw.chExtToInt); err != nil {
		return err
	}

	return nil
}

func (rw *RouterWorker) toExtern(msg *nats.Msg) {
	if 0 != strings.Index(msg.Subject, rw.subjIntPrefix) {
		return
	}

	xsubj := msg.Subject[len(rw.subjIntPrefix):]
	if response, err := rw.jh.xconn.Request(xsubj, msg.Data, time.Second*20); err != nil {
		rw.jh.iconn.PublishMsg(response)
	} else {
		rw.jh.iconn.PublishMsg(response)
	}
}

func (rw *RouterWorker) toIntern(msg *nats.Msg) {
	var isubj string
	if 0 == strings.Index(msg.Subject, rw.subjExtUnicastPrefix) {
		isubj = msg.Subject[len(rw.subjExtUnicastPrefix):]
	} else if 0 == strings.Index(msg.Subject, rw.subjExtAnycastPrefix) {
		isubj = msg.Subject[len(rw.subjExtAnycastPrefix):]
	} else {
		return
	}

	if response, err := rw.jh.iconn.Request(isubj, msg.Data, time.Second*20); err != nil {
		rw.jh.xconn.PublishMsg(response)
	} else {
		rw.jh.xconn.PublishMsg(response)
	}
}
