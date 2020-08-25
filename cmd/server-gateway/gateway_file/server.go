package file

import (
	"os"

	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_file/convert"
	"git.ronaksoft.com/nested/server/model"
	"go.uber.org/zap"
	"gopkg.in/fzerorubigd/onion.v3"
)

var (
	_Log           *zap.Logger
	_LogLevel      zap.AtomicLevel
	_FileConverter *convert.FileConverter
	_NestedModel   *nested.Manager
)

type Server struct {
	apiKey     string
	compressed bool
}

func NewServer(config *onion.Onion, model *nested.Manager) *Server {
	s := new(Server)
	s.apiKey = config.GetString("FILE_SYSTEM_KEY")
	s.compressed = false
	_NestedModel = model

	_LogLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	zap.NewProductionConfig()
	logConfig := zap.NewProductionConfig()
	logConfig.Encoding = "console"
	logConfig.Level = _LogLevel
	if v, err := logConfig.Build(); err != nil {
		os.Exit(1)
	} else {
		_Log = v
	}
	_FileConverter, _ = convert.NewFileConverter()

	return s
}
