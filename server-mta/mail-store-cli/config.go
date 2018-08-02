package main

import (
	"gopkg.in/fzerorubigd/onion.v3"
	"gopkg.in/fzerorubigd/onion.v3/extraenv"
	_ "gopkg.in/fzerorubigd/onion.v3/tomlloader"
)

func readConfig() *onion.Onion {
	dl := onion.NewDefaultLayer()

	// Model
	dl.SetDefault("MONGO_DSN", "mongodb://db-mongo.nst:27001")
	dl.SetDefault("REDIS_DSN", "cache-redis.nst:6379")

	// Mail
	dl.SetDefault("DOMAIN", "nested.dev")
	dl.SetDefault("MAILER_DAEMON", "MAILER-DAEMON")

	// HTTP GameServer
	dl.SetDefault("CYRUS_URL", "http://storage.xerxes.nst")
	dl.SetDefault("CYRUS_FILE_SYSTEM_KEY", "5b47e841ee52a16bc797f6bcc06c41d68546fb4620709227f8911fd969e0ed26")
	dl.SetDefault("CYRUS_INSECURE_HTTPS", false)

	// Job GameServer
	dl.SetDefault("JOB_ADDRESS", "nats://job.cyrus.nst:4222")

	cfg := onion.New()
	cfg.AddLayer(dl)
	cfg.AddLazyLayer(extraenv.NewExtraEnvLayer("NST"))

	return cfg
}
