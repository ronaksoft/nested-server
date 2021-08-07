package convert

import (
	"fmt"
	"os/exec"
	"path"
	"reflect"
	"strings"
)

var (
	_Commands ExecPath
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
			return err
		} else {
			cmdPath = execPath
		}

		vField.SetString(cmdPath)
	}

	return nil
}

func NewFileConverter() (*FileConverter, error) {
	if err := _Commands.init(); err != nil {
		return nil, err
	}

	return &FileConverter{}, nil
}
