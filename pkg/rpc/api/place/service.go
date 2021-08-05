package nestedServicePlace

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix = "place"
)
const (
	CmdAvailable           = "place/available"
	CmdAddUnlockedPlace    = "place/add_unlocked_place"
	CmdAddLockedPlace      = "place/add_locked_place"
	CmdAddGrandPlace       = "place/add_grand_place"
	CmdAddFavorite         = "place/add_favorite"
	CmdAddMember           = "place/add_member"
	CmdAddToBlacklist      = "place/add_to_blacklist"
	CmdCountUnreadPosts    = "place/count_unread_posts"
	CmdGet                 = "place/get"
	CmdGetMany             = "place/get_many"
	CmdGetAccess           = "place/get_access"
	CmdGetPosts            = "place/get_posts"
	CmdGetFiles            = "place/get_files"
	GetUnreadPosts         = "place/get_unread_posts"
	CmdGetKeyholders       = "place/get_key_holders"
	CmdGetCreators         = "place/get_creators"
	CmdGetMembers          = "place/get_members"
	CmdGetSubplaces        = "place/get_sub_places"
	CmdGetMutualPlaces     = "place/get_mutual_places"
	CmdGetNotification     = "place/get_notification"
	CmdGetActivities       = "place/get_activities"
	CmdGetBlockedAddresses = "place/get_blocked_addresses"
	CmdLeave               = "place/leave"
	CmdMarkAllRead         = "place/mark_all_read"
	CmdRemove              = "place/remove"
	CmdRemoveMember        = "place/remove_member"
	CmdRemovePicture       = "place/remove_picture"
	CmdRemoveFavorite      = "place/remove_favorite"
	CmdRemoveFromBlacklist = "place/remove_from_blacklist"
	CmdSetPicture          = "place/set_picture"
	CmdSetNotification     = "place/set_notification"
	CmdPromoteMember       = "place/promote_member"
	CmdPinPost             = "place/pin_post"
	CmdUnpinPost           = "place/unpin_post"
	CmdDemoteMember        = "place/demote_member"
	CmdInviteMember        = "place/invite_member"
	CmdUpdate              = "place/update"
)

type PlaceService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewPlaceService(worker *api.Worker) *PlaceService {
	s := new(PlaceService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CmdAvailable:           {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.placeIDAvailable},
		CmdAddLockedPlace:      {MinAuthLevel: api.AuthLevelUser, Execute: s.createLockedPlace},
		CmdAddUnlockedPlace:    {MinAuthLevel: api.AuthLevelUser, Execute: s.createUnlockedPlace},
		CmdAddGrandPlace:       {MinAuthLevel: api.AuthLevelUser, Execute: s.createGrandPlace},
		CmdAddFavorite:         {MinAuthLevel: api.AuthLevelUser, Execute: s.setPlaceAsFavorite},
		CmdAddMember:           {MinAuthLevel: api.AuthLevelUser, Execute: s.addPlaceMember},
		CmdCountUnreadPosts:    {MinAuthLevel: api.AuthLevelUser, Execute: s.countPlaceUnreadPosts},
		CmdDemoteMember:        {MinAuthLevel: api.AuthLevelUser, Execute: s.demoteMember},
		CmdGet:                 {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceInfo},
		CmdGetMany:             {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getManyPlacesInfo},
		CmdGetAccess:           {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceAccess},
		CmdGetFiles:            {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceFiles},
		CmdGetCreators:         {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceCreators},
		CmdGetKeyholders:       {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceKeyholders},
		CmdGetMembers:          {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceMembers},
		CmdGetSubplaces:        {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getSubPlaces},
		CmdGetPosts:            {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlacePosts},
		CmdGetActivities:       {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceActivities},
		GetUnreadPosts:         {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceUnreadPosts},
		CmdGetMutualPlaces:     {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getMutualPlaces},
		CmdGetNotification:     {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceNotification},
		CmdGetBlockedAddresses: {MinAuthLevel: api.AuthLevelUser, Execute: s.getBlockedAddresses},
		CmdInviteMember:        {MinAuthLevel: api.AuthLevelUser, Execute: s.invitePlaceMember},
		CmdMarkAllRead:         {MinAuthLevel: api.AuthLevelAppL3, Execute: s.markAllPostsAsRead},
		CmdPromoteMember:       {MinAuthLevel: api.AuthLevelUser, Execute: s.promoteMember},
		CmdRemove:              {MinAuthLevel: api.AuthLevelUser, Execute: s.remove},
		CmdRemoveMember:        {MinAuthLevel: api.AuthLevelUser, Execute: s.removeMember},
		CmdLeave:               {MinAuthLevel: api.AuthLevelUser, Execute: s.leavePlace},
		CmdRemoveFavorite:      {MinAuthLevel: api.AuthLevelUser, Execute: s.removePlaceFromFavorites},
		CmdRemovePicture:       {MinAuthLevel: api.AuthLevelUser, Execute: s.removePicture},
		CmdSetPicture:          {MinAuthLevel: api.AuthLevelAppL3, Execute: s.setPicture},
		CmdSetNotification:     {MinAuthLevel: api.AuthLevelAppL3, Execute: s.setPlaceNotification},
		CmdUpdate:              {MinAuthLevel: api.AuthLevelUser, Execute: s.update},
		CmdPinPost:             {MinAuthLevel: api.AuthLevelAppL3, Execute: s.pinPost},
		CmdUnpinPost:           {MinAuthLevel: api.AuthLevelAppL3, Execute: s.unpinPost},
		CmdAddToBlacklist:      {MinAuthLevel: api.AuthLevelUser, Execute: s.addToBlackList},
		CmdRemoveFromBlacklist: {MinAuthLevel: api.AuthLevelUser, Execute: s.removeFromBlacklist},
	}

	return s
}

func (s *PlaceService) GetServicePrefix() string {
	return ServicePrefix
}

func (s *PlaceService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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

func (s *PlaceService) Worker() *api.Worker {
	return s.worker
}
