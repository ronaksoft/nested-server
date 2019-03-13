package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"git.ronaksoftware.com/nested/server/cmd/server-mta/mail-store-cli/cache"
	"git.ronaksoftware.com/nested/server/cmd/server-mta/mail-store-cli/client-ntfy"
	"git.ronaksoftware.com/nested/server/cmd/server-mta/mail-store-cli/client-storage"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/globalsign/mgo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/fzerorubigd/onion.v3"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

type Model struct {
	Session    *mgo.Session
	DB         string
	Ntfy       *client_ntfy.NtfyClient
	Storage    *client_storage.Client
	Cache      *cache.CacheManager
	InstanceID string
	MongoDSN   string
	CyrusURL   string
}

type mailInfo struct {
	Sender     string   `json:"sender"`
	Domain     string   `json:"domain"`
	Recipients []string `json:"recipients"`
	Buffer     []byte   `json:"buffer"`
}

var (
	_LOG         *zap.Logger
	_Config      *onion.Onion
	instanceInf  = make(map[string]*Model)
	containerENV map[string]string
)

func main() {
	_Config = readConfig()
	initLogger()
	defer _LOG.Sync()
	defer recoverPanic()

	// detect information of nested instances
	detectInstances()
	// check running nested docker containers
	go runEvery(time.Minute*time.Duration(_Config.GetInt("WATCHDOG_INTERVAL")), watchdog)

	_LOG.Info("mail-store-cli::Start Listening::tcp:2300")
	listener, err := net.Listen("tcp", ":2300")
	if err != nil {
		_LOG.Error(err.Error())
	}
	defer listener.Close()

	// Listen for incoming email mailInfo
	for {
		conn, err := listener.Accept()
		if err != nil {
			_LOG.Error(err.Error())
			continue
		}
		d := json.NewDecoder(conn)
		m := mailInfo{}
		err = d.Decode(&m)
		if err != nil {
			_LOG.Error(err.Error())
			continue
		}
		if err := dispatch(m.Domain, m.Sender, m.Recipients, m.Buffer, instanceInf[m.Domain]); err != nil {
			_LOG.Error("failed to store mail", zap.Error(err))
		}
	}
}

func NewModel(session *mgo.Session, ntfy *client_ntfy.NtfyClient, storage *client_storage.Client, redisCache *cache.CacheManager, ID string, cyrusURL string) *Model {
	model := new(Model)
	model.Session = session
	model.Ntfy = ntfy
	model.Storage = storage
	model.Cache = redisCache
	model.InstanceID = ID
	model.DB = fmt.Sprintf("nested-%s", ID)
	model.CyrusURL = cyrusURL
	return model
}

func initMongo(mongoDSN string) (*mgo.Session, error) {
	// Initial MongoDB
	tlsConfig := new(tls.Config)
	tlsConfig.InsecureSkipVerify = true
	if dialInfo, err := mgo.ParseURL(mongoDSN); err != nil {
		_LOG.Error("initMongo::MongoDB URL Parse Failed::", zap.Error(err))
		return nil, err
	} else {
		dialInfo.Timeout = 5 * time.Second
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			if conn, err := tls.Dial("tcp", addr.String(), tlsConfig); err != nil {
				return conn, err
			} else {
				return conn, nil
			}
		}
		if mongoSession, err := mgo.DialWithInfo(dialInfo); err != nil {
			_LOG.Error("initMongo::DialWithInfo Failed::", zap.Error(err))
			if mongoSession, err = mgo.Dial(mongoDSN); err != nil {
				_LOG.Error("initMongo::Dial Failed::", zap.Error(err))
				return nil, err
			} else {
				_LOG.Debug("initMongo::MongoDB Connected")
				return mongoSession, nil
			}
		} else {
			_LOG.Debug("initMongo::MongoDB(TLS) Connected")
			return mongoSession, nil
		}
	}
}

