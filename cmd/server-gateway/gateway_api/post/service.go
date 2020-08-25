package nestedServicePost

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/model"
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
		CMD_ADD:                   {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.createPost},
		CMD_ADD_COMMENT:           {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.addComment},
		CMD_ADD_LABEL:             {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.addLabelToPost},
		CMD_GET:                   {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.getPost},
		CMD_GET_MANY:              {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.getManyPosts},
		CMD_GET_COMMENTS_BY_POST:  {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.getCommentsByPost},
		CMD_GET_COMMENT:           {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.getCommentByID},
		CMD_GET_MANY_COMMENTS:     {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.getManyCommentsByIDs},
		CMD_GET_ACTIVITIES:        {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.getPostActivities},
		CMD_ATTACH_PLACE:          {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.attachPlace},
		CMD_GET_COUNTERS:          {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.getPostCounters},
		CMD_GET_CHAIN:             {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.getPostChain},
		CMD_RETRACT:               {MinAuthLevel: api.AUTH_LEVEL_APP_L3, Execute: s.retractPost},
		CMD_WIPE:                  {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.retractPost},
		CMD_REMOVE:                {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.removePost},
		CMD_REMOVE_COMMENT:        {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.removeComment},
		CMD_REMOVE_LABEL:          {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.removeLabelFromPost},
		CMD_REPLACE:               {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.movePost}, // Deprecated
		CMD_MOVE:                  {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.movePost},
		CMD_MARK_AS_READ:          {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.markPostAsRead},
		CMD_SET_NOTIFICATION:      {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.setPostNotification},
		CMD_WHO_READ:              {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.whoHaveReadThisPost},
		CMD_ADD_TO_BOOKMARKS:      {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.addToBookmarks},
		CMD_REMOVE_FROM_BOOKMARKS: {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.removeFromBookmarks},
		CMD_EDIT:                  {MinAuthLevel: api.AUTH_LEVEL_USER, Execute: s.editPost},
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
