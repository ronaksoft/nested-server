package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/fzerorubigd/onion.v3"
	"gopkg.in/fzerorubigd/onion.v3/extraenv"
)

var (
	_Config *onion.Onion
	// pathYMLsDir pointing to the directory which docker-compose.yml files will be stored.
	// Each service (ie. cyrus, xerxes ...) have their own folder inside pathYMLsDir
	pathYMLsDir string
	// pathCertsDir pointing to the directory which cert files are stored. Each certificate is a pair of
	// PEM files. i.e. cyrus.crt & cyrus.key
	pathCertsDir string
	// pathTemplatesDir pointing to the template directory which docker-compose.yml files are created from those
	// templates. This folder MUST NOT be touched by users.
	pathTemplatesDir string
	// pathConfigFile is pointing to the directory which config.yml file exists. Administrators must change this file
	// and then call: "nested-ctl services install" to update all necessary docker-compose.yml files.
	pathConfigFile string
)

var RootCmd = &cobra.Command{
	Use:   "nested-ctl",
	Short: "nested controller command-line interface",
}

func init() {
	// read config
	_Config = readConfig()

	// prepare default paths
	pathYMLsDir = fmt.Sprintf("%s/yamls", _Config.GetString("NESTED_DIR"))
	pathTemplatesDir = fmt.Sprintf("%s/templates", _Config.GetString("NESTED_DIR"))
	pathCertsDir = fmt.Sprintf("%s/certs", _Config.GetString("NESTED_DIR"))
	pathConfigFile = fmt.Sprintf("%s/config/config.yml", _Config.GetString("NESTED_DIR"))

	// prepare Root flags
	RootCmd.PersistentFlags().String("config", pathConfigFile, "Config file")
}

func readConfig() *onion.Onion {
	dl := onion.NewDefaultLayer()
	dl.SetDefault("NESTED_DIR", "/ronak/nested")

	cfg := onion.New()
	cfg.AddLayer(dl)

	cfg.AddLazyLayer(extraenv.NewExtraEnvLayer(""))
	return cfg
}
