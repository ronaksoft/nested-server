package file

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"git.ronaksoftware.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoftware.com/nested/server/model"
	"git.ronaksoftware.com/nested/server/pkg/protocol"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kataras/iris"
)

func (fs *Server) ForceDownload(ctx iris.Context) {
	ctx.Values().Set("forceDownload", true)
	ctx.Next()
}
func (fs *Server) ServeFileByFileToken(ctx iris.Context) {
	fileToken := ctx.Params().Get("fileToken")
	resp := new(nestedGateway.Response)
	if v, err := _NestedModel.Token.GetFileByToken(fileToken); err != nil {
		ctx.StatusCode(http.StatusUnauthorized)
		resp.Error(nested.ERR_INVALID, []string{"fileToken"})
		ctx.JSON(resp)
		return
	} else {
		ctx.Params().Set("universalID", string(v))
	}

	// Go to next handler
	ctx.Next()
}
func (fs *Server) ServePublicFiles(ctx iris.Context) {
	var fileInfo *nested.FileInfo
	universalID := nested.UniversalID(ctx.Params().Get("universalID"))

	resp := new(nestedGateway.Response)
	if v := _NestedModel.File.GetByID(universalID, nil); v == nil {
		ctx.StatusCode(http.StatusUnauthorized)
		resp.Error(nested.ERR_INVALID, []string{"universal_id"})
		ctx.JSON(resp)
		return
	} else {
		fileInfo = v
	}
	switch fileInfo.Status {
	case nested.FILE_STATUS_PUBLIC, nested.FILE_STATUS_THUMBNAIL:
	default:
		ctx.StatusCode(http.StatusUnauthorized)
		resp.Error(nested.ERR_ACCESS, []string{})
		ctx.JSON(resp)
		return
	}

	// Go to next handler
	ctx.Next()
}
func (fs *Server) ServePrivateFiles(ctx iris.Context) {
	universalID := nested.UniversalID(ctx.Params().Get("universalID"))
	//sessionID := ctx.Params().Get("sessionID")
	downloadToken := ctx.Params().Get("downloadToken")
	resp := new(nestedGateway.Response)

	//if !bson.IsObjectIdHex(sessionID) {
	//    ctx.StatusCode(http.StatusUnauthorized)
	//    resp.Error(nested.ERR_ACCESS, []string{})
	//    ctx.JSON(resp)
	//    return
	//} else {
	//    session := _NestedModel.Session.GetByID(bson.ObjectIdHex(sessionID))
	//    if session == nil {
	//        ctx.StatusCode(http.StatusUnauthorized)
	//        resp.Error(nested.ERR_ACCESS, []string{})
	//        ctx.JSON(resp)
	//        return
	//    }
	//}

	if valid, uniID := nested.UseDownloadToken(downloadToken); !valid {
		ctx.StatusCode(http.StatusUnauthorized)
		resp.Error(nested.ERR_ACCESS, []string{})
		ctx.JSON(resp)
		return
	} else if universalID != uniID {
		ctx.StatusCode(http.StatusUnauthorized)
		resp.Error(nested.ERR_ACCESS, []string{})
		ctx.JSON(resp)
		return
	}

	// Go to next handler
	ctx.Next()
}
func (fs *Server) ServerFileBySystem(ctx iris.Context) {
	apiKey := ctx.Params().Get("apiKey")
	resp := new(nestedGateway.Response)
	if apiKey != fs.apiKey {
		ctx.StatusCode(http.StatusUnauthorized)
		resp.Error(nested.ERR_ACCESS, []string{})
		ctx.JSON(resp)
		return
	}

	// Go to next handler
	ctx.Next()
}
func (fs *Server) Download(ctx iris.Context) {
	var fileInfo *nested.FileInfo
	var file *mgo.GridFile
	resp := new(nestedGateway.Response)

	universalID := nested.UniversalID(ctx.Params().Get("universalID"))
	forceDownload, _ := ctx.Values().GetBool("forceDownload")
	if v := _NestedModel.File.GetByID(universalID, nil); v == nil {
		ctx.StatusCode(http.StatusUnauthorized)
		resp.Error(nested.ERR_INVALID, []string{"universal_id"})
		ctx.JSON(resp)
		return
	} else {
		fileInfo = v
	}

	if v, err := _NestedModel.Store.GetFile(universalID); err != nil {
		ctx.StatusCode(http.StatusExpectationFailed)
		resp.Error(nested.ERR_UNAVAILABLE, []string{"universal_id"})
		ctx.JSON(resp)
		return
	} else {
		file = v
		defer file.Close()
	}

	// Increment the download counter of the file
	_NestedModel.File.IncrementDownloadCounter(universalID, 1)

	if downloadRange := ctx.Request().Header.Get("Range"); downloadRange != "" {
		http.ServeContent(ctx.ResponseWriter(), ctx.Request(), fileInfo.Filename, time.Unix(int64(fileInfo.UploadTimestamp/1000), 0), file)
	} else {
		ctx.Header("Content-Type", fmt.Sprintf("%s: charset=UTF-8", fileInfo.MimeType))
		if forceDownload {
			ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileInfo.Filename))
		}
		if fs.compressed {
			ctx.Header("Content-Encoding", "gzip")
		} else {
			ctx.Header("Content-Length", fmt.Sprintf("%d", file.Size()))
		}
		ctx.ServeContent(file, fileInfo.Filename, time.Unix(int64(fileInfo.UploadTimestamp/1000), 0), fs.compressed)
	}
}

