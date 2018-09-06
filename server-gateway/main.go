package main

import (
    "go.uber.org/zap"
    "gopkg.in/fzerorubigd/onion.v3"
)

var (
    _Config   *onion.Onion
    _BundleID string
    _Log      zap.Logger
)

func init() {
    _Config = readConfig()
    _BundleID = _Config.GetString("BUNDLE_ID")

}

func main() {
    server := NewGatewayServer()
    server.Run()
    server.Shutdown()
}
