package nestedServiceFile

import (
	"git.ronaksoftware.com/nested/server/model"
	"git.ronaksoftware.com/nested/server/server-gateway/client"
	"github.com/globalsign/mgo/bson"
)

// @Command: file/get_download_token
// @CommandInfo:	Creates a download token based on input items
// @Input:	universal_id	string		*
// @Input:	post_id			string		+
// @Input: task_id			string		+
func (s *FileService) getDownloadToken(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var uniID nested.UniversalID
	var fileInfo *nested.FileInfo
	var post *nested.Post
	var task *nested.Task
	if v, ok := request.Data["universal_id"].(string); ok {
		uniID = nested.UniversalID(v)
		fileInfo = _Model.File.GetByID(uniID, nil)
		if fileInfo == nil {
			response.Error(nested.ERR_INVALID, []string{"universal_id"})
			return
		}
		// if file is public you do not need token
		if fileInfo.IsPublic() {
			if token, err := nested.GenerateDownloadToken(uniID, request.SessionKey, requester.ID); err != nil {
				response.Error(nested.ERR_UNKNOWN, []string{})
			} else {
				response.OkWithData(nested.M{"token": token})
			}
			return
		}
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"universal_id"})
		return
	}

	// check if post_id has been set
	if postID, ok := request.Data["post_id"].(string); ok {
		if bson.IsObjectIdHex(postID) {
			post = s.Worker().Model().Post.GetPostByID(bson.ObjectIdHex(postID))
			if post == nil {
				response.Error(nested.ERR_UNAVAILABLE, []string{"post_id"})
				return
			}
		}
	} else {
		post = nil
	}

	if post != nil {
		if !post.HasAccess(requester.ID) {
			response.Error(nested.ERR_ACCESS, []string{"post_id"})
			return
		}

		if s.Worker().Model().File.IsPostOwner(uniID, post.ID) {
			if token, err := nested.GenerateDownloadToken(uniID, request.SessionKey, requester.ID); err != nil {
				response.Error(nested.ERR_UNKNOWN, []string{})
			} else {
				response.OkWithData(nested.M{"token": token})
			}
			return
		}
		response.Error(nested.ERR_ACCESS, []string{"post_is_not_owner"})
		return
	}

	task = s.Worker().Argument().GetTask(request, response)
	if task != nil {
		if !task.HasAccess(requester.ID, nested.TASK_ACCESS_READ) {
			response.Error(nested.ERR_ACCESS, []string{"task_id"})
			return
		}
		if s.Worker().Model().File.IsTaskOwner(uniID, task.ID) {
			if token, err := nested.GenerateDownloadToken(uniID, request.SessionKey, requester.ID); err != nil {
				response.Error(nested.ERR_UNKNOWN, []string{})
			} else {
				response.OkWithData(nested.M{"token": token})
			}
		} else {
			response.Error(nested.ERR_INVALID, []string{"task_id"})
		}
		return
	}

	response.Error(nested.ERR_INCOMPLETE, []string{"post_id  or task_id required"})
}

// @Command: file/get_upload_token
// @CommandInfo:	Creates an upload token for the user of current session
func (s *FileService) getUploadToken(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	if token, err := nested.GenerateUploadToken(request.SessionKey); err != nil {
		response.Error(nested.ERR_UNKNOWN, []string{})
	} else {
		response.OkWithData(nested.M{"token": token})
	}
	return
}

// @Command: file/get
// @Input:	universal_id		string	*
func (s *FileService) getFileByID(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var f *nested.FileInfo
	if v, ok := request.Data["universal_id"].(string); ok {
		uniID := nested.UniversalID(v)
		f = _Model.File.GetByID(uniID, nil)
		if f == nil {
			response.Error(nested.ERR_INVALID, []string{"universal_id"})
			return
		}
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"universal_id"})
		return
	}
	response.OkWithData(s.Worker().Map().FileInfo(*f))
}

// @Command:	file/get_recent_files
// @Pagination
func (s *FileService) getRecentFiles(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	files := _Model.File.GetFilesByPlaces(requester.BookmarkedPlaceIDs, s.Worker().Argument().GetPagination(request))
	r := make([]nested.M, 0, len(files))
	for _, f := range files {
		r = append(r, s.Worker().Map().FileInfo(f))
	}
	response.OkWithData(nested.M{"files": r})
}

// @Command: file/get_by_token
// @Input:	token		string	*
func (s *FileService) getFileByToken(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var f *nested.FileInfo
	var uniID nested.UniversalID
	if v, ok := request.Data["token"].(string); ok {
		if uID, err := _Model.Token.GetFileByToken(v); err != nil {
			response.Error(nested.ERR_INVALID, []string{"token"})
			return
		} else {
			uniID = uID
		}
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"token"})
		return
	}
	if f = _Model.File.GetByID(uniID, nil); f == nil {
		response.Error(nested.ERR_UNKNOWN, []string{})
	}
	response.OkWithData(s.Worker().Map().FileInfo(*f))
}

func (s *FileService) uploadFile(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	// TODO:: implement it
}

func (s *FileService) downloadFile(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	// TODO:: implement it
}
