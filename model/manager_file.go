package nested

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"path/filepath"
	"strings"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/gomodule/redigo/redis"
)

// File Status
const (
	FileStatusTemp      string = "tmp"
	FileStatusPublic    string = "pub"
	FileStatusAttached  string = "attached"
	FileStatusThumbnail string = "thumb"
	FileStatusInternal  string = "internal"
)

// File Type
const (
	FileTypeGif       = "GIF"
	FileTypeVoice     = "VOC"
	FileTypeImage     = "IMG"
	FileTypeAudio     = "AUD"
	FileTypeDocument  = "DOC"
	FileTypeOther     = "OTH"
	FileTypeVideo     = "VID"
	FileTypeThumbnail = "THU"
	FileTypeAll       = "all"
)

// Upload Type
const (
	UploadTypeFile    = "FILE"
	UploadTypeImage   = "IMAGE"
	UploadTypeVideo = "VIDEO"
	UploadTypeVoice   = "VOICE"
	UploadTypeGif             = "GIF"
	UploadTypeAudio             = "AUDIO"
	UploadTypePlacePicture   = "PLACE_PIC"
	UploadTypeProfilePicture = "PROFILE_PIC"
)

// Token
const (
	TokenLifetime uint64 = 86400000
	TokenSeedSalt string = "NREGS431DTED#$!!"
)

type SortedFilesWithPost struct {
	PostId bson.ObjectId
	File   FileInfo
}

type DownloadToken struct {
	SessionKey  bson.ObjectId `json:"_sk" bson:"_sk"`
	AccountID   string        `json:"account_id" bson:"account_id"`
	UniversalID UniversalID   `json:"universal_id" bson:"universal_id"`
	ExpireTime  uint64        `json:"et" bson:"et"`
}

type FileManager struct{}

func NewFileManager() *FileManager {
	return new(FileManager)
}

func (fm *FileManager) readFromCache(fileID UniversalID) *FileInfo {
	file := new(FileInfo)
	c := _Cache.Pool.Get()
	defer c.Close()
	keyID := fmt.Sprintf("file:gob:%s", fileID)
	if gobFile, err := redis.Bytes(c.Do("GET", keyID)); err != nil {
		if err := _MongoDB.C(global.COLLECTION_FILES).FindId(fileID).One(file); err != nil {
			log.Warn(err.Error())
			return nil
		}
		gobFile := new(bytes.Buffer)
		if err := gob.NewEncoder(gobFile).Encode(file); err == nil {
			c.Do("SETEX", keyID, global.CacheLifetime, gobFile.Bytes())
		}
		return file
	} else if err := gob.NewDecoder(bytes.NewBuffer(gobFile)).Decode(file); err == nil {
		return file
	}
	return nil
}

func (fm *FileManager) readMultiFromCache(fileIDs []UniversalID) []FileInfo {
	files := make([]FileInfo, 0, len(fileIDs))
	c := _Cache.Pool.Get()
	defer c.Close()
	for _, fileID := range fileIDs {
		keyID := fmt.Sprintf("file:gob:%s", fileID)
		c.Send("GET", keyID)
	}
	c.Flush()
	for _, fileID := range fileIDs {
		if gobFile, err := redis.Bytes(c.Receive()); err == nil {
			file := new(FileInfo)
			if err := gob.NewDecoder(bytes.NewBuffer(gobFile)).Decode(file); err == nil {
				files = append(files, *file)
			}
		} else {
			if file := _Manager.File.readFromCache(fileID); file != nil {
				files = append(files, *file)
			}
		}
	}
	return files
}

func (fm *FileManager) removeCache(fileID UniversalID) bool {
	c := _Cache.Pool.Get()
	defer c.Close()
	keyID := fmt.Sprintf("file:gob:%s", fileID)
	c.Do("DEL", keyID)
	return true
}

func (fm *FileManager) AddFile(f FileInfo) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if f.Filename == "" {
		return false
	}
	if err := db.C(global.COLLECTION_FILES).Insert(f); err != nil {
		log.Warn(err.Error())
	}
	return true
}

func (fm *FileManager) AddPostAsOwner(uniID UniversalID, postID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_FILES).UpdateId(
		uniID,
		bson.M{"$inc": bson.M{"ref_count": 1}},
	); err != nil {
		log.Warn(err.Error())
	}

	if err := db.C(global.COLLECTION_POSTS_FILES).Insert(
		bson.M{
			"universal_id": uniID,
			"post_id":      postID,
		},
	); err != nil {
		log.Warn(err.Error())
	}
}

