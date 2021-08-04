package file

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_file/convert"
	"git.ronaksoft.com/nested/server/nested"
	"gopkg.in/fzerorubigd/onion.v3"
)

var (
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
	_FileConverter, _ = convert.NewFileConverter()

	return s
}
