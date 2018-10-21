package nestedServiceAdmin

import (
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server/server-gateway/client"
    "git.ronaksoftware.com/nested/server/server-gateway/gateway_api"
)

const (
    SERVICE_PREFIX string = "admin"
)
const (
    CMD_CREATE_POST             string = "admin/create_post"
    CMD_ADD_COMMENT             string = "admin/add_comment"
    CMD_HEALTH_CHECK            string = "admin/health_check"
    CMD_PROMOTE                 string = "admin/promote"
    CMD_DEMOTE                  string = "admin/demote"
    CMD_PLACE_CREATE_GRAND      string = "admin/create_grand_place"
    CMD_PLACE_CREATE            string = "admin/create_place"
    CMD_PLACE_LIST              string = "admin/place_list"
    CMD_PLACE_ADD_MEMBER        string = "admin/place_add_member"
    CMD_PLACE_ADD_DEFAULT       string = "admin/add_default_places"
    CMD_PLACE_PROMOTE_MEMBER    string = "admin/place_promote_member"
    CMD_PLACE_DEMOTE_MEMBER     string = "admin/place_demote_member"
    CMD_PLACE_REMOVE_MEMBER     string = "admin/place_remove_member"
    CMD_PLACE_GET_DEFAULT       string = "admin/get_default_places"
    CMD_PLACE_REMOVE            string = "admin/place_remove"
    CMD_PLACE_LIST_MEMBERS      string = "admin/place_list_members"
    CMD_PLACE_UPDATE            string = "admin/place_update"
    CMD_PLACE_SET_PICTURE       string = "admin/place_set_picture"
    CMD_PLACE_REMOVE_DEFAULT    string = "admin/remove_default_places"
    CMD_ACCOUNT_REGISTER        string = "admin/account_register"
    CMD_ACCOUNT_SET_PASS        string = "admin/account_set_pass"
    CMD_ACCOUNT_DISABLE         string = "admin/account_disable"
    CMD_ACCOUNT_ENABLE          string = "admin/account_enable"
    CMD_ACCOUNT_LIST            string = "admin/account_list"
    CMD_ACCOUNT_LIST_PLACES     string = "admin/account_list_places"
    CMD_ACCOUNT_UPDATE          string = "admin/account_update"
    CMD_ACCOUNT_SET_PICTURE     string = "admin/account_set_picture"
    CMD_ACCOUNT_JOIN_PLACES     string = "admin/account_join_places"
    CMD_ACCOUNT_REMOVE_PICTURE  string = "admin/account_remove_picture"
    CMD_ACCOUNT_POST_TO_ALL     string = "admin/create_post_for_all_accounts"
    CMD_SET_MESSAGE_TEMPLATE    string = "admin/set_message_template"
    CMD_GET_MESSAGE_TEMPLATES   string = "admin/get_message_templates"
    CMD_REMOVE_MESSAGE_TEMPLATE string = "admin/remove_message_template"
)

type AdminService struct {
    worker          *api.Worker
    serviceCommands api.ServiceCommands
}