func detectInstances() {
	cli, err := client.NewClientWithOpts(client.WithVersion("1.37"))
	if err != nil {
		_LOG.Error(err.Error())
	}
	ctx := context.Background()
	args := filters.NewArgs(filters.Arg("name", "gateway"))
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{Filters: args})
	if err != nil {
		_LOG.Error(err.Error())
	}
	for _, container := range containers {
		env, _ := cli.ContainerInspect(ctx, container.ID)
		containerENV = make(map[string]string, len(env.Config.Env))
		for _, item := range env.Config.Env {
			parts := strings.Split(item, "=")
			containerENV[parts[0]] = parts[1]
		}
		_LOG.Debug("containerENV :", zap.String("containerDomain", containerENV["NST_DOMAIN"]))
		session, err := initMongo(containerENV["NST_MONGO_DSN"])
		if err != nil {
			_LOG.Error(err.Error())
		}
		ntfy := client_ntfy.NewNtfyClient(containerENV["NST_JOB_ADDRESS"], containerENV["NST_DOMAIN"])
		storage, err := client_storage.NewClient(containerENV["NST_CYRUS_URL"], containerENV["NST_FILE_SYSTEM_KEY"], true)
		if err != nil {
			_LOG.Error(err.Error())
		}
		redisCache, err := cache.NewCacheManager(containerENV["NST_REDIS_DSN"])
		if err != nil {
			_LOG.Error(err.Error())
		}
		model := NewModel(session, ntfy, storage, redisCache, containerENV["NST_INSTANCE_ID"], containerENV["NST_CYRUS_URL"])
		instanceInf[containerENV["NST_DOMAIN"]] = model
	}
}

func initLogger() {
	logLevel := zap.NewAtomicLevelAt(zapcore.Level(_Config.GetInt("DEBUG_LEVEL")))
	fileLog, _ := os.Create("/var/log/mail-store-cli.log")
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

// recoverPanic
// This function will avoid server panics and logs the incident
func recoverPanic() {
	r := recover()
	if r != nil {
		_LOG.Warn("panic log", zap.Any("", r))
		debug.PrintStack()
		_LOG.Error("*********PANIC RECOVERED*********")
	}
}

// watchdog will catch newly added/removed nested containers and updates instanceInf map
func watchdog(t time.Time) {
	cli, err := client.NewClientWithOpts(client.WithVersion("1.37"))
	if err != nil {
		_LOG.Error(err.Error())
		return
	}
	ctx := context.Background()
	args := filters.NewArgs(
		filters.Arg("name", "gateway"))

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{Filters: args})
	if err != nil {
		_LOG.Error(err.Error())
		return
	}
	domains := make([]string, 0, len(containers))
	if len(containers) != len(instanceInf) {
		for _, container := range containers {
			env, _ := cli.ContainerInspect(ctx, container.ID)
			containerENV = make(map[string]string, len(env.Config.Env))
			for _, item := range env.Config.Env {
				parts := strings.Split(item, "=")
				containerENV[parts[0]] = parts[1]
			}
			domains = append(domains, containerENV["NST_DOMAIN"])
			if _, ok := instanceInf[containerENV["NST_DOMAIN"]]; ok {
				continue
			} else {
				session, err := initMongo(containerENV["NST_MONGO_DSN"])
				if err != nil {
					_LOG.Error(err.Error())
				}
				ntfy := client_ntfy.NewNtfyClient(containerENV["NST_JOB_ADDRESS"], containerENV["NST_DOMAIN"])
				storage, err := client_storage.NewClient(containerENV["NST_CYRUS_URL"], containerENV["NST_FILE_SYSTEM_KEY"], true)
				if err != nil {
					_LOG.Error(err.Error())
				}
				redisCache, err := cache.NewCacheManager(containerENV["NST_REDIS_DSN"])
				if err != nil {
					_LOG.Error(err.Error())
				}
				model := NewModel(session, ntfy, storage, redisCache, containerENV["NST_INSTANCE_ID"], containerENV["NST_CYRUS_URL"])
				instanceInf[containerENV["NST_DOMAIN"]] = model
				_LOG.Info("nested instance added: ", zap.String("DOMAIN", containerENV["NST_DOMAIN"]))
			}
		}
		for domainBefore := range instanceInf {
			exist := false
			for _, domain := range domains {
				if domain == domainBefore {
					exist = true
					continue
				}
			}
			if exist == false {
				delete(instanceInf, domainBefore)
				_LOG.Info("nested instance removed: ", zap.String("DOMAIN", domainBefore))
			}
		}
	}
}

func runEvery(t time.Duration, f func(time.Time)) {
	for x := range time.Tick(t) {
		f(x)
	}
}
