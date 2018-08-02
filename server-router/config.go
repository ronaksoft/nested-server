package main

import (
  "gopkg.in/fzerorubigd/onion.v2"
  "gopkg.in/fzerorubigd/onion.v2/extraenv"
  _ "gopkg.in/fzerorubigd/onion.v2/tomlloader"
)

func readConfig(filename string) *onion.Onion {
  dl := onion.NewDefaultLayer()

  // Bundle
  dl.SetDefault("BUNDLE_ID", "NESTED-000")

  // QM GameServer
  dl.SetDefault("JOB_INT_ADDRESS", "nats://job.nested.nst:4222")
  dl.SetDefault("JOB_INT_WORKERS_COUNT", 50)
  dl.SetDefault("JOB_INT_BUFFER_SIZE", 100)
  dl.SetDefault("JOB_EXT_ADDRESS", "nats://job.arsaces.nst:4222")
  dl.SetDefault("JOB_EXT_WORKERS_COUNT", 50)
  dl.SetDefault("JOB_EXT_BUFFER_SIZE", 100)

  cfg := onion.New()
  cfg.AddLayer(dl)
  cfg.AddLayer(onion.NewFileLayer(filename))
  cfg.AddLazyLayer(extraenv.NewExtraEnvLayer("NST"))

  return cfg
}

