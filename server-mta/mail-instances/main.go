package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"gopkg.in/op/go-logging.v1"
	"io"
	"net"
	"os"
	"strings"
	"fmt"
)

var (
	_Log *logging.Logger
)

const LOG_PREFIX string = "nested/mail-instances"

type mailInfo struct {
	Sender     string        `json:"sender"`
	Domain     string        `json:"domain"`
	Recipients []string      `json:"recipients"`
	Buffer     []byte        `json:"buffer"`
}

func main() {
	// --Configurations
	sender := flag.String("s", "", "Sender Address")
	domain := flag.String("d", "", "domain")
	flag.Parse()
	initLogger("file,std,syslog", 3)
	recipients := flag.Args()

	buf := new(bytes.Buffer)
	io.Copy(buf, os.Stdin)

	m := mailInfo{
		Sender:     *sender,
		Domain:     *domain,
		Recipients: recipients,
		Buffer:     buf.Bytes(),
	}
	_Log.Info("Buffer:     buf.Bytes(),", m.Recipients)
	b, err := json.Marshal(m)
	if err != nil {
		_Log.Error("mail-instances::json.Marshal(m)", err.Error())
		fmt.Println("mail-instances::json.Marshal(m)", err.Error())
	}
	_Log.Info("mail-instances::InstanceInfo:", *sender, *domain, recipients)
	fmt.Println("mail-instances::InstanceInfo:", *sender, *domain, recipients)

	conn, err := net.Dial("tcp", "127.0.0.1:2300")
	if err != nil {
		_Log.Error("net.Dial(tcp, :2300)", err.Error())
		fmt.Println("net.Dial(tcp, :2300)", err.Error())
	}
	_, err = conn.Write(b)
	if err != nil {
		_Log.Error("conn.Write(b)", err.Error())
		fmt.Println("conn.Write(b)", err.Error())
	}
	defer conn.Close()
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
				if fh, err := os.OpenFile("/tmp/mail-instances.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); nil == err {
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
