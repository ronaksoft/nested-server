package nested

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"go.uber.org/zap"
	"io"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type MetaData struct {
	Meta       interface{} `json:"meta" bson:"meta"`
	Type       string      `json:"type" bson:"type"`
	Uploader   string      `json:"uploader" bson:"uploader"`
	Thumbnails Thumbnails  `json:"thumbnails" bson:"thumbnails"`
}

// StoredFileInfo Database file's information which is stored/read to/from database
type StoredFileInfo struct {
	ID         UniversalID `json:"id" bson:"_id"`
	Hash       string      `json:"hash" bson:"md5"`
	Name       string      `json:"name" bson:"filename"`
	Size       int64       `json:"size" bson:"length"`
	Metadata   MetaData    `json:"metadata" bson:"metadata"`
	MimeType   string      `json:"mimetype" bson:"contentType"`
	UploadDate time.Time   `json:"upload_date" bson:"uploadDate"`
}

// MetaImage File's metadata which is used when file type is FileTypeImage
type MetaImage struct {
	Width          int64 `json:"width" bson:"width"`
	Height         int64 `json:"height" bson:"height"`
	OriginalWidth  int64 `json:"original_width"`
	OriginalHeight int64 `json:"original_height"`
	Orientation    int   `json:"orientation"`
}

// MetaVideo File's metadata which is used when file type is FileTypeVideo
type MetaVideo struct {
	Width      int64         `json:"width" bson:"width"`
	Height     int64         `json:"height" bson:"height"`
	Duration   time.Duration `json:"duration" bson:"duration"`
	VideoCodec string        `json:"video_codec" bson:"video_codec"`
	AudioCodec string        `json:"audio_codec" bson:"audio_codec"`
}

// MetaAudio File's metadata which is used when file type is FileTypeAudio
type MetaAudio struct {
	Duration   time.Duration `json:"duration" bson:"duration"`
	AudioCodec string        `json:"audio_codec" bson:"audio_codec"`
}

// MetaGif File's metadata which is used when file type is FileTypeGif
type MetaGif struct {
	Width  int64 `json:"width" bson:"width"`
	Height int64 `json:"height" bson:"height"`
	Frames uint  `json:"frames" bson:"frames"`
}

// MetaVoice File's metadata which is used when file type is FileTypeVoice
type MetaVoice struct {
	Samples    []uint8       `json:"samples" bson:"samples"`
	Duration   time.Duration `json:"duration" bson:"duration"`
	SampleRate uint8         `json:"sample_rate" bson:"sample_rate"`
}

// MetaDocument File's metadata which is used when file type is FileTypeDocument
type MetaDocument struct {
	PageCount int `json:"page_count" bson:"page_count"`
}

// MetaPdf File's metadata which is used when file type is FileTypeDocument and document is pdf
type MetaPdf struct {
	Width     float32 `json:"width" bson:"width"`
	Height    float32 `json:"height" bson:"height"`
	PageCount int     `json:"page_count" bson:"page_count"`
}

var Types = map[string]bool{
	FileTypeGif:       true,
	FileTypeAudio:     true,
	FileTypeImage:     true,
	FileTypeVideo:     true,
	FileTypeVoice:     true,
	FileTypeOther:     true,
	FileTypeDocument:  true,
	FileTypeThumbnail: true,
}

type StoreManager struct {
	m *Manager
}

func newStoreManager() *StoreManager {
	return new(StoreManager)
}

// GenerateFileInfo returns a FileInfo structure based on input parameters
func GenerateFileInfo(filename, uploader string, fileType string, meta interface{}) StoredFileInfo {
	return StoredFileInfo{
		ID:       GenerateUniversalID(filename, fileType),
		Name:     filename,
		MimeType: GetMimeTypeByFilename(filename),
		Metadata: MetaData{
			Meta:     meta,
			Uploader: uploader,
			Type:     GetTypeByFilename(filename),
		},
	}
}

// Save inserts file into database
func (fm *StoreManager) Save(r io.Reader, fileInfo StoredFileInfo) *StoredFileInfo {
	dbSession := _MongoSession.Copy()
	store := dbSession.DB(global.StoreName).GridFS("fs")
	defer dbSession.Close()

	var gFile *mgo.GridFile
	if gf, err := store.Create(fileInfo.Name); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	} else {
		gFile = gf
		defer gFile.Close()
	}

	gFile.SetId(fileInfo.ID)
	gFile.SetMeta(fileInfo.Metadata)
	gFile.SetContentType(fileInfo.MimeType)

	if n, err := io.Copy(gFile, r); err != nil {
		log.Warn("Got error", zap.Error(err))
		gFile.Abort()
		return nil
	} else if 0 == n {
		log.Warn(fmt.Sprintf("save empty file %s", fileInfo.ID))
		gFile.Abort()
		return nil
	} else {
		fileInfo.Size = n
		fileInfo.UploadDate = gFile.UploadDate()
		fileInfo.Hash = gFile.MD5()
	}

	return &fileInfo
}

// SetThumbnails sets a file's thumbnails map in file info
func (fm *StoreManager) SetThumbnails(uniID UniversalID, thumbnails Thumbnails) error {
	dbSession := _MongoSession.Clone()
	store := dbSession.DB(global.StoreName).GridFS("fs")
	defer dbSession.Close()

	if err := store.Files.UpdateId(uniID, bson.M{"$set": bson.M{"metadata.thumbnails": thumbnails}}); err != nil {
		log.Warn("Got error", zap.Error(err))
		return err
	}
	return nil
}

// SetMeta sets a file's meta object in file info
func (fm *StoreManager) SetMeta(uniID UniversalID, meta interface{}) error {
	dbSession := _MongoSession.Clone()
	store := dbSession.DB(global.StoreName).GridFS("fs")
	defer dbSession.Close()

	if err := store.Files.UpdateId(uniID, bson.M{"$set": bson.M{"metadata.meta": meta}}); err != nil {
		log.Warn("Got error", zap.Error(err))
		return err
	}
	return nil
}

// Exists checks if file exists
func (fm *StoreManager) Exists(uniID UniversalID) bool {
	dbSession := _MongoSession.Clone()
	store := dbSession.DB(global.StoreName).GridFS("fs")
	defer dbSession.Close()

	if c, err := store.Files.FindId(uniID).Count(); err != nil || 0 == c {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

// GetFile retrieves file content from database
func (fm *StoreManager) GetFile(uniID UniversalID) (*mgo.GridFile, error) {
	return _MongoStore.OpenId(uniID)
}

// GetInfo retrieves file information from database
func (fm *StoreManager) GetInfo(uniID UniversalID) (*StoredFileInfo, error) {
	finfo := new(StoredFileInfo)

	dbSession := _MongoSession.Clone()
	store := dbSession.DB(global.StoreName).GridFS("fs")
	defer dbSession.Close()

	if err := store.Files.FindId(uniID).One(finfo); err != nil {
		return nil, err
	}

	return finfo, nil
}

type Thumbnails map[string]StoredFileInfo

func (t Thumbnails) Get(size string) *StoredFileInfo {
	if v, ok := t[size]; ok {
		return &v
	}
	return nil
}

func (t *Thumbnails) Set(name string, info StoredFileInfo) {
	if nil == t {
		*t = make(map[string]StoredFileInfo)
	}
	(*t)[name] = info
}
