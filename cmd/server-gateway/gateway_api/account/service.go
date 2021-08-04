package nestedServiceAccount

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/model"
)

const (
	SERVICE_PREFIX = "account"
)
const (
	CMD_AVAILABLE             = "account/available"
	CMD_CHANGE_PHONE          = "account/change_phone"
	CMD_GET                   = "account/get"
	CMD_GET_BY_TOKEN          = "account/get_by_token"
	CMD_GET_MANY              = "account/get_many"
	CMD_GET_ALL_PLACES        = "account/get_all_places"
	CMD_GET_FAVORITE_PLACES   = "account/get_favorite_places"
	CMD_GET_POSTS             = "account/get_posts"
	CMD_GET_FAVORITE_POSTS    = "account/get_favorite_posts"
	CMD_GET_SENT_POSTS        = "account/get_sent_posts"
	CMD_GET_PINNED_POSTS      = "account/get_pinned_posts"
	CMD_REGISTER_DEVICE       = "account/register_device"
	CMD_REMOVE_PICTURE        = "account/remove_picture"
	CMD_SET_PICTURE           = "account/set_picture"
	CMD_SET_PASSWORD          = "account/set_password"
	CMD_SET_PASSWORD_BY_TOKEN = "account/set_password_by_token"
	CMD_TRUST_EMAIL           = "account/trust_email"
	CMD_UNREGISTER_DEVICE     = "account/unregister_device"
	CMD_UN_TRUST_EMAIL        = "account/untrust_email"
	CMD_UPDATE                = "account/update"
	CMD_UPDATE_EMAIL          = "account/update_email"
	CMD_REMOVE_EMAIL          = "account/remove_email"
)

var (
	_Model *nested.Manager
)

type AccountService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewAccountService(worker *api.Worker) *AccountService {
	s := new(AccountService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_UPDATE_EMAIL:          {MinAuthLevel: api.AuthLevelAppL1, Execute: s.updateEmail},
		CMD_REMOVE_EMAIL:          {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeEmail},
		CMD_GET_ALL_PLACES:        {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountAllPlaces},
		CMD_GET_FAVORITE_PLACES:   {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountFavoritePlaces},
		CMD_GET_POSTS:             {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountFavoritePosts},
		CMD_GET_FAVORITE_POSTS:    {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountFavoritePosts},
		CMD_GET_SENT_POSTS:        {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountSentPosts},
		CMD_GET_PINNED_POSTS:      {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountPinnedPosts},
		CMD_CHANGE_PHONE:          {MinAuthLevel: api.AuthLevelUser, Execute: s.changePhone},
		CMD_GET:                   {MinAuthLevel: api.AuthLevelUser, Execute: s.getAccountInfo},
		CMD_GET_MANY:              {MinAuthLevel: api.AuthLevelUser, Execute: s.getManyAccountsInfo},
		CMD_SET_PICTURE:           {MinAuthLevel: api.AuthLevelUser, Execute: s.setAccountPicture},
		CMD_TRUST_EMAIL:           {MinAuthLevel: api.AuthLevelUser, Execute: s.addToTrustList},
		CMD_REMOVE_PICTURE:        {MinAuthLevel: api.AuthLevelUser, Execute: s.removeAccountPicture},
		CMD_REGISTER_DEVICE:       {MinAuthLevel: api.AuthLevelUser, Execute: s.registerDevice},
		CMD_UNREGISTER_DEVICE:     {MinAuthLevel: api.AuthLevelUser, Execute: s.unregisterDevice},
		CMD_UN_TRUST_EMAIL:        {MinAuthLevel: api.AuthLevelUser, Execute: s.removeFromTrustList},
		CMD_UPDATE:                {MinAuthLevel: api.AuthLevelUser, Execute: s.updateAccount},
		CMD_AVAILABLE:             {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.accountIDAvailable},
		CMD_GET_BY_TOKEN:          {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getAccountInfoByToken},
		CMD_SET_PASSWORD:          {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.setAccountPassword},
		CMD_SET_PASSWORD_BY_TOKEN: {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.setAccountPasswordByLoginToken},
	}

	_Model = s.worker.Model()
	return s
}

func (s *AccountService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *AccountService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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

func (s *AccountService) Worker() *api.Worker {
	return s.worker
}
