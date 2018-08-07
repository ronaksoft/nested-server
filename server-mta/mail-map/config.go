package main

import (
    "gopkg.in/fzerorubigd/onion.v3"
    "gopkg.in/fzerorubigd/onion.v3/extraenv"
)

func readConfig() *onion.Onion {
    dl := onion.NewDefaultLayer()

    // InstanceID
    dl.SetDefault("INSTANCE_ID", "")

    // Model
    dl.SetDefault("MONGO_DSN", "mongodb://db-mongo.nst:27001")
    dl.SetDefault("REDIS_DSN", "cache-redis.nst:6379")

    // Mail
    dl.SetDefault("DOMAIN", "nested.dev")

    cfg := onion.New()
    cfg.AddLayer(dl)
    cfg.AddLazyLayer(extraenv.NewExtraEnvLayer("NST"))

    return cfg
}

