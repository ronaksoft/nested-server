package nestedServicePlace

import (
    "git.ronaksoftware.com/nested/server/server-gateway/client"
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server/server-gateway/gateway_api"
)

const (
    SERVICE_PREFIX = "place"
)
const (
    CMD_AVAILABLE          = "place/available"
    CMD_ADD_UNLOCKED_PLACE = "place/add_unlocked_place"
    CMD_ADD_LOCKED_PLACE   = "place/add_locked_place"
    CMD_ADD_GRAND_PLACE    = "place/add_grand_place"
    CMD_ADD_FAVORITE       = "place/add_favorite"
    CMD_ADD_MEMBER         = "place/add_member"
    CMD_COUNT_UNREAD_POSTS = "place/count_unread_posts"
    CMD_GET                = "place/get"
    CMD_GET_MANY           = "place/get_many"
    CMD_GET_ACCESS         = "place/get_access"
    CMD_GET_POSTS          = "place/get_posts"
    CMD_GET_FILES          = "place/get_files"
    CMD_GET_UNREAD_POSTS   = "place/get_unread_posts"
    CMD_GET_KEYHOLDERS     = "place/get_key_holders"
    CMD_GET_CREATORS       = "place/get_creators"
    CMD_GET_MEMBERS        = "place/get_members"
    CMD_GET_SUBPLACES      = "place/get_sub_places"
    CMD_GET_MUTUAL_PLACES  = "place/get_mutual_places"
    CMD_GET_NOTIFICATION   = "place/get_notification"
    CMD_GET_ACTIVITIES     = "place/get_activities"
    CMD_LEAVE              = "place/leave"
    CMD_MARK_ALL_READ      = "place/mark_all_read"
    CMD_REMOVE             = "place/remove"
    CMD_REMOVE_MEMBER      = "place/remove_member"
    CMD_REMOVE_PICTURE     = "place/remove_picture"
    CMD_REMOVE_FAVORITE    = "place/remove_favorite"
    CMD_SET_PICTURE        = "place/set_picture"
    CMD_SET_NOTIFICATION   = "place/set_notification"
    CMD_PROMOTE_MEMBER     = "place/promote_member"
    CMD_PIN_POST           = "place/pin_post"
    CMD_UNPIN_POST         = "place/unpin_post"
    CMD_DEMOTE_MEMBER      = "place/demote_member"
    CMD_INVITE_MEMBER      = "place/invite_member"
    CMD_UPDATE             = "place/update"
)

type PlaceService struct {
    worker          *api.Worker
    serviceCommands api.ServiceCommands
}

func NewPlaceService(worker *api.Worker) *PlaceService {
    s := new(PlaceService)
    s.worker = worker

    s.serviceCommands = api.ServiceCommands{
        CMD_AVAILABLE:          {api.AUTH_LEVEL_UNAUTHORIZED, s.placeIDAvailable},
        CMD_ADD_LOCKED_PLACE:   {api.AUTH_LEVEL_USER, s.createLockedPlace},
        CMD_ADD_UNLOCKED_PLACE: {api.AUTH_LEVEL_USER, s.createUnlockedPlace},
        CMD_ADD_GRAND_PLACE:    {api.AUTH_LEVEL_USER, s.createGrandPlace},
        CMD_ADD_FAVORITE:       {api.AUTH_LEVEL_USER, s.setPlaceAsFavorite},
        CMD_ADD_MEMBER:         {api.AUTH_LEVEL_USER, s.addPlaceMember},
        CMD_COUNT_UNREAD_POSTS: {api.AUTH_LEVEL_USER, s.countPlaceUnreadPosts},
        CMD_DEMOTE_MEMBER:      {api.AUTH_LEVEL_USER, s.demoteMember},
        CMD_GET:                {api.AUTH_LEVEL_APP_L3, s.getPlaceInfo},
        CMD_GET_MANY:           {api.AUTH_LEVEL_APP_L3, s.getManyPlacesInfo},
        CMD_GET_ACCESS:         {api.AUTH_LEVEL_APP_L3, s.getPlaceAccess},
        CMD_GET_FILES:          {api.AUTH_LEVEL_APP_L3, s.getPlaceFiles},
        CMD_GET_CREATORS:       {api.AUTH_LEVEL_USER, s.getPlaceCreators},
        CMD_GET_KEYHOLDERS:     {api.AUTH_LEVEL_USER, s.getPlaceKeyholders},
        CMD_GET_MEMBERS:        {api.AUTH_LEVEL_USER, s.getPlaceMembers},
        CMD_GET_SUBPLACES:      {api.AUTH_LEVEL_APP_L3, s.getSubPlaces},
        CMD_GET_POSTS:          {api.AUTH_LEVEL_APP_L3, s.getPlacePosts},
        CMD_GET_ACTIVITIES:     {api.AUTH_LEVEL_APP_L3, s.getPlaceActivities},
        CMD_GET_UNREAD_POSTS:   {api.AUTH_LEVEL_APP_L3, s.getPlaceUnreadPosts},
        CMD_GET_MUTUAL_PLACES:  {api.AUTH_LEVEL_APP_L3, s.getMutualPlaces},
        CMD_GET_NOTIFICATION:   {api.AUTH_LEVEL_USER, s.getPlaceNotification},
        CMD_INVITE_MEMBER:      {api.AUTH_LEVEL_USER, s.invitePlaceMember},
        CMD_MARK_ALL_READ:      {api.AUTH_LEVEL_APP_L3, s.markAllPostsAsRead},
        CMD_PROMOTE_MEMBER:     {api.AUTH_LEVEL_USER, s.promoteMember},
        CMD_REMOVE:             {api.AUTH_LEVEL_USER, s.remove},
        CMD_REMOVE_MEMBER:      {api.AUTH_LEVEL_USER, s.removeMember},
        CMD_LEAVE:              {api.AUTH_LEVEL_USER, s.leavePlace},
        CMD_REMOVE_FAVORITE:    {api.AUTH_LEVEL_USER, s.removePlaceFromFavorites},
        CMD_REMOVE_PICTURE:     {api.AUTH_LEVEL_USER, s.removePicture},
        CMD_SET_PICTURE:        {api.AUTH_LEVEL_APP_L3, s.setPicture},
        CMD_SET_NOTIFICATION:   {api.AUTH_LEVEL_APP_L3, s.setPlaceNotification},
        CMD_UPDATE:             {api.AUTH_LEVEL_USER, s.update},
        CMD_PIN_POST:           {api.AUTH_LEVEL_APP_L3, s.pinPost},
        CMD_UNPIN_POST:         {api.AUTH_LEVEL_APP_L3, s.unpinPost},
    }

    return s
}

func (s *PlaceService) GetServicePrefix() string {
    return SERVICE_PREFIX
}

func (s *PlaceService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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
