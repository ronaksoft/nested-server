package main

import (
	"git.ronaksoft.com/nested/server/pkg/config"
	"git.ronaksoft.com/nested/server/pkg/log"
)

func main() {
	// Set Log Level
	log.SetLevel(log.Level(config.GetInt(config.LogLevel)))

	app := NewAPP()
	app.Run()
	app.Shutdown()
}