func (fm *FileManager) AddTaskAsOwner(uniID UniversalID, taskID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_FILES).UpdateId(
		uniID,
		bson.M{"$inc": bson.M{"ref_count": 1}},
	); err != nil {
		log.Warn(err.Error())
	}

	if err := _MongoDB.C(global.COLLECTION_TASKS_FILES).Insert(
		bson.M{
			"universal_id": uniID,
			"task_id":      taskID,
		},
	); err != nil {
		log.Warn(err.Error())
	}

}

// Exists
// check if universalID exists
func (fm *FileManager) Exists(uniID UniversalID) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n, _ := db.C(global.COLLECTION_FILES).FindId(uniID).Count()

	return n > 0
}

// SetStatus
// Updates the status of the universalID
func (fm *FileManager) SetStatus(uniID UniversalID, fileStatus string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	switch fileStatus {
	case FileStatusPublic, FileStatusTemp, FileStatusThumbnail:
		if err := db.C(global.COLLECTION_FILES).UpdateId(
			uniID,
			bson.M{"$set": bson.M{
				"upload_time": time.Now().UnixNano(),
				"status":      fileStatus,
			}},
		); err != nil {
			log.Warn(err.Error())
		}
	case FileStatusAttached:
		if err := db.C(global.COLLECTION_FILES).Update(
			bson.M{"_id": uniID, "status": bson.M{"$ne": FileStatusPublic}},
			bson.M{"$set": bson.M{"status": fileStatus}},
		); err != nil {
			log.Warn(err.Error())
		}
	case FileStatusInternal:
		if err := db.C(global.COLLECTION_FILES).Update(
			bson.M{"_id": uniID},
			bson.M{"$set": bson.M{"status": fileStatus}},
		); err != nil {
			log.Warn(err.Error())
		}
	default:
		return false

	}
	return true
}

func (fm *FileManager) SetMetadata(uniID UniversalID, meta interface{}) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_FILES).Update(
		bson.M{"_id": uniID},
		bson.M{"$set": bson.M{"metadata": meta}},
	); err != nil {
		log.Warn(err.Error())
	}
}

func (fm *FileManager) GetType(filename string) (ft string) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case "bmp", "jpg", "jpeg", "gif", "jpe", "ief", "png":
		ft = FileTypeImage
	case "aac", "mp1", "mp2", "mp3", "mpg", "wma", "m4a", "ogg", "oga":
		ft = FileTypeAudio
	case "doc", "docx", "xls", "xlsx", "pdf":
		ft = FileTypeDocument
	case "mp4", "m4v", "3gp", "ogv", "webm", "mov":
		ft = FileTypeVideo
	default:
		ft = FileTypeOther
	}
	return
}

func (fm *FileManager) GetByID(uniID UniversalID, pj tools.M) *FileInfo {
	return _Manager.File.readFromCache(uniID)
}

func (fm *FileManager) GetFilesByIDs(uniIDs []UniversalID) (files []FileInfo) {
	return _Manager.File.readMultiFromCache(uniIDs)
}

func (fm *FileManager) GetFilesByPlace(placeID, filter, filename string, pg Pagination) (sortedList []SortedFilesWithPost) {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	q := []bson.M{
		{"$unwind": bson.M{
			"path": "$attaches",
		}},
		{"$match": bson.M{"places": placeID, "counters.attaches": bson.M{"$gt": 0}}},
		{"$lookup": bson.M{
			"from":         global.COLLECTION_FILES,
			"localField":   "attaches",
			"foreignField": "_id",
			"as":           "file",
		}},
		{"$match": bson.M{
			"file": bson.M{
				"$elemMatch": bson.M{
					"filename": bson.M{"$regex": fmt.Sprintf("%s", filename), "$options": "i"},
				},
			},
		}},
		{"$sort": bson.M{"timestamp": -1}},
		{"$skip": pg.GetSkip()},
		{"$limit": pg.GetLimit()},
	}
	iter := db.C(global.COLLECTION_POSTS).Pipe(q).Iter()
	defer iter.Close()
	fetchedDoc := struct {
		PostID bson.ObjectId `bson:"_id"`
		Files  []FileInfo    `bson:"file"`
	}{}
	totalAttachments := 0
	for iter.Next(&fetchedDoc) {
		for _, file := range fetchedDoc.Files {
			switch filter {
			case FileTypeAudio, FileTypeDocument, FileTypeImage, FileTypeVideo, FileTypeOther:
				if strings.HasPrefix(string(file.ID), filter) {
					totalAttachments++
					sortedList = append(sortedList, SortedFilesWithPost{
						PostId: fetchedDoc.PostID,
						File:   file,
					})
				}
			default:
				totalAttachments++
				sortedList = append(sortedList, SortedFilesWithPost{
					PostId: fetchedDoc.PostID,
					File:   file,
				})
			}
		}
		if totalAttachments >= pg.GetLimit() {
			break
		}
	}
	return
}

