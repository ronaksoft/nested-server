package main

import (
	"gopkg.in/fzerorubigd/onion.v3"
	"gopkg.in/fzerorubigd/onion.v3/extraenv"
	_ "gopkg.in/fzerorubigd/onion.v3/tomlloader"
)

func readConfig(filename string) *onion.Onion {
	dl := onion.NewDefaultLayer()

	// Bundle
	_ = dl.SetDefault("BUNDLE_ID", "NESTED-000")

	// QM Server
	_ = dl.SetDefault("JOB_INT_ADDRESS", "nats://job.nested.nst:4222")
	_ = dl.SetDefault("JOB_INT_WORKERS_COUNT", 50)
	_ = dl.SetDefault("JOB_INT_BUFFER_SIZE", 100)
	_ = dl.SetDefault("JOB_EXT_ADDRESS", "nats://job.arsaces.nst:4222")
	_ = dl.SetDefault("JOB_EXT_WORKERS_COUNT", 50)
	_ = dl.SetDefault("JOB_EXT_BUFFER_SIZE", 100)

	cfg := onion.New()
	_ = cfg.AddLayer(dl)
	_ = cfg.AddLayer(onion.NewFileLayer(filename))
	cfg.AddLazyLayer(extraenv.NewExtraEnvLayer("NST"))

	return cfg
}
