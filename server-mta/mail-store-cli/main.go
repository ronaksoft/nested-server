package main

import (
	"os"
	"flag"
	"strings"
	"gopkg.in/op/go-logging.v1"
	"gopkg.in/fzerorubigd/onion.v3"
	"git.ronaksoftware.com/nested/server/model"
	"log"
	"git.ronaksoftware.com/nested/server/server-ntfy/client"
	"git.ronaksoftware.com/nested/server/server-gateway/client"
)

const LOG_PREFIX string = "nested/mailbox-store"

var (
	_Verbosity     int
	_ClientStorage *nestedGateway.Client
	_ClientNtfy    *ntfy.Client
	_Config        *onion.Onion
	_Model *nested.Manager
	_Log           *logging.Logger
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
	sender := flag.String("s", "", "Sender Address")
	flag.IntVar(&_Verbosity, "v", 1, "Verbosity level [0, 3]")
	logWriters := flag.String("log", "syslog", "Log writer (:= syslog)")
	flag.Parse()
	recipients := flag.Args()
	initLogger(*logWriters, _Verbosity)

	if 0 == len(strings.TrimSpace(*sender)) {
		_Log.Fatal("Invalid Input: Sender is necessary")
	}

	if 0 == len(recipients) {
		_Log.Fatal("Invalid Input: At least one recipient is required")
	}

	_Config = readConfig()
	// --/Configurations

	_Log.Infof("Cyrus Url: %s", _Config.GetString("CYRUS_URL"))
	_Log.Infof("Cyrus Api DHKey: %s", _Config.GetString("CYRUS_FILE_SYSTEM_KEY"))
	_Log.Infof("Domain: %s", _Config.GetString("DOMAIN"))

	// Instantiate Storage Client
	if v, err := nestedGateway.NewClient(
		_Config.GetString("CYRUS_URL"),
		_Config.GetString("CYRUS_FILE_SYSTEM_KEY"),
		_Config.GetBool("CYRUS_INSECURE_HTTPS"),
	); err != nil {
		_Log.Error("Failed to instantiate Storage client:", err.Error())
		_Log.Fatal("Failed to connect to Storage Agent")
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
		_Log.Error("Failed to instantiate NTFY client")
		_Log.Fatal("Failed to connect to NTFY Agent")
	}
	defer _ClientNtfy.Close()

	if err := Dispatch(*sender, recipients, os.Stdin); err != nil {
		_Log.Errorf("Failed to decide on message: %s", err.Error())
		_Log.Fatal("Unknown Message")
	}
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
				if fh, err := os.OpenFile("/var/log/mailbox-store.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); nil == err {
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