func NewAdminService(worker *api.Worker) *AdminService {
    s := new(AdminService)
    s.worker = worker

    s.serviceCommands = api.ServiceCommands{
        CMD_CREATE_POST:             {api.AUTH_LEVEL_ADMIN_USER, s.createPost},
        CMD_ADD_COMMENT:             {api.AUTH_LEVEL_ADMIN_USER, s.addComment},
        CMD_PROMOTE:                 {api.AUTH_LEVEL_ADMIN_USER, s.promoteAccount},
        CMD_DEMOTE:                  {api.AUTH_LEVEL_ADMIN_USER, s.demoteAccount},
        CMD_ACCOUNT_REGISTER:        {api.AUTH_LEVEL_ADMIN_USER, s.createAccount},
        CMD_ACCOUNT_SET_PASS:        {api.AUTH_LEVEL_ADMIN_USER, s.setAccountPassword},
        CMD_ACCOUNT_DISABLE:         {api.AUTH_LEVEL_ADMIN_USER, s.disableAccount},
        CMD_ACCOUNT_ENABLE:          {api.AUTH_LEVEL_ADMIN_USER, s.enableAccount},
        CMD_ACCOUNT_LIST:            {api.AUTH_LEVEL_ADMIN_USER, s.listAccounts},
        CMD_ACCOUNT_LIST_PLACES:     {api.AUTH_LEVEL_ADMIN_USER, s.listPlacesOfAccount},
        CMD_ACCOUNT_UPDATE:          {api.AUTH_LEVEL_ADMIN_USER, s.updateAccount},
        CMD_ACCOUNT_SET_PICTURE:     {api.AUTH_LEVEL_ADMIN_USER, s.setAccountProfilePicture},
        CMD_ACCOUNT_REMOVE_PICTURE:  {api.AUTH_LEVEL_ADMIN_USER, s.removeAccountProfilePicture},
        CMD_ACCOUNT_JOIN_PLACES:     {api.AUTH_LEVEL_ADMIN_USER, s.accountJoinDefaultPlaces},
        CMD_ACCOUNT_POST_TO_ALL:     {api.AUTH_LEVEL_ADMIN_USER, s.createPostForAllAccounts},
        CMD_HEALTH_CHECK:            {api.AUTH_LEVEL_ADMIN_USER, s.checkSystemHealth},
        CMD_PLACE_CREATE:            {api.AUTH_LEVEL_ADMIN_USER, s.createPlace},
        CMD_PLACE_CREATE_GRAND:      {api.AUTH_LEVEL_ADMIN_USER, s.createGrandPlace},
        CMD_PLACE_LIST:              {api.AUTH_LEVEL_ADMIN_USER, s.listPlaces},
        CMD_PLACE_LIST_MEMBERS:      {api.AUTH_LEVEL_ADMIN_USER, s.listPlaceMembers},
        CMD_PLACE_ADD_MEMBER:        {api.AUTH_LEVEL_ADMIN_USER, s.addPlaceMember},
        CMD_PLACE_PROMOTE_MEMBER:    {api.AUTH_LEVEL_ADMIN_USER, s.promotePlaceMember},
        CMD_PLACE_DEMOTE_MEMBER:     {api.AUTH_LEVEL_ADMIN_USER, s.demotePlaceMember},
        CMD_PLACE_REMOVE_MEMBER:     {api.AUTH_LEVEL_ADMIN_USER, s.removePlaceMember},
        CMD_PLACE_REMOVE:            {api.AUTH_LEVEL_ADMIN_USER, s.removePlace},
        CMD_PLACE_UPDATE:            {api.AUTH_LEVEL_ADMIN_USER, s.updatePlace},
        CMD_PLACE_SET_PICTURE:       {api.AUTH_LEVEL_ADMIN_USER, s.setPlaceProfilePicture},
        CMD_PLACE_ADD_DEFAULT:       {api.AUTH_LEVEL_ADMIN_USER, s.addDefaultPlaces},
        CMD_PLACE_GET_DEFAULT:       {api.AUTH_LEVEL_ADMIN_USER, s.getDefaultPlaces},
        CMD_PLACE_REMOVE_DEFAULT:    {api.AUTH_LEVEL_ADMIN_USER, s.removeDefaultPlaces},
        CMD_SET_MESSAGE_TEMPLATE:    {api.AUTH_LEVEL_ADMIN_USER, s.setMessageTemplate},
        CMD_GET_MESSAGE_TEMPLATES:   {api.AUTH_LEVEL_ADMIN_USER, s.getMessageTemplates},
        CMD_REMOVE_MESSAGE_TEMPLATE: {api.AUTH_LEVEL_ADMIN_USER, s.removeMessageTemplates},
    }
    return s
}

func (s *AdminService) GetServicePrefix() string {
    return SERVICE_PREFIX
}

func (s *AdminService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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
