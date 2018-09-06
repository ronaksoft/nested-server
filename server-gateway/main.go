package main

import (
    "os"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "gopkg.in/fzerorubigd/onion.v3"
)

var (
    _Config   *onion.Onion
    _BundleID string
    _Log      *zap.Logger
    _LogLevel zap.AtomicLevel
)

func init() {
    _Config = readConfig()
    _BundleID = _Config.GetString("BUNDLE_ID")

    // Initialize Logger
    _LogLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
    zap.NewProductionConfig()
    config := zap.NewProductionConfig()
    config.Encoding = "console"
    config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
    config.Level = _LogLevel
    if v, err := config.Build(); err != nil {
        os.Exit(1)
    } else {
        _Log = v
    }

}

func main() {
    server := NewGatewayServer()
    server.Run()
    server.Shutdown()
}
