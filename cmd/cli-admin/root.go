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
	dl.SetDefault("CF_GLOBAL_API_KEY", "5a48658f65f812c47c1054536838e00cfc5f9")
	dl.SetDefault("CF_ORIGIN_CA_KEY", "v1.0-ade789beb91eb814a17204dec2b3c2d1fa5d745fd57af558a08987883382cb37-3bf69cd33841b5c4e4a34119b25f6be96c291943f2e476b7441b96a9737aa002bd059038c5e8853f11f8dd00eb77cf90237e1f52f8822ef5023036d419c07d0b-bd0d89d35d4e8b02eb930cf8bccfb4cb8a6c6fa8b26ac28009cc722941053532")

	cfg := onion.New()
	cfg.AddLayer(dl)

	cfg.AddLazyLayer(extraenv.NewExtraEnvLayer(""))
	return cfg
}
