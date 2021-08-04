package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

/*
   Creation Time: 2021 - Aug - 04
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

var (
	_Log      *zap.Logger
	_LogLevel zap.AtomicLevel
)

func init() {
	// Initialize Logger
	_LogLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	config := zap.NewProductionConfig()
	config.Encoding = "console"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.Level = _LogLevel
	if v, err := config.Build(); err != nil {
		os.Exit(1)
	} else {
		_Log = v
	}
}

func SetLevel(lvl zapcore.Level) {
	_LogLevel.SetLevel(lvl)
}

func Debug(msg string, fields ...zap.Field) {
	_Log.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	_Log.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	_Log.Warn(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	_Log.Fatal(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	_Log.Error(msg, fields...)
}

func Sugar() *zap.SugaredLogger {
	return _Log.Sugar()
}