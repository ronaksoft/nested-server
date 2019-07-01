package main

import (
	"go.uber.org/zap"
	"gopkg.in/fzerorubigd/onion.v3"
	"gopkg.in/fzerorubigd/onion.v3/extraenv"
)

func readConfig() *onion.Onion {
	dl := onion.NewDefaultLayer()

	// Nested ID
	dl.SetDefault("INSTANCE_ID", "")

	// Profiling Settings
	dl.SetDefault("DEBUG_CPU_PROFILE", true)
	dl.SetDefault("DEBUG_MEM_PROFILE", true)

	// Storage Settings
	// dl.SetDefault("QUEUE_STORAGE_PATH", "/ronak/store/q")
	dl.SetDefault("BUNDLE_ID", "")
	dl.SetDefault("LOG_LEVEL", zap.DebugLevel)

	// External Services
	dl.SetDefault("JOB_ADDRESS", "nats://localhost:4222")
	dl.SetDefault("JOB_USER", "")
	dl.SetDefault("JOB_PASS", "")
	dl.SetDefault("JOB_WORKERS_COUNT", 1000)
	dl.SetDefault("MONGO_DSN", "localhost:27017")
	dl.SetDefault("MONGO_USER", "ehsan")
	dl.SetDefault("MONGO_PASS", "ehsan2374")
	dl.SetDefault("REDIS_DSN", "localhost:6379")
	dl.SetDefault("REDIS_PASS", "")

	// NTFY Settings
	dl.SetDefault("GCM_API_KEY", "")

	cfg := onion.New()
	cfg.AddLayer(dl)
	cfg.AddLazyLayer(extraenv.NewExtraEnvLayer("NST"))
	return cfg
}
