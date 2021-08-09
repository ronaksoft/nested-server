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
	CmdGetKeyHolders       = "place/get_key_holders"
	CmdGetCreators         = "place/get_creators"
	CmdGetMembers          = "place/get_members"
	CmdGetSubPlaces        = "place/get_sub_places"
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
	CmdRemoveAllPosts      = "place/remove_all_posts"
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
		CmdAddFavorite:         {MinAuthLevel: api.AuthLevelUser, Execute: s.setPlaceAsFavorite},
		CmdAddGrandPlace:       {MinAuthLevel: api.AuthLevelUser, Execute: s.createGrandPlace},
		CmdAddLockedPlace:      {MinAuthLevel: api.AuthLevelUser, Execute: s.createLockedPlace},
		CmdAddMember:           {MinAuthLevel: api.AuthLevelUser, Execute: s.addPlaceMember},
		CmdAddToBlacklist:      {MinAuthLevel: api.AuthLevelUser, Execute: s.addToBlackList},
		CmdAddUnlockedPlace:    {MinAuthLevel: api.AuthLevelUser, Execute: s.createUnlockedPlace},
		CmdAvailable:           {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.placeIDAvailable},
		CmdCountUnreadPosts:    {MinAuthLevel: api.AuthLevelUser, Execute: s.countPlaceUnreadPosts},
		CmdDemoteMember:        {MinAuthLevel: api.AuthLevelUser, Execute: s.demoteMember},
		CmdGet:                 {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceInfo},
		CmdGetAccess:           {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceAccess},
		CmdGetActivities:       {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceActivities},
		CmdGetBlockedAddresses: {MinAuthLevel: api.AuthLevelUser, Execute: s.getBlockedAddresses},
		CmdGetCreators:         {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceCreators},
		CmdGetFiles:            {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceFiles},
		CmdGetKeyHolders:       {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceKeyholders},
		CmdGetMany:             {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getManyPlacesInfo},
		CmdGetMembers:          {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceMembers},
		CmdGetMutualPlaces:     {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getMutualPlaces},
		CmdGetNotification:     {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceNotification},
		CmdGetPosts:            {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlacePosts},
		CmdGetSubPlaces:        {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getSubPlaces},
		CmdInviteMember:        {MinAuthLevel: api.AuthLevelUser, Execute: s.invitePlaceMember},
		CmdLeave:               {MinAuthLevel: api.AuthLevelUser, Execute: s.leavePlace},
		CmdMarkAllRead:         {MinAuthLevel: api.AuthLevelAppL3, Execute: s.markAllPostsAsRead},
		CmdPinPost:             {MinAuthLevel: api.AuthLevelAppL3, Execute: s.pinPost},
		CmdPromoteMember:       {MinAuthLevel: api.AuthLevelUser, Execute: s.promoteMember},
		CmdRemove:              {MinAuthLevel: api.AuthLevelUser, Execute: s.remove},
		CmdRemoveAllPosts:      {MinAuthLevel: api.AuthLevelUser, Execute: s.removeAllPosts},
		CmdRemoveFavorite:      {MinAuthLevel: api.AuthLevelUser, Execute: s.removePlaceFromFavorites},
		CmdRemoveFromBlacklist: {MinAuthLevel: api.AuthLevelUser, Execute: s.removeFromBlacklist},
		CmdRemoveMember:        {MinAuthLevel: api.AuthLevelUser, Execute: s.removeMember},
		CmdRemovePicture:       {MinAuthLevel: api.AuthLevelUser, Execute: s.removePicture},
		CmdSetNotification:     {MinAuthLevel: api.AuthLevelAppL3, Execute: s.setPlaceNotification},
		CmdSetPicture:          {MinAuthLevel: api.AuthLevelAppL3, Execute: s.setPicture},
		CmdUnpinPost:           {MinAuthLevel: api.AuthLevelAppL3, Execute: s.unpinPost},
		CmdUpdate:              {MinAuthLevel: api.AuthLevelUser, Execute: s.update},
		GetUnreadPosts:         {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceUnreadPosts},
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
