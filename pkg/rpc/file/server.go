package file

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/config"
	"git.ronaksoft.com/nested/server/pkg/log"
	"git.ronaksoft.com/nested/server/pkg/rpc/file/convert"
	"go.uber.org/zap"
)

var (
	_FileConverter *convert.FileConverter
	_NestedModel   *nested.Manager
)

type Server struct {
	apiKey     string
	compressed bool
}

func NewServer(model *nested.Manager) *Server {
	s := new(Server)
	s.apiKey = config.GetString(config.FileSystemKey)
	s.compressed = false
	_NestedModel = model

	var err error
	_FileConverter, err = convert.NewFileConverter()
	if err != nil {
		log.Warn("We got error on initializing FileConverter", zap.Error(err))
	}

	return s
}
