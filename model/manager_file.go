package nested

import (
    "bytes"
    "encoding/gob"
    "fmt"
    "github.com/gomodule/redigo/redis"
    "github.com/globalsign/mgo/bson"
    "path/filepath"
    "strings"
    "time"
)

// File Status
const (
    FILE_STATUS_TEMP      string = "tmp"
    FILE_STATUS_PUBLIC    string = "pub"
    FILE_STATUS_ATTACHED  string = "attached"
    FILE_STATUS_THUMBNAIL string = "thumb"
    FILE_STATUS_INTERNAL  string = "internal"
)

// File Type
const (
    FILE_TYPE_GIF       = "GIF"
    FILE_TYPE_VOICE     = "VOC"
    FILE_TYPE_IMAGE     = "IMG"
    FILE_TYPE_AUDIO     = "AUD"
    FILE_TYPE_DOCUMENT  = "DOC"
    FILE_TYPE_OTHER     = "OTH"
    FILE_TYPE_VIDEO     = "VID"
    FILE_TYPE_THUMBNAIL = "THU"
    FILE_TYPE_ALL       = "all"
)

// Upload Type
const (
    UPLOAD_TYPE_FILE            = "FILE"
    UPLOAD_TYPE_IMAGE           = "IMAGE"
    UPLOAD_TYPE_VIDEO           = "VIDEO"
    UPLOAD_TYPE_VOICE           = "VOICE"
    UPLOAD_TYPE_GIF             = "GIF"
    UPLOAD_TYPE_AUDIO           = "AUDIO"
    UPLOAD_TYPE_PLACE_PICTURE   = "PLACE_PIC"
    UPLOAD_TYPE_PROFILE_PICTURE = "PROFILE_PIC"
)

// Token
const (
    TOKEN_LIFETIME  uint64 = 86400000
    TOKEN_SEED_SALT string = "NREGS431DTED#$!!"
)

// File Sort
const (
    FILE_SORT_UPLOAD_TIME string = "upload_time"
    FILE_SORT_TIMESTAMP   string = "timestamp"
)

type SortedFilesWithPost struct {
	PostId bson.ObjectId
	File FileInfo
}

type DownloadToken struct {
    SessionKey  bson.ObjectId `json:"_sk" bson:"_sk"`
    AccountID   string        `json:"account_id" bson:"account_id"`
    UniversalID UniversalID   `json:"universal_id" bson:"universal_id"`
    ExpireTime  uint64        `json:"et" bson:"et"`
}

// FileManager
type FileManager struct{}

func NewFileManager() *FileManager {
    return new(FileManager)
}

