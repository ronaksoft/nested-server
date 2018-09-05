package file

import (
    "errors"
    "fmt"
    "io"
    "path"
    "sync"

    "git.ronaksoftware.com/common/server-protocol"
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server/server-gateway/gateway_file/convert"
    "go.uber.org/zap"
)

const (
    THUMBNAIL_32      string = "x32"
    THUMBNAIL_64      string = "x64"
    THUMBNAIL_128     string = "x128"
    THUMBNAIL_PREVIEW string = "pre"
)

var DEFAULT_THUMBNAIL_SIZES = map[string]uint{
    THUMBNAIL_32:      32,
    THUMBNAIL_64:      64,
    THUMBNAIL_128:     256,
    THUMBNAIL_PREVIEW: 1024,
}

type pipe func(w io.Writer, r io.Reader) (int64, error)
type Processor interface {
    Process(io.Reader) error
}

// Thumbnail Generator
type thumbGenerator struct {
    MaxDimension uint
    Uploader     string
    Filename     string
    MimeType     string
    ThumbName    string

    Lock      sync.Locker
    MetaData  *nested.MetaData
    WaitGroup *sync.WaitGroup
}

func (p *thumbGenerator) Process(r io.Reader) error {
    defer p.WaitGroup.Done()

    var rThumb io.Reader
    var metaPreview *convert.PreviewMeta
    if rOut, m, err := _FileConverter.Preview.Thumbnail(r, p.MimeType, p.MaxDimension, p.MaxDimension); err != nil {
        _Log.Warn(err.Error())
        return err

    } else {
        rThumb = rOut
        metaPreview = m
    }

    ext := path.Ext(p.Filename)
    filename := fmt.Sprintf("%s-%s%s", p.Filename[:len(p.Filename)-len(ext)], p.ThumbName, metaPreview.Extension)
    fileInfo := nested.GenerateFileInfo(
        filename,
        p.Uploader,
        nested.FILE_TYPE_THUMBNAIL,
        nil,
        nested.MetaImage{
            Width:  metaPreview.Width,
            Height: metaPreview.Height,
        },
    )

    // Save File in Files Model
    if thumb := _NestedModel.Store.Save(rThumb, fileInfo); thumb == nil {
        err := errors.New("nested model save failed")
        _Log.Warn(err.Error())
        return err

    } else {
        meta := thumb.Metadata.Meta.(nested.MetaImage)
        thumbInfo := nested.FileInfo{
            ID:              nested.UniversalID(thumb.ID),
            Size:            int64(thumb.Size),
            Filename:        thumb.Name,
            Type:            thumb.Metadata.Type,
            MimeType:        thumb.MimeType,
            Status:          nested.FILE_STATUS_THUMBNAIL,
            UploadTimestamp: nested.Timestamp(),
            Width:           meta.Width,
            Height:          meta.Height,
        }

        // Save File in Nested Model
        if _NestedModel.File.AddFile(thumbInfo) != true {
            _Log.Warn("File info submit failed")
            return protocol.NewUnknownError(nil)

        } else {
            p.Lock.Lock()
            p.MetaData.Thumbnails[p.ThumbName] = *thumb
            p.Lock.Unlock()
        }
    }

    return nil
}

// Preview Generator
type previewGenerator struct {
    MaxWidth  uint
    Uploader  string
    Filename  string
    MimeType  string
    ThumbName string

    Lock      sync.Locker
    MetaData  *nested.MetaData
    WaitGroup *sync.WaitGroup
}

func (p *previewGenerator) Process(r io.Reader) error {
    defer p.WaitGroup.Done()

    var rPreview io.Reader
    var metaPreview *convert.PreviewMeta
    if rOut, m, err := _FileConverter.Preview.Resized(r, p.MimeType, p.MaxWidth, 0); err != nil {
        _Log.Warn(err.Error(),
            zap.String("FILENAME", p.Filename),
            zap.String("ThumbName", p.ThumbName),
        )

        return err

    } else {
        rPreview = rOut
        metaPreview = m
    }

    ext := path.Ext(p.Filename)
    filename := fmt.Sprintf("%s-%s%s", p.Filename[:len(p.Filename)-len(ext)], p.ThumbName, metaPreview.Extension)
    finfo := nested.GenerateFileInfo(
        filename,
        p.Uploader,
        nested.FILE_TYPE_THUMBNAIL,
        nil,
        nested.MetaImage{
            Width:  metaPreview.Width,
            Height: metaPreview.Height,
        },
    )

    // Save File in Files Model
    if thumb := _NestedModel.Store.Save(rPreview, finfo); thumb == nil {
        err := errors.New("file content submit failed")
        _Log.Warn(err.Error(),
            zap.String("FILENAME", finfo.Name),
            zap.String("ThumbName", string(finfo.ID)),
        )
        return err
    } else {
        meta := thumb.Metadata.Meta.(nested.MetaImage)
        thumbInfo := nested.FileInfo{
            ID:              nested.UniversalID(thumb.ID),
            Size:            int64(thumb.Size),
            Filename:        thumb.Name,
            Type:            thumb.Metadata.Type,
            MimeType:        thumb.MimeType,
            Status:          nested.FILE_STATUS_THUMBNAIL,
            UploadTimestamp: nested.Timestamp(),
            Width:           meta.Width,
            Height:          meta.Height,
        }

        if _NestedModel.File.AddFile(thumbInfo) != true {
            return protocol.NewUnknownError(nil)

        } else {
            p.Lock.Lock()
            p.MetaData.Thumbnails[p.ThumbName] = *thumb
            p.Lock.Unlock()
        }
    }

    return nil
}

