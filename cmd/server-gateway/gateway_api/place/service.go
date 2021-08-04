package nestedServicePlace

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/model"
	"git.ronaksoft.com/nested/server/pkg/rpc"
)

const (
	SERVICE_PREFIX = "place"
)
const (
	CMD_AVAILABLE             = "place/available"
	CMD_ADD_UNLOCKED_PLACE    = "place/add_unlocked_place"
	CMD_ADD_LOCKED_PLACE      = "place/add_locked_place"
	CMD_ADD_GRAND_PLACE       = "place/add_grand_place"
	CMD_ADD_FAVORITE          = "place/add_favorite"
	CMD_ADD_MEMBER            = "place/add_member"
	CMD_ADD_TO_BLACKLIST      = "place/add_to_blacklist"
	CMD_COUNT_UNREAD_POSTS    = "place/count_unread_posts"
	CMD_GET                   = "place/get"
	CMD_GET_MANY              = "place/get_many"
	CMD_GET_ACCESS            = "place/get_access"
	CMD_GET_POSTS             = "place/get_posts"
	CMD_GET_FILES             = "place/get_files"
	CMD_GET_UNREAD_POSTS      = "place/get_unread_posts"
	CMD_GET_KEYHOLDERS        = "place/get_key_holders"
	CMD_GET_CREATORS          = "place/get_creators"
	CMD_GET_MEMBERS           = "place/get_members"
	CMD_GET_SUBPLACES         = "place/get_sub_places"
	CMD_GET_MUTUAL_PLACES     = "place/get_mutual_places"
	CMD_GET_NOTIFICATION      = "place/get_notification"
	CMD_GET_ACTIVITIES        = "place/get_activities"
	CMD_GET_BLOCKED_ADDRESSES = "place/get_blocked_addresses"
	CMD_LEAVE                 = "place/leave"
	CMD_MARK_ALL_READ         = "place/mark_all_read"
	CMD_REMOVE                = "place/remove"
	CMD_REMOVE_MEMBER         = "place/remove_member"
	CMD_REMOVE_PICTURE        = "place/remove_picture"
	CMD_REMOVE_FAVORITE       = "place/remove_favorite"
	CMD_REMOVE_FROM_BLACKLIST = "place/remove_from_blacklist"
	CMD_SET_PICTURE           = "place/set_picture"
	CMD_SET_NOTIFICATION      = "place/set_notification"
	CMD_PROMOTE_MEMBER        = "place/promote_member"
	CMD_PIN_POST              = "place/pin_post"
	CMD_UNPIN_POST            = "place/unpin_post"
	CMD_DEMOTE_MEMBER         = "place/demote_member"
	CMD_INVITE_MEMBER         = "place/invite_member"
	CMD_UPDATE                = "place/update"
)

type PlaceService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewPlaceService(worker *api.Worker) *PlaceService {
	s := new(PlaceService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_AVAILABLE:             {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.placeIDAvailable},
		CMD_ADD_LOCKED_PLACE:      {MinAuthLevel: api.AuthLevelUser, Execute: s.createLockedPlace},
		CMD_ADD_UNLOCKED_PLACE:    {MinAuthLevel: api.AuthLevelUser, Execute: s.createUnlockedPlace},
		CMD_ADD_GRAND_PLACE:       {MinAuthLevel: api.AuthLevelUser, Execute: s.createGrandPlace},
		CMD_ADD_FAVORITE:          {MinAuthLevel: api.AuthLevelUser, Execute: s.setPlaceAsFavorite},
		CMD_ADD_MEMBER:            {MinAuthLevel: api.AuthLevelUser, Execute: s.addPlaceMember},
		CMD_COUNT_UNREAD_POSTS:    {MinAuthLevel: api.AuthLevelUser, Execute: s.countPlaceUnreadPosts},
		CMD_DEMOTE_MEMBER:         {MinAuthLevel: api.AuthLevelUser, Execute: s.demoteMember},
		CMD_GET:                   {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceInfo},
		CMD_GET_MANY:              {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getManyPlacesInfo},
		CMD_GET_ACCESS:            {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceAccess},
		CMD_GET_FILES:             {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceFiles},
		CMD_GET_CREATORS:          {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceCreators},
		CMD_GET_KEYHOLDERS:        {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceKeyholders},
		CMD_GET_MEMBERS:           {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceMembers},
		CMD_GET_SUBPLACES:         {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getSubPlaces},
		CMD_GET_POSTS:             {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlacePosts},
		CMD_GET_ACTIVITIES:        {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceActivities},
		CMD_GET_UNREAD_POSTS:      {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getPlaceUnreadPosts},
		CMD_GET_MUTUAL_PLACES:     {MinAuthLevel: api.AuthLevelAppL3, Execute: s.getMutualPlaces},
		CMD_GET_NOTIFICATION:      {MinAuthLevel: api.AuthLevelUser, Execute: s.getPlaceNotification},
		CMD_GET_BLOCKED_ADDRESSES: {MinAuthLevel: api.AuthLevelUser, Execute: s.getBlockedAddresses},
		CMD_INVITE_MEMBER:         {MinAuthLevel: api.AuthLevelUser, Execute: s.invitePlaceMember},
		CMD_MARK_ALL_READ:         {MinAuthLevel: api.AuthLevelAppL3, Execute: s.markAllPostsAsRead},
		CMD_PROMOTE_MEMBER:        {MinAuthLevel: api.AuthLevelUser, Execute: s.promoteMember},
		CMD_REMOVE:                {MinAuthLevel: api.AuthLevelUser, Execute: s.remove},
		CMD_REMOVE_MEMBER:         {MinAuthLevel: api.AuthLevelUser, Execute: s.removeMember},
		CMD_LEAVE:                 {MinAuthLevel: api.AuthLevelUser, Execute: s.leavePlace},
		CMD_REMOVE_FAVORITE:       {MinAuthLevel: api.AuthLevelUser, Execute: s.removePlaceFromFavorites},
		CMD_REMOVE_PICTURE:        {MinAuthLevel: api.AuthLevelUser, Execute: s.removePicture},
		CMD_SET_PICTURE:           {MinAuthLevel: api.AuthLevelAppL3, Execute: s.setPicture},
		CMD_SET_NOTIFICATION:      {MinAuthLevel: api.AuthLevelAppL3, Execute: s.setPlaceNotification},
		CMD_UPDATE:                {MinAuthLevel: api.AuthLevelUser, Execute: s.update},
		CMD_PIN_POST:              {MinAuthLevel: api.AuthLevelAppL3, Execute: s.pinPost},
		CMD_UNPIN_POST:            {MinAuthLevel: api.AuthLevelAppL3, Execute: s.unpinPost},
		CMD_ADD_TO_BLACKLIST:      {MinAuthLevel: api.AuthLevelUser, Execute: s.addToBlackList},
		CMD_REMOVE_FROM_BLACKLIST: {MinAuthLevel: api.AuthLevelUser, Execute: s.removeFromBlacklist},
	}

	return s
}

func (s *PlaceService) GetServicePrefix() string {
	return SERVICE_PREFIX
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
