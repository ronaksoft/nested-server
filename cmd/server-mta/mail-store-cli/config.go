package main

import (
	"gopkg.in/fzerorubigd/onion.v3"
	"gopkg.in/fzerorubigd/onion.v3/extraenv"
	_ "gopkg.in/fzerorubigd/onion.v3/tomlloader"
)

func readConfig() *onion.Onion {
	dl := onion.NewDefaultLayer()

	// Log
	dl.SetDefault("DEBUG_LEVEL", "")

	// InstanceID
	dl.SetDefault("INSTANCE_ID", "")

	// Watchdog interval
	dl.SetDefault("WATCHDOG_INTERVAL", "1")

	// Model
	dl.SetDefault("MONGO_DSN", "mongodb://db-mongo.nst:27001")
	dl.SetDefault("REDIS_DSN", "cache-redis.nst:6379")

	// Mail
	dl.SetDefault("DOMAIN", "nested.dev")
	dl.SetDefault("MAILER_DAEMON", "MAILER-DAEMON")

	// HTTP Server
	dl.SetDefault("CYRUS_URL", "http://storage.xerxes.nst")
	dl.SetDefault("CYRUS_FILE_SYSTEM_KEY", "5b47e841ee52a16bc797f6bcc06c41d68546fb4620709227f8911fd969e0ed26")
	dl.SetDefault("CYRUS_INSECURE_HTTPS", false)

	// Job Server
	dl.SetDefault("JOB_ADDRESS", "nats://job.cyrus.nst:4222")

	cfg := onion.New()
	cfg.AddLayer(dl)
	cfg.AddLazyLayer(extraenv.NewExtraEnvLayer("NST"))

	return cfg
}
