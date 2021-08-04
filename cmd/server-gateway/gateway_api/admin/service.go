package nestedServiceAdmin

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/model"
	"git.ronaksoft.com/nested/server/pkg/rpc"
)

const (
	SERVICE_PREFIX string = "admin"
)
const (
	CMD_CREATE_POST                string = "admin/create_post"
	CMD_ADD_COMMENT                string = "admin/add_comment"
	CMD_HEALTH_CHECK               string = "admin/health_check"
	CMD_PROMOTE                    string = "admin/promote"
	CMD_DEMOTE                     string = "admin/demote"
	CMD_PLACE_CREATE_GRAND         string = "admin/create_grand_place"
	CMD_PLACE_CREATE               string = "admin/create_place"
	CMD_PLACE_LIST                 string = "admin/place_list"
	CMD_PLACE_ADD_MEMBER           string = "admin/place_add_member"
	CMD_PLACE_ADD_DEFAULT          string = "admin/default_places_add"
	CMD_PLACE_PROMOTE_MEMBER       string = "admin/place_promote_member"
	CMD_PLACE_DEMOTE_MEMBER        string = "admin/place_demote_member"
	CMD_PLACE_REMOVE_MEMBER        string = "admin/place_remove_member"
	CMD_PLACE_GET_DEFAULT          string = "admin/default_places_get"
	CMD_PLACE_REMOVE               string = "admin/place_remove"
	CMD_PLACE_LIST_MEMBERS         string = "admin/place_list_members"
	CMD_PLACE_UPDATE               string = "admin/place_update"
	CMD_PLACE_SET_PICTURE          string = "admin/place_set_picture"
	CMD_PLACE_REMOVE_DEFAULT       string = "admin/default_places_remove"
	CMD_ACCOUNT_REGISTER           string = "admin/account_register"
	CMD_ACCOUNT_SET_PASS           string = "admin/account_set_pass"
	CMD_ACCOUNT_DISABLE            string = "admin/account_disable"
	CMD_ACCOUNT_ENABLE             string = "admin/account_enable"
	CMD_ACCOUNT_LIST               string = "admin/account_list"
	CMD_ACCOUNT_LIST_PLACES        string = "admin/account_list_places"
	CMD_ACCOUNT_UPDATE             string = "admin/account_update"
	CMD_ACCOUNT_SET_PICTURE        string = "admin/account_set_picture"
	CMD_ACCOUNT_SET_DEFAULT_PLACES string = "admin/default_places_set_users"
	CMD_ACCOUNT_REMOVE_PICTURE     string = "admin/account_remove_picture"
	CMD_ACCOUNT_POST_TO_ALL        string = "admin/create_post_for_all_accounts"
	CMD_SET_MESSAGE_TEMPLATE       string = "admin/set_message_template"
	CMD_GET_MESSAGE_TEMPLATES      string = "admin/get_message_templates"
	CMD_REMOVE_MESSAGE_TEMPLATE    string = "admin/remove_message_template"
)

type AdminService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewAdminService(worker *api.Worker) *AdminService {
	s := new(AdminService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_CREATE_POST:                {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.createPost},
		CMD_ADD_COMMENT:                {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.addComment},
		CMD_PROMOTE:                    {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.promoteAccount},
		CMD_DEMOTE:                     {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.demoteAccount},
		CMD_ACCOUNT_REGISTER:           {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.createAccount},
		CMD_ACCOUNT_SET_PASS:           {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setAccountPassword},
		CMD_ACCOUNT_DISABLE:            {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.disableAccount},
		CMD_ACCOUNT_ENABLE:             {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.enableAccount},
		CMD_ACCOUNT_LIST:               {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.listAccounts},
		CMD_ACCOUNT_LIST_PLACES:        {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.listPlacesOfAccount},
		CMD_ACCOUNT_UPDATE:             {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.updateAccount},
		CMD_ACCOUNT_SET_PICTURE:        {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setAccountProfilePicture},
		CMD_ACCOUNT_REMOVE_PICTURE:     {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.removeAccountProfilePicture},
		CMD_ACCOUNT_SET_DEFAULT_PLACES: {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.defaultPlacesSetUsers},
		CMD_ACCOUNT_POST_TO_ALL:        {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.createPostForAllAccounts},
		CMD_HEALTH_CHECK:               {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.checkSystemHealth},
		CMD_PLACE_CREATE:               {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.createPlace},
		CMD_PLACE_CREATE_GRAND:         {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.createGrandPlace},
		CMD_PLACE_LIST:                 {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.listPlaces},
		CMD_PLACE_LIST_MEMBERS:         {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.listPlaceMembers},
		CMD_PLACE_ADD_MEMBER:           {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.addPlaceMember},
		CMD_PLACE_PROMOTE_MEMBER:       {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.promotePlaceMember},
		CMD_PLACE_DEMOTE_MEMBER:        {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.demotePlaceMember},
		CMD_PLACE_REMOVE_MEMBER:        {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.removePlaceMember},
		CMD_PLACE_REMOVE:               {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.removePlace},
		CMD_PLACE_UPDATE:               {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.updatePlace},
		CMD_PLACE_SET_PICTURE:          {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setPlaceProfilePicture},
		CMD_PLACE_ADD_DEFAULT:          {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.addDefaultPlaces},
		CMD_PLACE_GET_DEFAULT:          {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.getDefaultPlaces},
		CMD_PLACE_REMOVE_DEFAULT:       {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.removeDefaultPlaces},
		CMD_SET_MESSAGE_TEMPLATE:       {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.setMessageTemplate},
		CMD_GET_MESSAGE_TEMPLATES:      {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.getMessageTemplates},
		CMD_REMOVE_MESSAGE_TEMPLATE:    {MinAuthLevel: api.AuthLevelAdminUser, Execute: s.removeMessageTemplates},
	}
	return s
}

func (s *AdminService) GetServicePrefix() string {
	return SERVICE_PREFIX
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
