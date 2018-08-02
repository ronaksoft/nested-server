package main

import (
    "io"
    "os"
    "time"
    "errors"
    "git.ronaksoftware.com/nested/server-gateway/client"
    "git.ronaksoftware.com/nested/server-model-nested"
)

func uploadFile(filename, uploaderId, status string, ownerIds []string, r io.Reader) (*nestedGateway.UploadedFile, error) {
    f := &MyFile{
        r:        r,
        name:     filename,
        uploader: uploaderId,
        status:   status,
        owners:   ownerIds,
    }

    _Log.Debugf("Uploading file %s...", filename)

    if res, err := _ClientStorage.Upload(nested.UPLOAD_TYPE_FILE, f); err != nil {
        return nil, err
    } else if 0 == len(res.Files) {
        return nil, errors.New("file not uploaded")
    } else {
        return &res.Files[0], nil
    }
}

type MyFile struct {
    r        io.Reader
    name     string
    uploader string
    status   string
    owners   []string
}

func (f *MyFile) Read(p []byte) (int, error) {
    return f.r.Read(p)
}

func (f MyFile) Name() string {
    return f.name
}

func (f MyFile) Size() int64 {
    return 0
}

func (f MyFile) Mode() os.FileMode {
    return os.ModePerm
}

func (f MyFile) ModTime() time.Time {
    return time.Now()
}

func (f MyFile) IsDir() bool {
    return false
}

func (f MyFile) Sys() interface{} {
    return nil
}

func (f *MyFile) Stat() (os.FileInfo, error) {
    return f, nil
}
