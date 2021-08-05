package nestedServicePost

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix = "post"
)
const (
	CmdAdd                 = "post/add"
	CmdAddComment          = "post/add_comment"
	CmdAddLabel            = "post/add_label"
	CmdAttachPlace         = "post/attach_place"
	CmdGet                 = "post/get"
	CmdGetCounters         = "post/get_counters"
	CmdGetMany             = "post/get_many"
	CmdGetChain            = "post/get_chain"
	CmdGetCommentsByPost   = "post/get_comments"
	CmdGetComment          = "post/get_comment"
	CmdGetManyComments     = "post/get_many_comments"
	CmdGetActivities       = "post/get_activities"
	CmdWipe                = "post/wipe"
	CmdRetract             = "post/retract"
	CmdRemove              = "post/remove"
	CmdRemoveComment       = "post/remove_comment"
	CmdRemoveLabel         = "post/remove_label"
	CmdReplace             = "post/replace"
	CmdMarkAsRead          = "post/mark_as_read"
	CmdMove                = "post/move"
	CmdSetNotification     = "post/set_notification"
	CmdWhoRead             = "post/who_read"
	CmdAddToBookmarks      = "post/add_to_bookmarks"
	CmdRemoveFromBookmarks = "post/remove_from_bookmarks"
	CmdEdit                = "post/edit"
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
		CmdAdd:                 {MinAuthLevel: api.AuthLevelAppL3, Execute: s.createPost},
		CmdAddComment:          {MinAuthLevel: api.AuthLevelAppL3, Execute: s.addComment},
		CmdAddLabel:            {MinAuthLevel: api.AuthLevelAppL3, Execute: s.addLabelToPost},
		CmdGet:                 {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPost},
		CmdGetMany:             {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getManyPosts},
		CmdGetCommentsByPost:   {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getCommentsByPost},
		CmdGetComment:          {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getCommentByID},
		CmdGetManyComments:     {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getManyCommentsByIDs},
		CmdGetActivities:       {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPostActivities},
		CmdAttachPlace:         {MinAuthLevel: api.AuthLevelUser, Execute: s.attachPlace},
		CmdGetCounters:         {MinAuthLevel: api.AuthLevelUser, Execute: s.getPostCounters},
		CmdGetChain:            {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPostChain},
		CmdRetract:             {MinAuthLevel: api.AuthLevelAppL3, Execute: s.retractPost},
		CmdWipe:                {MinAuthLevel: api.AuthLevelUser, Execute: s.retractPost},
		CmdRemove:              {MinAuthLevel: api.AuthLevelUser, Execute: s.removePost},
		CmdRemoveComment:       {MinAuthLevel: api.AuthLevelUser, Execute: s.removeComment},
		CmdRemoveLabel:         {MinAuthLevel: api.AuthLevelUser, Execute: s.removeLabelFromPost},
		CmdReplace:             {MinAuthLevel: api.AuthLevelUser, Execute: s.movePost}, // Deprecated
		CmdMove:                {MinAuthLevel: api.AuthLevelUser, Execute: s.movePost},
		CmdMarkAsRead:          {MinAuthLevel: api.AuthLevelUser, Execute: s.markPostAsRead},
		CmdSetNotification:     {MinAuthLevel: api.AuthLevelUser, Execute: s.setPostNotification},
		CmdWhoRead:             {MinAuthLevel: api.AuthLevelUser, Execute: s.whoHaveReadThisPost},
		CmdAddToBookmarks:      {MinAuthLevel: api.AuthLevelUser, Execute: s.addToBookmarks},
		CmdRemoveFromBookmarks: {MinAuthLevel: api.AuthLevelUser, Execute: s.removeFromBookmarks},
		CmdEdit:                {MinAuthLevel: api.AuthLevelUser, Execute: s.editPost},
	}

	_Model = s.worker.Model()
	return s
}

func (s *PostService) GetServicePrefix() string {
	return ServicePrefix
}

func (s *PostService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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
