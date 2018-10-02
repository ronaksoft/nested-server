package main

import (
    "flag"
    "log"
    "os"
    "strings"

    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server/server-gateway/client"
    "git.ronaksoftware.com/nested/server/server-ntfy/client"
    "gopkg.in/fzerorubigd/onion.v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"fmt"
)

const LOG_PREFIX string = "nested/mailbox-store"

var (
	_ClientStorage *nestedGateway.Client
	_ClientNtfy    *ntfy.Client
	_Config        *onion.Onion
	_Model *nested.Manager
	_LOG           *zap.Logger
)

/**
 *  argv list:
 *      1 - Sender Address
 *      2 - Recipient(1)
 *      3 - Recipient(2)
 *      ...
 *      n - Recipient(n-1)
 */
func main() {
	// --Configurations
	_Config = readConfig()
	initLogger()
	defer _LOG.Sync()

	sender := flag.String("s", "", "Sender Address")
	recipients := flag.Args()
	flag.Parse()

	if 0 == len(strings.TrimSpace(*sender)) {
		_LOG.Fatal("Invalid Input: Sender is necessary")
	}

	if 0 == len(recipients) {
		_LOG.Fatal("Invalid Input: At least one recipient is required")
	}

	_LOG.Debug(fmt.Sprintf("Cyrus Url: %s", _Config.GetString("CYRUS_URL")))
	_LOG.Debug(fmt.Sprintf("Cyrus Api DHKey: %s", _Config.GetString("CYRUS_FILE_SYSTEM_KEY")))
	_LOG.Debug(fmt.Sprintf("Domain: %s", _Config.GetString("DOMAIN")))

	// Instantiate Storage Client
	if v, err := nestedGateway.NewClient(
		_Config.GetString("CYRUS_URL"),
		_Config.GetString("CYRUS_FILE_SYSTEM_KEY"),
		_Config.GetBool("CYRUS_INSECURE_HTTPS"),
	); err != nil {
		_LOG.Error(err.Error())
		_LOG.Fatal("Failed to connect to Storage Agent")
	} else {
		_ClientStorage = v
	}

	// Instantiate Nested Model Manager
	if n, err := nested.NewManager(
		_Config.GetString("INSTANCE_ID"),
		_Config.GetString("MONGO_DSN"),
		_Config.GetString("REDIS_DSN"),
		_Config.GetInt("DEBUG_LEVEL"),
	); err != nil {
		log.Println("MAILSTORE::Main::Nested Manager Error::", err.Error())
		os.Exit(1)
	} else {
		_Model = n
	}

	// Instantiate NTFY Client
	if _ClientNtfy = ntfy.NewClient(_Config.GetString("JOB_ADDRESS"), _Model); _ClientNtfy == nil {
		_LOG.Error("Failed to instantiate NTFY client")
		_LOG.Fatal("Failed to connect to NTFY Agent")
	}
	defer _ClientNtfy.Close()

	if err := Dispatch(*sender, recipients, os.Stdin); err != nil {
		_LOG.Error(err.Error())
		_LOG.Fatal("Unknown Message")
	}
}

func initLogger() {
	logLevel := zap.NewAtomicLevelAt(zapcore.Level(_Config.GetInt("CONF_LOG_LEVEL")))
	fileLog, _ := os.Create("/var/log/mailbox-store.log")
	defer fileLog.Close()
	consoleWriteSyncer := zapcore.Lock(os.Stdout)
	consoleEncoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})
	fileWriteSyncer := zapcore.Lock(fileLog)
	fileEncoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, fileWriteSyncer, logLevel),
		zapcore.NewCore(consoleEncoder, consoleWriteSyncer, logLevel),
	)
	_LOG = zap.New(core)
}
