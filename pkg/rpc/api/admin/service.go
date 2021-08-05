package nestedServiceAdmin

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix string = "admin"
)
const (
	CmdCreatePost              string = "admin/create_post"
	CmdAddComment              string = "admin/add_comment"
	CmdHealthCheck             string = "admin/health_check"
	CmdPromote                 string = "admin/promote"
	CmdDemote                  string = "admin/demote"
	CmdPlaceCreateGrand        string = "admin/create_grand_place"
	CmdPlaceCreate             string = "admin/create_place"
	CmdPlaceList               string = "admin/place_list"
	CmdPlaceAddMember          string = "admin/place_add_member"
	CmdPlaceAddDefault         string = "admin/default_places_add"
	CmdPlacePromoteMember      string = "admin/place_promote_member"
	CmdPlaceDemoteMember       string = "admin/place_demote_member"
	CmdPlaceRemoveMember       string = "admin/place_remove_member"
	CmdPlaceGetDefault         string = "admin/default_places_get"
	CmdPlaceRemove             string = "admin/place_remove"
	CmdPlaceListMembers        string = "admin/place_list_members"
	CmdPlaceUpdate             string = "admin/place_update"
	CmdPlaceSetPicture         string = "admin/place_set_picture"
	CmdPlaceRemoveDefault      string = "admin/default_places_remove"
	CmdAccountRegister         string = "admin/account_register"
	CmdAccountSetPass          string = "admin/account_set_pass"
	CmdAccountDisable          string = "admin/account_disable"
	CmdAccountEnable           string = "admin/account_enable"
	CmdAccountList             string = "admin/account_list"
	CmdAccountListPlaces       string = "admin/account_list_places"
	CmdAccountUpdate           string = "admin/account_update"
	CmdAccountSetPicture       string = "admin/account_set_picture"
	CmdAccountSetDefaultPlaces string = "admin/default_places_set_users"
	CmdAccountRemovePicture    string = "admin/account_remove_picture"
	CmdAccountPostToAll        string = "admin/create_post_for_all_accounts"
	CmdSetMessageTemplate      string = "admin/set_message_template"
	CmdGetMessageTemplates     string = "admin/get_message_templates"
	CmdRemoveMessageTemplate   string = "admin/remove_message_template"
)

type AdminService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewAdminService(worker *api.Worker) *AdminService {
	s := new(AdminService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CmdCreatePost:              {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.createPost},
		CmdAddComment:              {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.addComment},
		CmdPromote:                 {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.promoteAccount},
		CmdDemote:                  {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.demoteAccount},
		CmdAccountRegister:         {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.createAccount},
		CmdAccountSetPass:          {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setAccountPassword},
		CmdAccountDisable:          {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.disableAccount},
		CmdAccountEnable:           {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.enableAccount},
		CmdAccountList:             {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.listAccounts},
		CmdAccountListPlaces:       {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.listPlacesOfAccount},
		CmdAccountUpdate:           {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.updateAccount},
		CmdAccountSetPicture:       {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setAccountProfilePicture},
		CmdAccountRemovePicture:    {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.removeAccountProfilePicture},
		CmdAccountSetDefaultPlaces: {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.defaultPlacesSetUsers},
		CmdAccountPostToAll:        {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.createPostForAllAccounts},
		CmdHealthCheck:             {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.checkSystemHealth},
		CmdPlaceCreate:             {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.createPlace},
		CmdPlaceCreateGrand:        {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.createGrandPlace},
		CmdPlaceList:               {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.listPlaces},
		CmdPlaceListMembers:        {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.listPlaceMembers},
		CmdPlaceAddMember:          {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.addPlaceMember},
		CmdPlacePromoteMember:      {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.promotePlaceMember},
		CmdPlaceDemoteMember:       {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.demotePlaceMember},
		CmdPlaceRemoveMember:       {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.removePlaceMember},
		CmdPlaceRemove:             {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.removePlace},
		CmdPlaceUpdate:             {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.updatePlace},
		CmdPlaceSetPicture:         {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setPlaceProfilePicture},
		CmdPlaceAddDefault:         {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.addDefaultPlaces},
		CmdPlaceGetDefault:         {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.getDefaultPlaces},
		CmdPlaceRemoveDefault:      {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.removeDefaultPlaces},
		CmdSetMessageTemplate:      {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setMessageTemplate},
		CmdGetMessageTemplates:     {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.getMessageTemplates},
		CmdRemoveMessageTemplate:   {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.removeMessageTemplates},
	}
	return s
}

func (s *AdminService) GetServicePrefix() string {
	return ServicePrefix
}

func (s *AdminService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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

func (s *AdminService) Worker() *api.Worker {
	return s.worker
}
