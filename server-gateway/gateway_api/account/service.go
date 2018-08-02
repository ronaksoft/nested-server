package nestedServiceAccount

import (
    "git.ronaksoftware.com/nested/server-model-nested"
    "git.ronaksoftware.com/nested/server-gateway/client"
    "git.ronaksoftware.com/nested/server-gateway/gateway_api"
)

const (
    SERVICE_PREFIX = "account"
)
const (
    CMD_AVAILABLE              = "account/available"
    CMD_CHANGE_PHONE           = "account/change_phone"
    CMD_GET                    = "account/get"
    CMD_GET_BY_TOKEN           = "account/get_by_token"
    CMD_GET_MANY               = "account/get_many"
    CMD_GET_ALL_PLACES         = "account/get_all_places"
    CMD_GET_FAVORITE_PLACES    = "account/get_favorite_places"
    CMD_GET_POSTS              = "account/get_posts"
    CMD_GET_FAVORITE_POSTS     = "account/get_favorite_posts"
    CMD_GET_SENT_POSTS         = "account/get_sent_posts"
    CMD_GET_PINNED_POSTS       = "account/get_pinned_posts"
    CMD_REGISTER_DEVICE        = "account/register_device"
    CMD_REMOVE_PICTURE         = "account/remove_picture"
    CMD_SET_PICTURE            = "account/set_picture"
    CMD_SET_PASSWORD           = "account/set_password"
    CMD_SET_PASSWORD_BY_TOKEN  = "account/set_password_by_token"
    CMD_TRUST_EMAIL            = "account/trust_email"
    CMD_UNREGISTER_DEVICE      = "account/unregister_device"
    CMD_UN_TRUST_EMAIL         = "account/untrust_email"
    CMD_UPDATE                 = "account/update"
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
        CMD_GET_ALL_PLACES:        {api.AUTH_LEVEL_APP_L3, s.getAccountAllPlaces},
        CMD_GET_FAVORITE_PLACES:   {api.AUTH_LEVEL_APP_L3, s.getAccountFavoritePlaces},
        CMD_GET_POSTS:             {api.AUTH_LEVEL_APP_L3, s.getAccountFavoritePosts},
        CMD_GET_FAVORITE_POSTS:    {api.AUTH_LEVEL_APP_L3, s.getAccountFavoritePosts},
        CMD_GET_SENT_POSTS:        {api.AUTH_LEVEL_APP_L3, s.getAccountSentPosts},
        CMD_GET_PINNED_POSTS:      {api.AUTH_LEVEL_APP_L3, s.getAccountPinnedPosts},
        CMD_CHANGE_PHONE:          {api.AUTH_LEVEL_USER, s.changePhone},
        CMD_GET:                   {api.AUTH_LEVEL_USER, s.getAccountInfo},
        CMD_GET_MANY:              {api.AUTH_LEVEL_USER, s.getManyAccountsInfo},
        CMD_SET_PICTURE:           {api.AUTH_LEVEL_USER, s.setAccountPicture},
        CMD_TRUST_EMAIL:           {api.AUTH_LEVEL_USER, s.addToTrustList},
        CMD_REMOVE_PICTURE:        {api.AUTH_LEVEL_USER, s.removeAccountPicture},
        CMD_REGISTER_DEVICE:       {api.AUTH_LEVEL_USER, s.registerDevice},
        CMD_UNREGISTER_DEVICE:     {api.AUTH_LEVEL_USER, s.unregisterDevice},
        CMD_UN_TRUST_EMAIL:        {api.AUTH_LEVEL_USER, s.removeFromTrustList},
        CMD_UPDATE:                {api.AUTH_LEVEL_USER, s.updateAccount},
        CMD_AVAILABLE:             {api.AUTH_LEVEL_UNAUTHORIZED, s.accountIDAvailable},
        CMD_GET_BY_TOKEN:          {api.AUTH_LEVEL_UNAUTHORIZED, s.getAccountInfoByToken},
        CMD_SET_PASSWORD:          {api.AUTH_LEVEL_UNAUTHORIZED, s.setAccountPassword},
        CMD_SET_PASSWORD_BY_TOKEN: {api.AUTH_LEVEL_UNAUTHORIZED, s.setAccountPasswordByLoginToken},
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
