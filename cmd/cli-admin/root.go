package main

import (
	"github.com/spf13/cobra"
	"gopkg.in/fzerorubigd/onion.v3"
	"gopkg.in/fzerorubigd/onion.v3/extraenv"
)

var (
	_Config        *onion.Onion
	pathConfigFile string
)

var RootCmd = &cobra.Command{
	Use:   "nested-admin",
	Short: "nested admin command-line interface",
}

func init() {
	// read config
	_Config = readConfig()

}

func readConfig() *onion.Onion {
	dl := onion.NewDefaultLayer()

	dl.SetDefault("DOMAIN", "nested.me")

	// Cloudflare Configs
	dl.SetDefault("CF_EMAIL_ADDR", "ehsan@ronaksoft.com")
	dl.SetDefault("CF_GLOBAL_API_KEY", "***REMOVED***")
	dl.SetDefault("CF_ORIGIN_CA_KEY", "***REMOVED***")

	cfg := onion.New()
	cfg.AddLayer(dl)

	cfg.AddLazyLayer(extraenv.NewExtraEnvLayer(""))
	return cfg
}
