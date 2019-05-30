package main

import (
	"flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"runtime"
)

const LOG_PREFIX string = "nested/router"

var (
	_Log      *zap.Logger
	_LogLevel zap.AtomicLevel
)

func init() {
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
	cPath := flag.String("c", "/etc/nested.toml", "Config file path")
	flag.Parse()

	_Log.Info("Loading config file ",
		zap.String("Path", *cPath),
	)
	conf := readConfig(*cPath)

	if jh, err := NewJobHandler(conf); err != nil {
		_Log.Fatal("Failed to create workers",
			zap.Error(err),
		)
	} else if err := jh.RegisterWorkers(); err != nil {
		_Log.Fatal("Failed to run workers",
			zap.Error(err),
		)
	}

	runtime.Goexit()
}