func (fs *Server) UploadSystem(ctx iris.Context) {
	var multipartReader *multipart.Reader
	apiKey := ctx.Params().Get("apiKey")
	resp := new(nestedGateway.Response)
	if apiKey != fs.apiKey {
		ctx.StatusCode(http.StatusUnauthorized)
		resp.Error(nested.ERR_ACCESS, []string{})
		ctx.JSON(resp)
	}

	uploaderID := apiKey
	uploadType := strings.ToUpper(ctx.Params().Get("uploadType"))
	if r, err := ctx.Request().MultipartReader(); err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		resp.Error(nested.ERR_INVALID, []string{"request"})
		ctx.JSON(resp)
		return
	} else {
		multipartReader = r
	}
	switch uploadType {
	case nested.UPLOAD_TYPE_FILE, nested.UPLOAD_TYPE_GIF, nested.UPLOAD_TYPE_VOICE,
		nested.UPLOAD_TYPE_AUDIO, nested.UPLOAD_TYPE_IMAGE, nested.UPLOAD_TYPE_VIDEO:
		var fileInfos []nested.FileInfo
		for part, err := multipartReader.NextPart(); nil == err; part, err = multipartReader.NextPart() {
			if fileInfo, err := uploadFile(part, uploadType, uploaderID, false); err != nil {
				resp.Error(nested.ERR_UNKNOWN, []string{})
				ctx.StatusCode(http.StatusExpectationFailed)
				ctx.JSON(resp)
				return

			} else {
				fileInfos = append(fileInfos, *fileInfo)
			}
		}
		r := make([]nestedGateway.UploadedFile, 0, len(fileInfos))
		for _, fileInfo := range fileInfos {
			expDate := uint64(time.Now().Add(24*time.Hour).UnixNano() / 1000000)
			uploadedFile := nestedGateway.UploadedFile{
				Type:                fileInfo.Type,
				Size:                fileInfo.Size,
				Name:                fileInfo.Filename,
				UniversalId:         fileInfo.ID,
				ExpirationTimestamp: expDate,
			}
			if len(string(fileInfo.Thumbnails.Original)) > 0 {
				uploadedFile.Thumbs = fileInfo.Thumbnails
			}
			r = append(r, uploadedFile)
		}
		resp.OkWithData(nested.M{"files": r})
	case nested.UPLOAD_TYPE_PLACE_PICTURE, nested.UPLOAD_TYPE_PROFILE_PICTURE:
		if p, err := multipartReader.NextPart(); err != nil {
			resp.Error(nested.ERR_UNKNOWN, []string{})
			ctx.StatusCode(http.StatusExpectationFailed)
			ctx.JSON(resp)
			return

		} else if fileInfo, err := uploadFile(p, uploadType, uploaderID, false); err != nil {
			resp.Error(nested.ERR_INVALID, []string{})
			ctx.StatusCode(http.StatusExpectationFailed)
			ctx.JSON(resp)
			return

		} else {
			expDate := uint64(time.Now().Add(24*time.Hour).UnixNano() / 1000000)
			uploadedFile := nestedGateway.UploadedFile{
				Type:                fileInfo.Type,
				Size:                fileInfo.Size,
				Name:                fileInfo.Filename,
				UniversalId:         fileInfo.ID,
				ExpirationTimestamp: expDate,
			}
			if len(string(fileInfo.Thumbnails.Original)) > 0 {
				uploadedFile.Thumbs = fileInfo.Thumbnails
			}
			resp.OkWithData(nested.M{"files": []nestedGateway.UploadedFile{uploadedFile}})
		}
	default:
		ctx.StatusCode(http.StatusBadRequest)
		resp.Error(nested.ERR_INVALID, []string{"request"})
	}
	ctx.JSON(resp)
}
func (fs *Server) UploadUser(ctx iris.Context) {
	var session *nested.Session
	var multipartReader *multipart.Reader
	uploadType := strings.ToUpper(ctx.Params().Get("uploadType"))
	sessionID := ctx.Params().Get("sessionID")
	resp := new(nestedGateway.Response)
	if !bson.IsObjectIdHex(sessionID) {
		ctx.StatusCode(http.StatusUnauthorized)
		resp.Error(nested.ERR_ACCESS, []string{})
		ctx.JSON(resp)
		return
	} else {
		session = _NestedModel.Session.GetByID(bson.ObjectIdHex(sessionID))
		if session == nil {
			ctx.StatusCode(http.StatusUnauthorized)
			resp.Error(nested.ERR_ACCESS, []string{})
			ctx.JSON(resp)
			return
		}
	}
	uploaderID := session.AccountID

	if r, err := ctx.Request().MultipartReader(); err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		resp.Error(nested.ERR_INVALID, []string{"request"})
		ctx.JSON(resp)
		return
	} else {
		multipartReader = r
	}
	switch uploadType {
	case nested.UPLOAD_TYPE_FILE, nested.UPLOAD_TYPE_GIF, nested.UPLOAD_TYPE_VOICE,
		nested.UPLOAD_TYPE_AUDIO, nested.UPLOAD_TYPE_IMAGE, nested.UPLOAD_TYPE_VIDEO:
		var fileInfos []nested.FileInfo
		for part, err := multipartReader.NextPart(); nil == err; part, err = multipartReader.NextPart() {
			if fileInfo, err := uploadFile(part, uploadType, uploaderID, false); err != nil {
				resp.Error(nested.ERR_UNKNOWN, []string{})
				ctx.StatusCode(http.StatusExpectationFailed)
				ctx.JSON(resp)
				return

			} else {
				fileInfos = append(fileInfos, *fileInfo)
			}
		}
		r := make([]nestedGateway.UploadedFile, 0, len(fileInfos))
		for _, fileInfo := range fileInfos {
			expDate := uint64(time.Now().Add(24*time.Hour).UnixNano() / 1000000)
			uploadedFile := nestedGateway.UploadedFile{
				Type:                fileInfo.Type,
				Size:                fileInfo.Size,
				Name:                fileInfo.Filename,
				UniversalId:         fileInfo.ID,
				ExpirationTimestamp: expDate,
			}
			if len(string(fileInfo.Thumbnails.Original)) > 0 {
				uploadedFile.Thumbs = fileInfo.Thumbnails
			}
			r = append(r, uploadedFile)
		}
		resp.OkWithData(nested.M{"files": r})
	case nested.UPLOAD_TYPE_PLACE_PICTURE, nested.UPLOAD_TYPE_PROFILE_PICTURE:
		if p, err := multipartReader.NextPart(); err != nil {
			resp.Error(nested.ERR_UNKNOWN, []string{})
			ctx.StatusCode(http.StatusExpectationFailed)
			ctx.JSON(resp)
			return

		} else if fileInfo, err := uploadFile(p, uploadType, uploaderID, false); err != nil {
			resp.Error(nested.ERR_INVALID, []string{})
			ctx.StatusCode(http.StatusExpectationFailed)
			ctx.JSON(resp)
			return

		} else {
			expDate := uint64(time.Now().Add(24*time.Hour).UnixNano() / 1000000)
			uploadedFile := nestedGateway.UploadedFile{
				Type:                fileInfo.Type,
				Size:                fileInfo.Size,
				Name:                fileInfo.Filename,
				UniversalId:         fileInfo.ID,
				ExpirationTimestamp: expDate,
			}
			if len(string(fileInfo.Thumbnails.Original)) > 0 {
				uploadedFile.Thumbs = fileInfo.Thumbnails
			}
			resp.OkWithData(nested.M{"files": []nestedGateway.UploadedFile{uploadedFile}})
		}
	default:
		ctx.StatusCode(http.StatusBadRequest)
		resp.Error(nested.ERR_INVALID, []string{"request"})

	}
	ctx.JSON(resp)
}
func (fs *Server) UploadApp(ctx iris.Context) {
	var multipartReader *multipart.Reader
	uploadType := strings.ToUpper(ctx.Params().Get("uploadType"))
	appToken := ctx.Params().Get("appToken")
	resp := new(nestedGateway.Response)
	token := _NestedModel.Token.GetAppToken(appToken)
	if token == nil {
		ctx.StatusCode(http.StatusUnauthorized)
		resp.Error(nested.ERR_ACCESS, []string{})
		ctx.JSON(resp)
		return
	}

	uploaderID := token.AccountID

	if r, err := ctx.Request().MultipartReader(); err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		resp.Error(nested.ERR_INVALID, []string{"request"})
		ctx.JSON(resp)
		return
	} else {
		multipartReader = r
	}
	switch uploadType {
	case nested.UPLOAD_TYPE_FILE, nested.UPLOAD_TYPE_GIF, nested.UPLOAD_TYPE_VOICE,
		nested.UPLOAD_TYPE_AUDIO, nested.UPLOAD_TYPE_IMAGE, nested.UPLOAD_TYPE_VIDEO:
		var fileInfos []nested.FileInfo
		for part, err := multipartReader.NextPart(); nil == err; part, err = multipartReader.NextPart() {
			if fileInfo, err := uploadFile(part, uploadType, uploaderID, false); err != nil {
				resp.Error(nested.ERR_UNKNOWN, []string{})
				ctx.StatusCode(http.StatusExpectationFailed)
				ctx.JSON(resp)
				return

			} else {
				fileInfos = append(fileInfos, *fileInfo)
			}
		}
		r := make([]nestedGateway.UploadedFile, 0, len(fileInfos))
		for _, fileInfo := range fileInfos {
			expDate := uint64(time.Now().Add(24*time.Hour).UnixNano() / 1000000)
			uploadedFile := nestedGateway.UploadedFile{
				Type:                fileInfo.Type,
				Size:                fileInfo.Size,
				Name:                fileInfo.Filename,
				UniversalId:         fileInfo.ID,
				ExpirationTimestamp: expDate,
			}
			if len(string(fileInfo.Thumbnails.Original)) > 0 {
				uploadedFile.Thumbs = fileInfo.Thumbnails
			}
			r = append(r, uploadedFile)
		}
		resp.OkWithData(nested.M{"files": r})
	case nested.UPLOAD_TYPE_PLACE_PICTURE, nested.UPLOAD_TYPE_PROFILE_PICTURE:
		if p, err := multipartReader.NextPart(); err != nil {
			resp.Error(nested.ERR_UNKNOWN, []string{})
			ctx.StatusCode(http.StatusExpectationFailed)
			ctx.JSON(resp)
			return

		} else if fileInfo, err := uploadFile(p, uploadType, uploaderID, false); err != nil {
			resp.Error(nested.ERR_INVALID, []string{})
			ctx.StatusCode(http.StatusExpectationFailed)
			ctx.JSON(resp)
			return

		} else {
			expDate := uint64(time.Now().Add(24*time.Hour).UnixNano() / 1000000)
			uploadedFile := nestedGateway.UploadedFile{
				Type:                fileInfo.Type,
				Size:                fileInfo.Size,
				Name:                fileInfo.Filename,
				UniversalId:         fileInfo.ID,
				ExpirationTimestamp: expDate,
			}
			if len(string(fileInfo.Thumbnails.Original)) > 0 {
				uploadedFile.Thumbs = fileInfo.Thumbnails
			}
			resp.OkWithData(nested.M{"files": []nestedGateway.UploadedFile{uploadedFile}})
		}
	default:
		ctx.StatusCode(http.StatusBadRequest)
		resp.Error(nested.ERR_INVALID, []string{"request"})

	}
	ctx.JSON(resp)
}