func (fm *FileManager) GetFilesByPlaces(placeIDs []string, pg Pagination) []FileInfo {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	post := new(Post)
	attachmentIDs := make([]UniversalID, 0, pg.GetLimit())
	iter := db.C(global.COLLECTION_POSTS).Find(
		bson.M{"places": bson.M{"$in": placeIDs}, "counters.attaches": bson.M{"$gt": 0}},
	).Sort("-timestamp").Skip(pg.GetSkip()).Iter()
	defer iter.Close()
	for iter.Next(post) {
		for _, fileID := range post.AttachmentIDs {
			attachmentIDs = append(attachmentIDs, fileID)
		}
		if len(attachmentIDs) >= pg.GetLimit() {
			break
		}
	}

	return _Manager.File.GetFilesByIDs(attachmentIDs)
}

// RemoveTaskAsOwner removes the connection between uniID and fileID
func (fm *FileManager) RemoveTaskAsOwner(uniID UniversalID, taskID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_FILES).UpdateId(
		uniID,
		bson.M{"$inc": bson.M{"ref_count": -1}},
	); err != nil {
		log.Warn(err.Error())
	}

	if err := db.C(global.COLLECTION_TASKS_FILES).Remove(
		bson.M{"universal_id": uniID, "task_id": taskID},
	); err != nil {
		log.Warn(err.Error())
	}
	return
}

// RemovePostAsOwner removes the file from post
func (fm *FileManager) RemovePostAsOwner(uniID UniversalID, postID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_FILES).UpdateId(
		uniID,
		bson.M{"$inc": bson.M{"ref_count": -1}},
	); err != nil {
		log.Warn(err.Error())
	}

	if err := db.C(global.COLLECTION_POSTS_FILES).Remove(
		bson.M{"universal_id": uniID, "post_id": postID},
	); err != nil {
		log.Warn(err.Error())
	}
	return
}

func (fm *FileManager) SetDimension(uniID UniversalID, width, height int64) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_FILES).UpdateId(
		uniID,
		bson.M{"$set": bson.M{
			"width":  width,
			"height": height,
		}},
	); err != nil {
		log.Warn(err.Error())
		return false
	}

	return true
}

func (fm *FileManager) SetThumbnails(uniID UniversalID, thumbs Picture) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_FILES).UpdateId(
		uniID,
		bson.M{"$set": bson.M{
			"thumbs": thumbs,
		}},
	); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

// IncrementDownloadCounter Increments the download counter of the file identified by "uniID"
func (fm *FileManager) IncrementDownloadCounter(uniID UniversalID, count int) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_FILES).UpdateId(uniID, bson.M{"$inc": bson.M{"downloads": count}}); err != nil {
		log.Warn(err.Error())
		return false
	}

	return true
}

// IsTaskOwner returns true if "taskID" is one of the owners of the file identified by "uniID"
func (fm *FileManager) IsTaskOwner(uniID UniversalID, taskID bson.ObjectId) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if n, err := db.C(global.COLLECTION_TASKS_FILES).Find(
		bson.M{"universal_id": uniID, "task_id": taskID},
	).Count(); err != nil || n == 0 {
		return false
	}
	return true

}

// IsPostOwner returns true if "postID" is one of the owners of the file identified by "uniID"
func (fm *FileManager) IsPostOwner(uniID UniversalID, postID bson.ObjectId) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if n, err := db.C(global.COLLECTION_POSTS_FILES).Find(
		bson.M{"universal_id": uniID, "post_id": postID},
	).Count(); err != nil || n == 0 {
		return false
	}
	return true

}

/*
	FileInfo Methods
*/

type FileInfo struct {
	ID              UniversalID `json:"_id" bson:"_id"`
	Size            int64       `json:"size,omitempty" bson:"size"`
	Type            string      `json:"type,omitempty" bson:"type"`
	UploadType      string      `json:"upload_type" bson:"upload_type"`
	Status          string      `json:"status,omitempty" bson:"status"`
	Filename        string      `json:"filename" bson:"filename"`
	MimeType        string      `json:"mimetype" bson:"mimetype"`
	Downloads       int64       `json:"downloads,omitempty" bson:"downloads"`
	Thumbnails      Picture     `json:"thumbs" bson:"thumbs"`
	UploaderId      string      `json:"uploader,omitempty" bson:"uploader"`
	UploadTimestamp uint64      `json:"upload_time,omitempty" bson:"upload_time"`
	Width           int64       `json:"width,omitempty" bson:"width,omitempty"`
	Height          int64       `json:"height,omitempty" bson:"height,omitempty"`
	Metadata        interface{} `json:"metadata" bson:"metadata"`
}

func (f *FileInfo) IsPublic() bool {
	return f.Status == FileStatusPublic || f.Status == FileStatusThumbnail
}
