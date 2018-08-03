package main

import (
    "gopkg.in/fzerorubigd/onion.v3"
)

var (
    _Config        *onion.Onion
    _BundleID      string
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