func uploadFile(p *multipart.Part, uploadType, uploader string, earlyResponse bool) (*nested.FileInfo, error) {
	defer p.Close()

	filename := p.FileName()
	if len(filename) == 0 {
		filename = "BLOB-File"
	}
	extension := path.Ext(filename)
	basename := filename[:len(filename)-len(extension)]

	storedFileInfo := nested.GenerateFileInfo(filename, uploader, "", nil, nil)

	// Setup Nested file info
	fileInfo := nested.FileInfo{
		ID:              storedFileInfo.ID,
		Status:          nested.FILE_STATUS_TEMP,
		Filename:        filename,
		UploaderId:      uploader,
		UploadType:      nested.UPLOAD_TYPE_FILE,
		UploadTimestamp: nested.Timestamp(),
		Type:            nested.GetTypeByFilename(filename),
		MimeType:        nested.GetMimeTypeByFilename(filename),
	}

	// Save Pre-Processor
	var savePreprocessor pipe

	// File Process
	var processList []Processor
	metaData := new(nested.MetaData)
	metaData.Thumbnails = make(nested.Thumbnails)
	metaDataLock := new(sync.Mutex)
	wgMetaData := new(sync.WaitGroup)

	switch uploadType {
	case nested.UPLOAD_TYPE_FILE:
		switch storedFileInfo.Metadata.Type {
		case nested.FILE_TYPE_IMAGE, nested.FILE_TYPE_GIF, nested.FILE_TYPE_VIDEO, nested.FILE_TYPE_AUDIO:
			// Preview
			// Thumbs
			processList = []Processor{
				&previewGenerator{
					MaxWidth:  DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_PREVIEW],
					Uploader:  storedFileInfo.Metadata.Uploader,
					Filename:  storedFileInfo.Name,
					MimeType:  storedFileInfo.MimeType,
					ThumbName: THUMBNAIL_PREVIEW,
					MetaData:  metaData,
					Lock:      metaDataLock,
					WaitGroup: wgMetaData,
				},
				&thumbGenerator{
					MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_32],
					Uploader:     storedFileInfo.Metadata.Uploader,
					Filename:     storedFileInfo.Name,
					MimeType:     storedFileInfo.MimeType,
					ThumbName:    THUMBNAIL_32,
					MetaData:     metaData,
					Lock:         metaDataLock,
					WaitGroup:    wgMetaData,
				},
				&thumbGenerator{
					MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_64],
					Uploader:     storedFileInfo.Metadata.Uploader,
					Filename:     storedFileInfo.Name,
					MimeType:     storedFileInfo.MimeType,
					ThumbName:    THUMBNAIL_64,
					MetaData:     metaData,
					Lock:         metaDataLock,
					WaitGroup:    wgMetaData,
				},
				&thumbGenerator{
					MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_128],
					Uploader:     storedFileInfo.Metadata.Uploader,
					Filename:     storedFileInfo.Name,
					MimeType:     storedFileInfo.MimeType,
					ThumbName:    THUMBNAIL_128,
					MetaData:     metaData,
					Lock:         metaDataLock,
					WaitGroup:    wgMetaData,
				},
			}
		}

		// Meta Reader
		switch storedFileInfo.Metadata.Type {
		case nested.FILE_TYPE_IMAGE:
			processList = append(processList, &imageMetaReader{
				MetaData:  metaData,
				Lock:      metaDataLock,
				WaitGroup: wgMetaData,
			})

		case nested.FILE_TYPE_AUDIO:
			processList = append(processList, &audioMetaReader{
				MetaData:  metaData,
				Lock:      metaDataLock,
				WaitGroup: wgMetaData,
			})

		case nested.FILE_TYPE_VIDEO:
			processList = append(processList, &videoMetaReader{
				MetaData:  metaData,
				Lock:      metaDataLock,
				WaitGroup: wgMetaData,
			})

		case nested.FILE_TYPE_DOCUMENT:
			processList = append(processList, &documentMetaReader{
				MimeType:  storedFileInfo.MimeType,
				MetaData:  metaData,
				Lock:      metaDataLock,
				WaitGroup: wgMetaData,
			})
		}

		wgMetaData.Add(len(processList))

	case nested.UPLOAD_TYPE_PLACE_PICTURE, nested.UPLOAD_TYPE_PROFILE_PICTURE:
		if nested.FILE_TYPE_IMAGE != storedFileInfo.Metadata.Type {
			_Log.Warn("Invalid file uploaded as place/profile picture")
			return nil, protocol.NewInvalidError([]string{"mime_type"}, nil)

		}

		// Thumbs
		fileInfo.Status = nested.FILE_STATUS_PUBLIC
		processList = []Processor{
			&previewGenerator{MaxWidth: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_PREVIEW], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_PREVIEW, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_32], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_32, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_64], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_64, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_128], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_128, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
		}

		wgMetaData.Add(len(processList))

	case nested.UPLOAD_TYPE_VIDEO:
		if nested.FILE_TYPE_VIDEO != storedFileInfo.Metadata.Type {
			_Log.Warn("Invalid file uploaded as Video")
			return nil, protocol.NewInvalidError([]string{"mime_type"}, nil)

		}

		savePreprocessor = func(w io.Writer, r io.Reader) (int64, error) {
			if or, err := _FileConverter.Video.ToMp4(r, 23, 0, 720, 128); err != nil {
				return 0, err
			} else {
				return io.Copy(w, or)
			}
		}

		fileInfo.UploadType = nested.UPLOAD_TYPE_VIDEO
		fileInfo.Filename = fmt.Sprintf("%s.mp4", basename)

		// Thumbs
		// Meta Reader
		processList = []Processor{
			&videoMetaReader{MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&previewGenerator{MaxWidth: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_PREVIEW], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_PREVIEW, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_32], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_32, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_64], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_64, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_128], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_128, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
		}

		wgMetaData.Add(len(processList))

	case nested.UPLOAD_TYPE_AUDIO:
		if nested.FILE_TYPE_AUDIO != storedFileInfo.Metadata.Type {
			_Log.Warn("Invalid file uploaded as Audio")
			return nil, protocol.NewInvalidError([]string{"mime_type"}, nil)
		}

		savePreprocessor = func(w io.Writer, r io.Reader) (int64, error) {
			if or, err := _FileConverter.Audio.ToMp3(r, 3); err != nil {
				return 0, err
			} else {
				return io.Copy(w, or)
			}
		}

		fileInfo.UploadType = nested.UPLOAD_TYPE_AUDIO
		fileInfo.Filename = fmt.Sprintf("%s.mp3", basename)

		// FIXME: Waveform
		// Meta Reader
		processList = []Processor{
			&audioMetaReader{MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&previewGenerator{MaxWidth: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_PREVIEW], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_PREVIEW, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_32], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_32, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_64], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_64, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_128], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_128, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
		}

		wgMetaData.Add(len(processList))

	case nested.UPLOAD_TYPE_VOICE:
		if nested.FILE_TYPE_AUDIO != storedFileInfo.Metadata.Type {
			_Log.Warn("Invalid file uploaded as Voice")
			return nil, protocol.NewInvalidError([]string{"mime_type"}, nil)

		}

		savePreprocessor = func(w io.Writer, r io.Reader) (int64, error) {
			if or, err := _FileConverter.Voice.ToMp3(r, 9); err != nil {
				return 0, err
			} else {
				return io.Copy(w, or)
			}
		}

		// TODO:: GenerateFileInfo does not support FILE_TYPE_VOICE ?!?!?!
		storedFileInfo = nested.GenerateFileInfo(filename, uploader, nested.FILE_TYPE_VOICE, nil, nil)
		storedFileInfo.Metadata.Type = nested.FILE_TYPE_VOICE

		fileInfo.ID = nested.UniversalID(storedFileInfo.ID)
		fileInfo.UploadType = nested.UPLOAD_TYPE_VOICE
		fileInfo.Filename = fmt.Sprintf("%s.mp3", basename)

		// TODO: Waveform
		// Meta Reader
		processList = []Processor{
			&voiceMetaReader{MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
		}

		wgMetaData.Add(len(processList))

	case nested.UPLOAD_TYPE_IMAGE:
		if nested.FILE_TYPE_IMAGE != storedFileInfo.Metadata.Type {
			_Log.Warn("Invalid file uploaded as Image")
			return nil, protocol.NewInvalidError([]string{"mime_type"}, nil)

		}

		savePreprocessor = func(w io.Writer, r io.Reader) (int64, error) {
			if or, err := _FileConverter.Image.ToJpeg(r, 1200, 0); err != nil {
				return 0, err
			} else {
				return io.Copy(w, or)
			}
		}

		fileInfo.UploadType = nested.UPLOAD_TYPE_IMAGE
		fileInfo.Filename = fmt.Sprintf("%s.jpg", basename)

		// Thumbs
		// Preview
		// Meta Reader
		processList = []Processor{
			&imageMetaReader{MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&previewGenerator{MaxWidth: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_PREVIEW], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_PREVIEW, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_32], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_32, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_64], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_64, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_128], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_128, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
		}

		wgMetaData.Add(len(processList))

	case nested.UPLOAD_TYPE_GIF:
		if nested.FILE_TYPE_GIF != storedFileInfo.Metadata.Type {
			_Log.Warn("Invalid file uploaded as Gif")
			return nil, protocol.NewInvalidError([]string{"mime_type"}, nil)

		}

		savePreprocessor = func(w io.Writer, r io.Reader) (int64, error) {
			if or, err := _FileConverter.Gif.ToMp4(r, 23, 0, 0); err != nil {
				return 0, err
			} else {
				return io.Copy(w, or)
			}
		}

		fileInfo.UploadType = nested.UPLOAD_TYPE_GIF
		fileInfo.Filename = fmt.Sprintf("%s.mp4", basename)

		// Thumbs
		// Preview
		// Meta Reader
		processList = []Processor{
			&gifMetaReader{MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&previewGenerator{MaxWidth: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_PREVIEW], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_PREVIEW, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_32], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_32, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_64], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_64, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
			&thumbGenerator{MaxDimension: DEFAULT_THUMBNAIL_SIZES[THUMBNAIL_128], Uploader: storedFileInfo.Metadata.Uploader, Filename: storedFileInfo.Name, MimeType: storedFileInfo.MimeType, ThumbName: THUMBNAIL_128, MetaData: metaData, Lock: metaDataLock, WaitGroup: wgMetaData},
		}

		wgMetaData.Add(len(processList))
	}

	// Piped Reader/Writers
	rs := make([]*io.PipeReader, len(processList)+1)
	ws := make([]io.Writer, len(processList)+1)

	// Wait groups
	wgMain := sync.WaitGroup{}
	wgProcess := sync.WaitGroup{}

	// Failure indicator
	chErr := make(chan error, 2)

	for k := range rs {
		if 0 == k && savePreprocessor != nil {
			var ir *io.PipeReader
			var iw *io.PipeWriter

			ir, ws[k] = io.Pipe()
			rs[k], iw = io.Pipe()

			go func(w *io.PipeWriter, r *io.PipeReader) {
				if n, err := savePreprocessor(w, r); err != nil {
					_Log.Warn(err.Error())
					r.CloseWithError(err) // Occur error on multi-writer write
					w.CloseWithError(err) // Occur error on save read
				} else if 0 == n {
					_Log.Warn("Save Pre-Processor returned empty")
					err := protocol.NewInvalidError([]string{"input"}, nil)
					chErr <- err
					r.CloseWithError(err) // Occur error on multi-writer write
					w.CloseWithError(err) // Occur error on save read
				} else {
					w.Close()
				}
			}(iw, ir)
		} else {
			rs[k], ws[k] = io.Pipe()
		}
	}

	// Save File in Xerxes & Nested Database
	wgMain.Add(1)
	go func(r io.Reader) {
		defer wgMain.Done()

		if info := _NestedModel.Store.Save(r, storedFileInfo); info == nil {
			err := errors.New("file insertion in storage database failed")
			_Log.Warn(err.Error())
			r.(*io.PipeReader).CloseWithError(err)
		} else {
			fileInfo.Size = int64(info.Size)
			if !_NestedModel.File.AddFile(fileInfo) {
				err := errors.New("file submit fail")
				_Log.Warn(err.Error())
				r.(*io.PipeReader).CloseWithError(err)
			}
		}
	}(rs[0])

	// Process File
	for k, v := range processList {
		wgProcess.Add(1)
		go func(r io.Reader, process Processor) {
			defer wgProcess.Done()

			if err := process.Process(r); err != nil {
				_Log.Warn(err.Error())
				// TODO: Retry
			}

			// Let's read the remaining
			io.Copy(ioutil.Discard, r)
		}(rs[k+1], v)
	}

	// Meta Data Collector
	wgProcess.Add(1)
	go func() {
		defer wgProcess.Done()

		wgMetaData.Wait()
		wgMain.Wait()

		storedFileInfo.Metadata = *metaData
		if storedFileInfo.Metadata.Meta != nil {
			// Update Files Model
			if err := _NestedModel.Store.SetMeta(storedFileInfo.ID, storedFileInfo.Metadata); err != nil {
				_Log.Warn(err.Error())
				// TODO: Retry
			}

			if nil != storedFileInfo.Metadata.Meta {
				_NestedModel.File.SetMetadata(fileInfo.ID, storedFileInfo.Metadata.Meta) // TODO: Check the result
			}

			switch m := storedFileInfo.Metadata.Meta.(type) {
			case nested.MetaImage:
				fileInfo.Width = m.OriginalWidth
				fileInfo.Height = m.OriginalHeight
			case nested.MetaVideo:
				fileInfo.Width = m.Width
				fileInfo.Height = m.Height
			case nested.MetaPdf:
				fileInfo.Width = int64(m.Width)
				fileInfo.Height = int64(m.Height)
			case nested.MetaGif:
				fileInfo.Width = int64(m.Width)
				fileInfo.Height = int64(m.Height)
			}

			if _NestedModel.File.SetDimension(fileInfo.ID, fileInfo.Width, fileInfo.Height) != true {
				_Log.Warn("file dimension update failed")
				// TODO: Retry
			}
		}

		if len(metaData.Thumbnails) > 0 {
			// Update Files Model
			if err := _NestedModel.Store.SetThumbnails(storedFileInfo.ID, metaData.Thumbnails); err != nil {
				_Log.Warn(err.Error())
				// TODO: Retry
			}

			fileInfo.Thumbnails = nested.Picture{
				Original: nested.UniversalID(storedFileInfo.ID),
			}

			for k, v := range metaData.Thumbnails {
				switch k {
				case THUMBNAIL_32:
					fileInfo.Thumbnails.X32 = nested.UniversalID(v.ID)

				case THUMBNAIL_64:
					fileInfo.Thumbnails.X64 = nested.UniversalID(v.ID)

				case THUMBNAIL_128:
					fileInfo.Thumbnails.X128 = nested.UniversalID(v.ID)

				case THUMBNAIL_PREVIEW:
					fileInfo.Thumbnails.Preview = nested.UniversalID(v.ID)
				}
			}

			// Update FileInfo Thumbnails in Nested DB
			if _NestedModel.File.SetThumbnails(fileInfo.ID, fileInfo.Thumbnails) != true {
				_Log.Warn("File thumbnail update failed for")
				// TODO: Retry
				return

			}
		}
	}()

	mw := io.MultiWriter(ws...)

	// FIXME: Obey BW shaping
	rateLimit := int64(10 * 1024 * 1024)
	chTick := time.NewTicker(time.Second)
	defer chTick.Stop()
upload:
	for {
		select {
		case <-chTick.C:
			if n, err := io.CopyN(mw, p, rateLimit); 0 == n || err != nil {
				switch err {
				case io.EOF:
				default:
					_Log.Warn(err.Error())
					select {
					case chErr <- err:
					default:
					}
				}
				break upload
			}
		}
	}

	for _, w := range ws {
		w.(*io.PipeWriter).Close()
	}

	wgMain.Wait()

	// Block client connection if request is not early-responded until all thumbs are created
	if !earlyResponse {
		wgProcess.Wait()
	}

	select {
	case err := <-chErr:
		return nil, err

	default:
	}

	return &fileInfo, nil
}