// Image Metadata Extractor
type imageMetaReader struct {
    Lock      sync.Locker
    MetaData  *nested.MetaData
    WaitGroup *sync.WaitGroup
}

func (p *imageMetaReader) Process(r io.Reader) error {
    defer p.WaitGroup.Done()

    if m, err := _FileConverter.Image.Meta(r); err != nil {
        _Log.Warn(err.Error())
        return err

    } else {
        p.Lock.Lock()
        if nil == p.MetaData.Meta {
            p.MetaData.Meta = nested.MetaImage{}
        }

        v := p.MetaData.Meta.(nested.MetaImage)
        v.Width = m.Width
        v.Height = m.Height
        v.OriginalWidth = m.OriginalWidth
        v.OriginalHeight = m.OriginalHeight
        v.Orientation = m.Orientation
        p.MetaData.Meta = v

        p.Lock.Unlock()
    }

    return nil
}

// Video Metadata Extractor
type videoMetaReader struct {
    Lock      sync.Locker
    MetaData  *nested.MetaData
    WaitGroup *sync.WaitGroup
}

func (p *videoMetaReader) Process(r io.Reader) error {
    defer p.WaitGroup.Done()

    if m, err := _FileConverter.Video.Meta(r); err != nil {
        _Log.Warn(err.Error())
        return err

    } else {
        p.Lock.Lock()
        if nil == p.MetaData.Meta {
            p.MetaData.Meta = nested.MetaVideo{}
        }

        v := p.MetaData.Meta.(nested.MetaVideo)
        v.Width = m.Width
        v.Height = m.Height
        v.Duration = m.Duration
        v.AudioCodec = m.AudioCodec
        v.VideoCodec = m.VideoCodec
        p.MetaData.Meta = v

        p.Lock.Unlock()
    }

    return nil
}

// Audio Metadata Extractor
type audioMetaReader struct {
    Lock      sync.Locker
    MetaData  *nested.MetaData
    WaitGroup *sync.WaitGroup
}

func (p *audioMetaReader) Process(r io.Reader) error {
    defer p.WaitGroup.Done()

    if m, err := _FileConverter.Audio.Meta(r); err != nil {
        _Log.Warn(err.Error())
        return err

    } else {
        p.Lock.Lock()
        if nil == p.MetaData.Meta {
            p.MetaData.Meta = nested.MetaAudio{}
        }

        v := p.MetaData.Meta.(nested.MetaAudio)
        v.Duration = m.Duration
        v.AudioCodec = m.AudioCodec
        p.MetaData.Meta = v

        p.Lock.Unlock()
    }

    return nil
}

// Voice Metadata Extractor
type voiceMetaReader struct {
    Lock      sync.Locker
    MetaData  *nested.MetaData
    WaitGroup *sync.WaitGroup
}

func (p *voiceMetaReader) Process(r io.Reader) error {
    defer p.WaitGroup.Done()

    if m, err := _FileConverter.Voice.Meta(r); err != nil {
        _Log.Warn(err.Error())
        return err

    } else {
        p.Lock.Lock()
        if nil == p.MetaData.Meta {
            p.MetaData.Meta = nested.MetaVoice{}
        }

        v := p.MetaData.Meta.(nested.MetaVoice)
        v.Duration = m.Duration
        p.MetaData.Meta = v

        p.Lock.Unlock()
    }

    return nil
}

// Document Metadata Extractor
type documentMetaReader struct {
    MimeType string

    Lock      sync.Locker
    MetaData  *nested.MetaData
    WaitGroup *sync.WaitGroup
}

func (p *documentMetaReader) Process(r io.Reader) error {
    defer p.WaitGroup.Done()

    switch p.MimeType {
    case "application/pdf":
        if m, err := _FileConverter.Pdf.Meta(r); err != nil {
            _Log.Warn(err.Error())
            return err

        } else {
            p.Lock.Lock()
            p.MetaData.Meta = nested.MetaPdf{
                Width:     m.Width,
                Height:    m.Height,
                PageCount: m.PageCount,
            }
            p.Lock.Unlock()
        }
    }

    return nil
}

// GIF Metadata Extractor
type gifMetaReader struct {
    Lock      sync.Locker
    MetaData  *nested.MetaData
    WaitGroup *sync.WaitGroup
}

func (p *gifMetaReader) Process(r io.Reader) error {
    defer p.WaitGroup.Done()

    if m, err := _FileConverter.Gif.Meta(r); err != nil {
        _Log.Warn(err.Error())
        return err

    } else {
        p.Lock.Lock()
        if nil == p.MetaData.Meta {
            p.MetaData.Meta = nested.MetaGif{}
        }

        v := p.MetaData.Meta.(nested.MetaGif)
        v.Height = m.Height
        v.Width = m.Width
        v.Frames = m.Frames
        p.MetaData.Meta = v

        p.Lock.Unlock()
    }

    return protocol.NewNotImplementedError(nil)
}
