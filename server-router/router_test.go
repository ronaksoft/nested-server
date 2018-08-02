package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/go-nats"
	"gopkg.in/fzerorubigd/onion.v2"

	"git.ronaksoftware.com/common/server-protocol"
	"git.ronaksoftware.com/nested/server-router/client"
)

type bundle struct {
	conf *onion.Onion

	group string
	index string

	jobHandler *JobHandler

	client *router.Client
}

var (
	bPouyan *bundle
	bEhsan  *bundle
)

func init() {
	b1nats := flag.String("b1nats", "nats://b1.job", "Bundle 1's Internal NATS (=nats://b1.job)")
	b2nats := flag.String("b2nats", "nats://b2.job", "Bundle 1's Internal NATS (=nats://b2.job)")
	xnats := flag.String("xnats", "nats://x.job", "External NATS (=nats://x.job)")
	flag.Parse()
	initLogger("std", 3)

	log.Println(*b1nats, *b2nats, *xnats)

	bPouyan = &bundle{
		conf:  onion.New(),
		group: "POUYAN",
		index: "0001",
	}
	dl1 := onion.NewDefaultLayer()
	dl1.SetDefault("BUNDLE_ID", fmt.Sprintf("%s-%s", bPouyan.group, bPouyan.index))
	dl1.SetDefault("JOB_INT_ADDRESS", *b1nats)
	dl1.SetDefault("JOB_INT_WORKERS_COUNT", 10)
	dl1.SetDefault("JOB_INT_BUFFER_SIZE", 100)
	dl1.SetDefault("JOB_EXT_ADDRESS", *xnats)
	dl1.SetDefault("JOB_EXT_WORKERS_COUNT", 10)
	dl1.SetDefault("JOB_EXT_BUFFER_SIZE", 100)
	bPouyan.conf.AddLayer(dl1)
	if client, err := router.NewClient(bPouyan.conf.GetString("JOB_INT_ADDRESS"), bPouyan.conf.GetString("BUNDLE_ID")); err != nil {
		log.Panicf("Unable to create router client: Address: %s, Error: %s", bPouyan.conf.GetString("JOB_INT_ADDRESS"), err)
	} else {
		bPouyan.client = client
	}

	log.Printf("Bundle 1: GROUP: %s, INDEX: %s", bPouyan.group, bPouyan.index)

	if jh, err := NewJobHandler(bPouyan.conf); err != nil {
		log.Panicln("Failed to initialize Router API Job Handler", err.Error())
	} else if err := jh.RegisterWorkers(); err != nil {
		log.Panicln("Failed to register Router API Workers", err.Error())
	} else {
		bPouyan.jobHandler = jh
	}

	bEhsan = &bundle{
		conf:  onion.New(),
		group: "EHSAN",
		index: "0001",
	}
	dl2 := onion.NewDefaultLayer()
	dl2.SetDefault("BUNDLE_ID", fmt.Sprintf("%s-%s", bEhsan.group, bEhsan.index))
	dl2.SetDefault("JOB_INT_ADDRESS", *b2nats)
	dl2.SetDefault("JOB_INT_WORKERS_COUNT", 10)
	dl2.SetDefault("JOB_INT_BUFFER_SIZE", 100)
	dl2.SetDefault("JOB_EXT_ADDRESS", *xnats)
	dl2.SetDefault("JOB_EXT_WORKERS_COUNT", 10)
	dl2.SetDefault("JOB_EXT_BUFFER_SIZE", 100)
	bEhsan.conf.AddLayer(dl2)
	if client, err := router.NewClient(bEhsan.conf.GetString("JOB_INT_ADDRESS"), bEhsan.conf.GetString("BUNDLE_ID")); err != nil {
		log.Panicf("Unable to create router client: Address: %s, Error: %s", bEhsan.conf.GetString("JOB_INT_ADDRESS"), err)
	} else {
		bEhsan.client = client
	}

	log.Printf("Bundle 2: GROUP: %s, INDEX: %s", bEhsan.group, bEhsan.index)

	if jh, err := NewJobHandler(bEhsan.conf); err != nil {
		log.Panicln("Failed to initialize Router API Job Handler", err.Error())
	} else if err := jh.RegisterWorkers(); err != nil {
		log.Panicln("Failed to register Router API Workers", err.Error())
	} else {
		bEhsan.jobHandler = jh
	}

	rand.Seed(time.Now().UTC().UnixNano())
}

func TestAnyCast(t *testing.T) {
	secret := func(len int) string {
		b := make([]byte, len)
		rand.Read(b)

		return fmt.Sprintf("%x", b)
	}(32)

	// Create Packet
	packet := bPouyan.client.Router.PrepareAnycast(protocol.NewRequest("test/hello", protocol.D{"secret": secret}), "HELLO", "EHSAN")
	b, _ := json.Marshal(packet.Datagram())

	// Subscribe
	chRequests := make(chan *nats.Msg)
	received := false
	if subs, err := bEhsan.jobHandler.iconn.ChanSubscribe("HELLO", chRequests); err != nil {
		log.Printf("Unable to subscribe to %s. Error: %s", "HELLO", err)
	} else {
		defer subs.Unsubscribe()
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			select {
			case msg := <-chRequests:
				func(msg *nats.Msg) {
					defer wg.Done()
					received = true
					if !bytes.Equal(b, msg.Data) {
						t.Errorf("Invalid data received. Expected: %s got %s", string(b), string(msg.Data))
					}
				}(msg)

			case <-time.After(time.Second * 10):
				wg.Done()
			}
		}
	}()

	// Publish
	if err := bPouyan.jobHandler.iconn.Publish(packet.Address(), b); err != nil {
		log.Printf("Unable to publish to %s. Error: %s", packet.Address(), err)
	}

	wg.Wait()

	if !received {
		t.Error("Message not received")
	}
}