func (fm *FileManager) readFromCache(fileID UniversalID) *FileInfo {
    _funcName := "FileManager::readFromCache"
    file := new(FileInfo)
    c := _Cache.Pool.Get()
    defer c.Close()
    keyID := fmt.Sprintf("file:gob:%s", fileID)
    if gobFile, err := redis.Bytes(c.Do("GET", keyID)); err != nil {
        if err := _MongoDB.C(COLLECTION_FILES).FindId(fileID).One(file); err != nil {
            _Log.Error(_funcName, err.Error(), fileID)
            return nil
        }
        gobFile := new(bytes.Buffer)
        if err := gob.NewEncoder(gobFile).Encode(file); err == nil {
            c.Do("SETEX", keyID, CACHE_LIFETIME, gobFile.Bytes())
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
    _funcName := "FileManager::AddFile"
    _Log.FunctionStarted(_funcName, f)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()


    if f.Filename == "" {
        return false
    }
    if err := db.C(COLLECTION_FILES).Insert(f); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return true
}

func (fm *FileManager) AddPostAsOwner(uniID UniversalID, postID bson.ObjectId) {
    _funcName := "FileManager::AddPostAsOwner"
    _Log.FunctionStarted(_funcName, uniID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_FILES).UpdateId(
        uniID,
        bson.M{"$inc": bson.M{"ref_count": 1}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
    }

    if err := db.C(COLLECTION_POSTS_FILES).Insert(
        bson.M{
            "universal_id": uniID,
            "post_id":      postID,
        },
    ); err != nil {
        _Log.Error(_funcName, err.Error())
    }
}

func (fm *FileManager) AddTaskAsOwner(uniID UniversalID, taskID bson.ObjectId) {
    _funcName := "FileManager::AddTaskAsOwner"
    _Log.FunctionStarted(_funcName, uniID, taskID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_FILES).UpdateId(
        uniID,
        bson.M{"$inc": bson.M{"ref_count": 1}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
    }

    if err := _MongoDB.C(COLLECTION_TASKS_FILES).Insert(
        bson.M{
            "universal_id": uniID,
            "task_id":      taskID,
        },
    ); err != nil {
        _Log.Error(_funcName, err.Error())
    }

}

// Exists
// check if universalID exists
func (fm *FileManager) Exists(uniID UniversalID) bool {
    _funcName := "FileManager::Exists"
    _Log.FunctionStarted(_funcName, uniID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n, _ := db.C(COLLECTION_FILES).FindId(uniID).Count()

    return n > 0
}

// SetStatus
// Updates the status of the universalID
func (fm *FileManager) SetStatus(uniID UniversalID, fileStatus string) bool {
    _funcName := "FileManager::SetStatus"
    _Log.FunctionStarted(_funcName, uniID, fileStatus)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    switch fileStatus {
    case FILE_STATUS_PUBLIC, FILE_STATUS_TEMP, FILE_STATUS_THUMBNAIL:
        if err := db.C(COLLECTION_FILES).UpdateId(
            uniID,
            bson.M{"$set": bson.M{
                "upload_time": time.Now().UnixNano(),
                "status":      fileStatus,
            }},
        ); err != nil {
            _Log.Error(_funcName, err.Error(), uniID, fileStatus)
        }
    case FILE_STATUS_ATTACHED:
        if err := db.C(COLLECTION_FILES).Update(
            bson.M{"_id": uniID, "status": bson.M{"$ne": FILE_STATUS_PUBLIC}},
            bson.M{"$set": bson.M{"status": fileStatus}},
        ); err != nil {
            _Log.Error(_funcName, err.Error(), uniID, fileStatus)
        }
    case FILE_STATUS_INTERNAL:
        if err := db.C(COLLECTION_FILES).Update(
            bson.M{"_id": uniID},
            bson.M{"$set": bson.M{"status": fileStatus}},
        ); err != nil {
            _Log.Error(_funcName, err.Error(), uniID, fileStatus)
        }
    default:
        return false

    }
    return true
}

// Description:
// Set
func (fm *FileManager) SetMetadata(uniID UniversalID, meta interface{}) {
    _funcName := "FileManager::SetMetadata"
    _Log.FunctionStarted(_funcName, uniID, meta)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_FILES).Update(
        bson.M{"_id": uniID},
        bson.M{"$set": bson.M{"metadata": meta}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
    }
}

// Description:
// Get file's type
func (fm *FileManager) GetType(filename string) (ft string) {
    _funcName := "FileManager::GetType"
    _Log.FunctionStarted(_funcName, filename)
    defer _Log.FunctionFinished(_funcName)

    ext := strings.ToLower(filepath.Ext(filename))
    switch ext {
    case "bmp", "jpg", "jpeg", "gif", "jpe", "ief", "png":
        ft = FILE_TYPE_IMAGE
    case "aac", "mp1", "mp2", "mp3", "mpg", "wma", "m4a", "ogg", "oga":
        ft = FILE_TYPE_AUDIO
    case "doc", "docx", "xls", "xlsx", "pdf":
        ft = FILE_TYPE_DOCUMENT
    case "mp4", "m4v", "3gp", "ogv", "webm", "mov":
        ft = FILE_TYPE_VIDEO
    default:
        ft = FILE_TYPE_OTHER
    }
    return
}

// Description:
// Get file by universalID and returns only keys identified by "pj", if pj is set to NIL then
// it returns all the keys of the document
func (fm *FileManager) GetByID(uniID UniversalID, pj M) *FileInfo {
    _funcName := "FileManager::GetByID"
    _Log.FunctionStarted(_funcName, uniID, pj)
    defer _Log.FunctionFinished(_funcName)

    return _Manager.File.readFromCache(uniID)
}

// Description:
//	Get files by universalIDs
func (fm *FileManager) GetFilesByIDs(uniIDs []UniversalID) (files []FileInfo) {
    _funcName := "FileManager::GetFilesByIDs"
    _Log.FunctionStarted(_funcName, uniIDs)
    defer _Log.FunctionFinished(_funcName)

    return _Manager.File.readMultiFromCache(uniIDs)
}

// Description:
// Get files by placeID and types are filtered by "filter" and name is filtered by "filename"
// "pg" is also used for pagination
func (fm *FileManager) GetFilesByPlace(placeID, filter, filename string, pg Pagination) (sortedList []SortedFilesWithPost) {
	_funcName := "FileManager::GetFilesByPlace"
	_Log.FunctionStarted(_funcName, placeID, filter, filename)
	defer _Log.FunctionFinished(_funcName)

	dbSession := _MongoSession.Copy()
	db := dbSession.DB(DB_NAME)
	defer dbSession.Close()

	q := []bson.M{
		{"$unwind": bson.M{
			"path": "$attaches",
		}},
		{"$match": bson.M{"places": placeID, "counters.attaches": bson.M{"$gt": 0}}},
		{"$lookup": bson.M{
			"from":         COLLECTION_FILES,
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
	iter := db.C(COLLECTION_POSTS).Pipe(q).Iter()
	defer iter.Close()
	fetchedDoc := struct {
		PostID bson.ObjectId `bson:"_id"`
		Files  []FileInfo    `bson:"file"`
	}{}
	totalAttachments := 0
	for iter.Next(&fetchedDoc) {
		for _, file := range fetchedDoc.Files {
			switch filter {
			case FILE_TYPE_AUDIO, FILE_TYPE_DOCUMENT, FILE_TYPE_IMAGE, FILE_TYPE_VIDEO, FILE_TYPE_OTHER:
				if strings.HasPrefix(string(file.ID), filter) {
					totalAttachments++
					sortedList = append(sortedList, SortedFilesWithPost{
						PostId: fetchedDoc.PostID,
						File: file,
					})
				}
			default:
				totalAttachments++
				sortedList = append(sortedList, SortedFilesWithPost{
					PostId: fetchedDoc.PostID,
					File: file,
				})
			}
		}
		if totalAttachments >= pg.GetLimit() {
			break
		}
	}
	return
}

// Description:
//	Get files which are in placeIDs and types are filtered by "filter" and name is filtered by "filename"
//	"pg" is also used for pagination
func (fm *FileManager) GetFilesByPlaces(placeIDs []string, pg Pagination) []FileInfo {
    _funcName := "FileManager::GetFilesByPlace"
    _Log.FunctionStarted(_funcName, placeIDs)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    post := new(Post)
    attachmentIDs := make([]UniversalID, 0, pg.GetLimit())
    iter := db.C(COLLECTION_POSTS).Find(
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
    _funcName := "FileManager::RemoveTaskAsOwner"
    _Log.FunctionStarted(_funcName, uniID, taskID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_FILES).UpdateId(
        uniID,
        bson.M{"$inc": bson.M{"ref_count": -1}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
    }

    if err := db.C(COLLECTION_TASKS_FILES).Remove(
        bson.M{"universal_id": uniID, "task_id": taskID},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return
}

// RemoveTaskAsOwner removes the connection between uniID and fileID
func (fm *FileManager) RemovePostAsOwner(uniID UniversalID, postID bson.ObjectId) {
    _funcName := "FileManager::RemoveTaskAsOwner"
    _Log.FunctionStarted(_funcName, uniID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_FILES).UpdateId(
        uniID,
        bson.M{"$inc": bson.M{"ref_count": -1}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
    }

    if err := db.C(COLLECTION_POSTS_FILES).Remove(
        bson.M{"universal_id": uniID, "post_id": postID},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return
}

// Description:
// Update a file dimensions
func (fm *FileManager) SetDimension(uniID UniversalID, width, height int64) bool {
    _funcName := "FileManager::SetDimension"
    _Log.FunctionStarted(_funcName, uniID, width, height)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_FILES).UpdateId(
        uniID,
        bson.M{"$set": bson.M{
            "width":  width,
            "height": height,
        }},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }

    return true
}

// Description:
// Set a file thumbnails
func (fm *FileManager) SetThumbnails(uniID UniversalID, thumbs Picture) bool {
    _funcName := "FileManager::SetThumbnails"
    _Log.FunctionStarted(_funcName, uniID, thumbs)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_FILES).UpdateId(
        uniID,
        bson.M{"$set": bson.M{
            "thumbs": thumbs,
        }},
    ); err != nil {
        _Log.Error(_funcName, err.Error(), uniID)
        return false
    }
    return true
}

// Description:
// Increments the download counter of the file identified by "uniID"
func (fm *FileManager) IncrementDownloadCounter(uniID UniversalID, count int) bool {
    _funcName := "FileManager::IncrementDownloadCounter"
    _Log.FunctionStarted(_funcName, uniID, count)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_FILES).UpdateId(uniID, bson.M{"$inc": bson.M{"downloads": count}}); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }

    return true
}

// IsTaskOwner returns true if "taskID" is one of the owners of the file identified by "uniID"
func (fm *FileManager) IsTaskOwner(uniID UniversalID, taskID bson.ObjectId) bool {
    _funcName := "FileManager::IsTaskOwner"
    _Log.FunctionStarted(_funcName, uniID, taskID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if n, err := db.C(COLLECTION_TASKS_FILES).Find(
        bson.M{"universal_id": uniID, "task_id": taskID},
    ).Count(); err != nil || n == 0 {
        return false
    }
    return true

}

// IsPostOwner returns true if "postID" is one of the owners of the file identified by "uniID"
func (fm *FileManager) IsPostOwner(uniID UniversalID, postID bson.ObjectId) bool {
    _funcName := "FileManager::IsPostOwner"
    _Log.FunctionStarted(_funcName, uniID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if n, err := db.C(COLLECTION_POSTS_FILES).Find(
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
    return f.Status == FILE_STATUS_PUBLIC || f.Status == FILE_STATUS_THUMBNAIL
}
