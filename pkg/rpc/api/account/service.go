package nestedServiceAccount

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	SERVICE_PREFIX = "account"
)
const (
	CmdAvailable          = "account/available"
	CmdChangePhone        = "account/change_phone"
	CmdGet                = "account/get"
	CmdGetByToken         = "account/get_by_token"
	CmdGetMany            = "account/get_many"
	CmdGetAllPlaces       = "account/get_all_places"
	CmdGetFavoritePlaces  = "account/get_favorite_places"
	CmdGetPosts           = "account/get_posts"
	CmdGetSpamPosts       = "account/get_spam_posts"
	CmdGetFavoritePosts   = "account/get_favorite_posts"
	CmdGetSentPosts       = "account/get_sent_posts"
	CmdGetPinnedPosts     = "account/get_pinned_posts"
	CmdRegisterDevice     = "account/register_device"
	CmdRemovePicture      = "account/remove_picture"
	CmdSetPicture         = "account/set_picture"
	CmdSetPassword        = "account/set_password"
	CmdSetPasswordByToken = "account/set_password_by_token"
	CmdTrustEmail         = "account/trust_email"
	CmdUnregisterDevice   = "account/unregister_device"
	CmdUnTrustEmail       = "account/untrust_email"
	CmdUpdate             = "account/update"
	CmdUpdateEmail        = "account/update_email"
	CmdRemoveEmail        = "account/remove_email"
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
		CmdUpdateEmail:        {MinAuthLevel: api.AuthLevelAppL1, Execute: s.updateEmail},
		CmdRemoveEmail:        {MinAuthLevel: api.AuthLevelAppL1, Execute: s.removeEmail},
		CmdGetAllPlaces:       {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountAllPlaces},
		CmdGetFavoritePlaces:  {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountFavoritePlaces},
		CmdGetPosts:           {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountFavoritePosts},
		CmdGetSpamPosts:       {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountSpamPosts},
		CmdGetFavoritePosts:   {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountFavoritePosts},
		CmdGetSentPosts:       {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountSentPosts},
		CmdGetPinnedPosts:     {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getAccountPinnedPosts},
		CmdChangePhone:        {MinAuthLevel: api.AuthLevelUser, Execute: s.changePhone},
		CmdGet:                {MinAuthLevel: api.AuthLevelUser, Execute: s.getAccountInfo},
		CmdGetMany:            {MinAuthLevel: api.AuthLevelUser, Execute: s.getManyAccountsInfo},
		CmdSetPicture:         {MinAuthLevel: api.AuthLevelUser, Execute: s.setAccountPicture},
		CmdTrustEmail:         {MinAuthLevel: api.AuthLevelUser, Execute: s.addToTrustList},
		CmdRemovePicture:      {MinAuthLevel: api.AuthLevelUser, Execute: s.removeAccountPicture},
		CmdRegisterDevice:     {MinAuthLevel: api.AuthLevelUser, Execute: s.registerDevice},
		CmdUnregisterDevice:   {MinAuthLevel: api.AuthLevelUser, Execute: s.unregisterDevice},
		CmdUnTrustEmail:       {MinAuthLevel: api.AuthLevelUser, Execute: s.removeFromTrustList},
		CmdUpdate:             {MinAuthLevel: api.AuthLevelUser, Execute: s.updateAccount},
		CmdAvailable:          {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.accountIDAvailable},
		CmdGetByToken:         {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getAccountInfoByToken},
		CmdSetPassword:        {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.setAccountPassword},
		CmdSetPasswordByToken: {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.setAccountPasswordByLoginToken},
	}

	_Model = s.worker.Model()
	return s
}

func (s *AccountService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *AccountService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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
