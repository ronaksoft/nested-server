package file

import (
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server-gateway/gateway_file/convert"
    "git.ronaksoftware.com/ronak/toolbox/logger"
    "gopkg.in/fzerorubigd/onion.v3"
)

var (
    _Log           *log.Logger
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
    _Log = log.NewTerminalLogger(log.LEVEL_DEBUG)
    _FileConverter, _ = convert.NewFileConverter()

    return s
}
