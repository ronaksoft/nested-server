package convert

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/log"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"

	"go.uber.org/zap"
)

var (
	_Commands ExecPath
	_Log      *zap.Logger
	_LogLevel zap.AtomicLevel
)

type FileConverter struct {
	Pdf     Pdf
	Gif     Gif
	Audio   Audio
	Image   Image
	Video   Video
	Preview Preview
	Voice   Voice
}

type ExecPath struct {
	Tr       string
	Awk      string
	Sed      string
	Grep     string
	Ffmpeg   string
	Ffprobe  string
	Convert  string
	Identify string
	PdfInfo  string
}

func (ep *ExecPath) init() error {
	v := reflect.ValueOf(ep)
	e := v.Elem()
	t := e.Type()

	for i := 0; i < t.NumField(); i++ {
		tField := t.Field(i)
		vField := e.Field(i)

		cmd := strings.ToLower(tField.Name)
		cmdPath := fmt.Sprintf("%s/%s", path.Dir(vField.String()), cmd)

		if execPath, err := exec.LookPath(cmd); err != nil {
			log.Warn(err.Error())
			return err
		} else {
			cmdPath = execPath
		}

		vField.SetString(cmdPath)
	}

	return nil
}

func NewFileConverter() (*FileConverter, error) {
	if nil == _Log {
		_LogLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
		zap.NewProductionConfig()
		config := zap.NewProductionConfig()
		config.Encoding = "console"
		config.Level = _LogLevel
		if v, err := config.Build(); err != nil {
			os.Exit(1)
		} else {
			_Log = v
		}
	}

	if err := _Commands.init(); err != nil {
		return nil, err
	}

	fcnv := &FileConverter{}

	return fcnv, nil
}
