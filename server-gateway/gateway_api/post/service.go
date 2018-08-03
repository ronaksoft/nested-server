package nestedServicePost

import (
    "git.ronaksoftware.com/nested/server/server-gateway/client"
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server/server-gateway/gateway_api"
)

const (
    SERVICE_PREFIX = "post"
)
const (
    CMD_ADD                   = "post/add"
    CMD_ADD_COMMENT           = "post/add_comment"
    CMD_ADD_LABEL             = "post/add_label"
    CMD_ATTACH_PLACE          = "post/attach_place"
    CMD_GET                   = "post/get"
    CMD_GET_COUNTERS          = "post/get_counters"
    CMD_GET_MANY              = "post/get_many"
    CMD_GET_CHAIN             = "post/get_chain"
    CMD_GET_COMMENTS_BY_POST  = "post/get_comments"
    CMD_GET_COMMENT           = "post/get_comment"
    CMD_GET_MANY_COMMENTS     = "post/get_many_comments"
    CMD_GET_ACTIVITIES        = "post/get_activities"
    CMD_WIPE                  = "post/wipe"
    CMD_RETRACT               = "post/retract"
    CMD_REMOVE                = "post/remove"
    CMD_REMOVE_COMMENT        = "post/remove_comment"
    CMD_REMOVE_LABEL          = "post/remove_label"
    CMD_REPLACE               = "post/replace"
    CMD_MARK_AS_READ          = "post/mark_as_read"
    CMD_MOVE                  = "post/move"
    CMD_SET_NOTIFICATION      = "post/set_notification"
    CMD_WHO_READ              = "post/who_read"
    CMD_ADD_TO_BOOKMARKS      = "post/add_to_bookmarks"
    CMD_REMOVE_FROM_BOOKMARKS = "post/remove_from_bookmarks"
    CMD_EDIT                  = "post/edit"
)

var (
    _Model *nested.Manager
)

type PostService struct {
    worker          *api.Worker
    serviceCommands api.ServiceCommands
}

func NewPostService(worker *api.Worker) *PostService {
    s := new(PostService)
    s.worker = worker

    s.serviceCommands = api.ServiceCommands{
        CMD_ADD:                   {api.AUTH_LEVEL_APP_L3, s.createPost},
        CMD_ADD_COMMENT:           {api.AUTH_LEVEL_APP_L3, s.addComment},
        CMD_ADD_LABEL:             {api.AUTH_LEVEL_APP_L3, s.addLabelToPost},
        CMD_GET:                   {api.AUTH_LEVEL_APP_L3, s.getPost},
        CMD_GET_MANY:              {api.AUTH_LEVEL_APP_L3, s.getManyPosts},
        CMD_GET_COMMENTS_BY_POST:  {api.AUTH_LEVEL_APP_L3, s.getCommentsByPost},
        CMD_GET_COMMENT:           {api.AUTH_LEVEL_APP_L3, s.getCommentByID},
        CMD_GET_MANY_COMMENTS:     {api.AUTH_LEVEL_APP_L3, s.getManyCommentsByIDs},
        CMD_GET_ACTIVITIES:        {api.AUTH_LEVEL_APP_L3, s.getPostActivities},
        CMD_ATTACH_PLACE:          {api.AUTH_LEVEL_USER, s.attachPlace},
        CMD_GET_COUNTERS:          {api.AUTH_LEVEL_USER, s.getPostCounters},
        CMD_GET_CHAIN:             {api.AUTH_LEVEL_APP_L3, s.getPostChain},
        CMD_RETRACT:               {api.AUTH_LEVEL_APP_L3, s.retractPost},
        CMD_WIPE:                  {api.AUTH_LEVEL_USER, s.retractPost},
        CMD_REMOVE:                {api.AUTH_LEVEL_USER, s.removePost},
        CMD_REMOVE_COMMENT:        {api.AUTH_LEVEL_USER, s.removeComment},
        CMD_REMOVE_LABEL:          {api.AUTH_LEVEL_USER, s.removeLabelFromPost},
        CMD_REPLACE:               {api.AUTH_LEVEL_USER, s.movePost}, // Deprecated
        CMD_MOVE:                  {api.AUTH_LEVEL_USER, s.movePost},
        CMD_MARK_AS_READ:          {api.AUTH_LEVEL_USER, s.markPostAsRead},
        CMD_SET_NOTIFICATION:      {api.AUTH_LEVEL_USER, s.setPostNotification},
        CMD_WHO_READ:              {api.AUTH_LEVEL_USER, s.whoHaveReadThisPost},
        CMD_ADD_TO_BOOKMARKS:      {api.AUTH_LEVEL_USER, s.addToBookmarks},
        CMD_REMOVE_FROM_BOOKMARKS: {api.AUTH_LEVEL_USER, s.removeFromBookmarks},
        CMD_EDIT:                  {api.AUTH_LEVEL_USER, s.editPost},
    }

    _Model = s.worker.Model()
    return s
}

func (s *PostService) GetServicePrefix() string {
    return SERVICE_PREFIX
}

func (s *PostService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    commandName := request.Command
    if cmd, ok := s.serviceCommands[commandName]; ok {
        if authLevel >= cmd.MinAuthLevel {
            cmd.Execute(requester, request, response)
        } else {
            response.NotAuthorized()
        }
    } else {
        response.NotImplemented()
    }
}

func (s *PostService) Worker() *api.Worker {
    return s.worker
}
