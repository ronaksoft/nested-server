package main

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/config"
	"os"
)

var (
	_Nested *nested.Manager
)

func main() {
	var (
		err error
	)

	// Initialize Nested Model
	_Nested, err = nested.NewManager(
		config.GetString(config.InstanceID),
		config.GetString(config.MongoDSN),
		config.GetString(config.RedisDSN),
		config.GetInt(config.DebugLevel),
	)
	if err != nil {
		os.Exit(1)
	}

}
