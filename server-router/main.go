package main

import (
	"os"
	"flag"
	"runtime"
	"strings"

	"gopkg.in/op/go-logging.v1"
)

const LOG_PREFIX string = "nested/router"

var (
	_Log *logging.Logger
)

func main() {
	cPath := flag.String("c", "/etc/nested.toml", "Config file path")

	verbosity := flag.Int("v", 1, "Verbosity level [0, 3]")
	logWriters := flag.String("log", "std", "Log writer (:= std)")

	flag.Parse()
	initLogger(*logWriters, *verbosity)

	_Log.Infof("Loading config file '%s'...\n", *cPath)
	conf := readConfig(*cPath)

	if jh, err := NewJobHandler(conf); err != nil {
		_Log.Errorf("Failed to initialize Router API Job Handler: %s", err.Error())
		_Log.Fatal("Failed to create workers")
	} else if err := jh.RegisterWorkers(); err != nil {
		_Log.Errorf("Failed to register Router API Workers: %s", err.Error())
		_Log.Fatal("Failed to run workers")
	}

	runtime.Goexit()
}

func initLogger(writers string, verbosity int) {
	if logger, err := logging.GetLogger("main"); err != nil {
		os.Exit(1)
	} else {
		_Log = logger
		if 0 == verbosity {
			return
		}

		writers := strings.Split(writers, ",")
		level := logging.CRITICAL
		switch {
		case 2 == verbosity:
			level = logging.INFO

		case 3 <= verbosity:
			level = logging.DEBUG
		}

		var backends []logging.Backend
		for _, v := range writers {
			var backend logging.Backend
			switch strings.TrimSpace(v) {
			case "std":
				backend = logging.NewLogBackend(os.Stdout, "", 0)

			case "file":
				if fh, err := os.OpenFile("/var/log/router.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); nil == err {
					backend = logging.NewLogBackend(fh, "", 0)
				}

			case "syslog":
				if b, err := logging.NewSyslogBackend(LOG_PREFIX); nil == err {
					backend = b
				} else {
					panic(err)
				}
			}

			if backend != nil {
				lvlBackend := logging.AddModuleLevel(backend)
				lvlBackend.SetLevel(level, "")
				backends = append(backends, lvlBackend)
			}
		}

		logging.SetBackend(backends...)
	}
}
