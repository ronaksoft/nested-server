package nestedServiceFile

import (
    "git.ronaksoft.com/nested/server/nested"
    "git.ronaksoft.com/nested/server/pkg/global"
    "git.ronaksoft.com/nested/server/pkg/rpc"
    tools "git.ronaksoft.com/nested/server/pkg/toolbox"
    "github.com/globalsign/mgo/bson"
)

// @Command: file/get_download_token
// @CommandInfo:	Creates a download token based on input items
// @Input:	universal_id	string		*
// @Input:	post_id			string		+
// @Input: task_id			string		+
func (s *FileService) getDownloadToken(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
    var uniID nested.UniversalID
    var fileInfo *nested.FileInfo
    var post *nested.Post
    var task *nested.Task
    if v, ok := request.Data["universal_id"].(string); ok {
        uniID = nested.UniversalID(v)
        fileInfo = s.Worker().Model().File.GetByID(uniID, nil)
        if fileInfo == nil {
            response.Error(global.ErrInvalid, []string{"universal_id"})
            return
        }
        // if file is public you do not need token
        if fileInfo.IsPublic() {
            if token, err := nested.GenerateDownloadToken(uniID, request.SessionKey, requester.ID); err != nil {
                response.Error(global.ErrUnknown, []string{})
            } else {
                response.OkWithData(tools.M{"token": token})
            }
            return
        }
    } else {
        response.Error(global.ErrIncomplete, []string{"universal_id"})
        return
    }

    // check if post_id has been set
    if postID, ok := request.Data["post_id"].(string); ok {
        if bson.IsObjectIdHex(postID) {
            post = s.Worker().Model().Post.GetPostByID(bson.ObjectIdHex(postID))
            if post == nil {
                response.Error(global.ErrUnavailable, []string{"post_id"})
                return
            }
        }
    } else {
        post = nil
    }

    if post != nil {
        if !post.HasAccess(requester.ID) {
            response.Error(global.ErrAccess, []string{"post_id"})
            return
        }

        if s.Worker().Model().File.IsPostOwner(uniID, post.ID) {
            if token, err := nested.GenerateDownloadToken(uniID, request.SessionKey, requester.ID); err != nil {
                response.Error(global.ErrUnknown, []string{})
            } else {
                response.OkWithData(tools.M{"token": token})
            }
            return
        }
        response.Error(global.ErrAccess, []string{"post_is_not_owner"})
        return
    }

    task = s.Worker().Argument().GetTask(request, response)
    if task != nil {
        if !task.HasAccess(requester.ID, nested.TaskAccessRead) {
            response.Error(global.ErrAccess, []string{"task_id"})
            return
        }
        if s.Worker().Model().File.IsTaskOwner(uniID, task.ID) {
            if token, err := nested.GenerateDownloadToken(uniID, request.SessionKey, requester.ID); err != nil {
                response.Error(global.ErrUnknown, []string{})
            } else {
                response.OkWithData(tools.M{"token": token})
            }
        } else {
            response.Error(global.ErrInvalid, []string{"task_id"})
        }
        return
    }

    response.Error(global.ErrIncomplete, []string{"post_id or task_id required"})
}

// @Command: file/get_upload_token
// @CommandInfo:	Creates an upload token for the user of current session
func (s *FileService) getUploadToken(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
    if token, err := nested.GenerateUploadToken(request.SessionKey); err != nil {
        response.Error(global.ErrUnknown, []string{})
    } else {
        response.OkWithData(tools.M{"token": token})
    }
    return
}

// @Command: file/get
// @Input:	universal_id		string	*
func (s *FileService) getFileByID(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
    var f *nested.FileInfo
    if v, ok := request.Data["universal_id"].(string); ok {
        uniID := nested.UniversalID(v)
        f = s.Worker().Model().File.GetByID(uniID, nil)
        if f == nil {
            response.Error(global.ErrInvalid, []string{"universal_id"})
            return
        }
    } else {
        response.Error(global.ErrIncomplete, []string{"universal_id"})
        return
    }
    response.OkWithData(s.Worker().Map().FileInfo(*f))
}

// @Command:	file/get_recent_files
// @Pagination
func (s *FileService) getRecentFiles(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
    files := s.Worker().Model().File.GetFilesByPlaces(requester.BookmarkedPlaceIDs, s.Worker().Argument().GetPagination(request))
    r := make([]tools.M, 0, len(files))
    for _, f := range files {
        r = append(r, s.Worker().Map().FileInfo(f))
    }
    response.OkWithData(tools.M{"files": r})
}

// @Command: file/get_by_token
// @Input:	token		string	*
func (s *FileService) getFileByToken(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
    var f *nested.FileInfo
    var uniID nested.UniversalID
    if v, ok := request.Data["token"].(string); ok {
        if uID, err := s.Worker().Model().Token.GetFileByToken(v); err != nil {
            response.Error(global.ErrInvalid, []string{"token"})
            return
        } else {
            uniID = uID
        }
    } else {
        response.Error(global.ErrIncomplete, []string{"token"})
        return
    }
    if f = s.Worker().Model().File.GetByID(uniID, nil); f == nil {
        response.Error(global.ErrUnknown, []string{})
    }
    response.OkWithData(s.Worker().Map().FileInfo(*f))
}

func (s *FileService) uploadFile(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
    // TODO:: implement it
}

func (s *FileService) downloadFile(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
    // TODO:: implement it
}
