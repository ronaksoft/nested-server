package nestedServicePlace

import (
    "regexp"
    "strings"

    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server/server-gateway/client"
)

// @Command: place/add_member
// @Input:	place_id		string	*
// @Input:	member_id		string	*	(comma separated)
func (s *PlaceService) addPlaceMember(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    var memberIDs []string
    if placeID, ok := request.Data["place_id"].(string); ok {
        place = s.Worker().Model().Place.GetByID(placeID, nil)
        if place == nil {
            response.Error(nested.ERR_UNAVAILABLE, []string{"place_id"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"place_id"})
        return
    }
    if v, ok := request.Data["member_id"].(string); ok {
        memberIDs = strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"member_id"})
        return
    }
    // for grand places use invite
    if place.IsGrandPlace() {
        response.Error(nested.ERR_INVALID, []string{"cmd"})
        return
    }
    // check users right access
    access := place.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_ADD_MEMBERS] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    grandPlace := place.GetGrandParent()
    var invalidIDs []string
    for _, m := range memberIDs {
        if grandPlace.IsMember(m) && !place.IsMember(m) {
            if !place.HasKeyholderLimit() {
                s.Worker().Model().Place.AddKeyholder(place.ID, m)

                // Enables notification by default
                s.Worker().Model().Account.SetPlaceNotification(m, place.ID, true)

                // Add the place to the added user's feed list
                s.Worker().Model().Account.AddPlaceToBookmarks(m, place.ID)

                // Handle push notifications and activities
                s.Worker().Pusher().PlaceJoined(place, requester.ID, m)

                place.Counter.Keyholders += 1
            }
        } else {
            invalidIDs = append(invalidIDs, m)
        }
    }
    response.OkWithData(nested.M{"invalid_ids": invalidIDs})
}

// @Command: place/count_unread_posts
// @Input:	place_id		string	*
// @Input:	subs			bool	+
func (s *PlaceService) countPlaceUnreadPosts(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var placeIDs []string
    var withSubPlaces bool
    if v, ok := request.Data["place_id"].(string); ok {
        placeIDs = strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
    } else {
        response.Error(nested.ERR_INVALID, []string{"place_id"})
        return
    }
    if v, ok := request.Data["subs"].(bool); ok {
        withSubPlaces = v
    }
    r := make([]nested.M, 0)
    if withSubPlaces {
        places := s.Worker().Model().Place.GetPlacesByIDs(placeIDs)
        for _, place := range places {
            if !place.HasReadAccess(requester.ID) {
                continue
            }
            subPlaceIDs := []string{place.ID}
            for _, myPlaceID := range requester.AccessPlaceIDs {
                if s.Worker().Model().Place.IsSubPlace(place.ID, myPlaceID) {
                    subPlaceIDs = append(subPlaceIDs, myPlaceID)
                }
            }
            r = append(r, nested.M{
                "place_id": place.ID,
                "count":    s.Worker().Model().Place.CountUnreadPosts(subPlaceIDs, requester.ID),
            })
        }
    } else {
        places := s.Worker().Model().Place.GetPlacesByIDs(placeIDs)
        for _, place := range places {
            if !place.HasReadAccess(requester.ID) {
                continue
            }
            r = append(r, nested.M{
                "place_id": place.ID,
                "count":    s.Worker().Model().Place.CountUnreadPosts([]string{place.ID}, requester.ID),
            })
        }
    }
    response.OkWithData(nested.M{"counts": r})
    return
}

// @Command:	place/add_grand_place
// @Input:	place_id				string	*
// @Input:	place_name			string	*
// @Input:	place_description	string	+
// @Input:	privacy.receptive	string	*	(external | off)
// @Input:	privacy.search		bool		*
// @Input:	policy.add_member	string	*	(creators | everyone)
// @Input:	policy.add_post		string	*	(creators | everyone)
// @Input:	policy.add_place		string	*	(creators | everyone)
func (s *PlaceService) createGrandPlace(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    pcr := nested.PlaceCreateRequest{}
    if requester.Limits.GrandPlaces == 0 {
        response.Error(nested.ERR_LIMIT, []string{"no_grand_places"})
        return
    }
    if v, ok := request.Data["place_id"].(string); ok {
        pcr.ID = strings.ToLower(v)
        if pcr.ID == "" || len(pcr.ID) > nested.DEFAULT_MAX_PLACE_ID {
            response.Error(nested.ERR_INVALID, []string{"place_id"})
            return
        }
        if matched, err := regexp.MatchString(nested.DEFAULT_REGEX_GRANDPLACE_ID, pcr.ID); err != nil {
            response.Error(nested.ERR_UNKNOWN, []string{err.Error()})
            return
        } else if !matched {
            response.Error(nested.ERR_INVALID, []string{"place_id"})
            return
        }
        if !s.Worker().Model().Place.Available(pcr.ID) {
            response.Error(nested.ERR_DUPLICATE, []string{"place_id"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"place_id"})
        return
    }
    if v, ok := request.Data["place_name"].(string); ok {
        pcr.Name = v
        if pcr.Name == "" || len(pcr.Name) > nested.DEFAULT_MAX_PLACE_NAME {
            response.Error(nested.ERR_INVALID, []string{"place_name"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"place_name"})
        return
    }
    if v, ok := request.Data["place_description"].(string); ok {
        pcr.Description = v
    }

    // Privacy
    if v, ok := request.Data["privacy.receptive"].(string); ok {
        pcr.Privacy.Receptive = nested.PrivacyReceptive(v)
        switch pcr.Privacy.Receptive {
        case nested.PLACE_RECEPTIVE_EXTERNAL, nested.PLACE_RECEPTIVE_OFF:
        default:
            pcr.Privacy.Receptive = nested.PLACE_RECEPTIVE_OFF
        }
    } else {
        pcr.Privacy.Receptive = nested.PLACE_RECEPTIVE_OFF
    }
    if v, ok := request.Data["privacy.search"].(bool); ok {
        pcr.Privacy.Search = v
        if v {
            s.Worker().Model().Search.AddPlaceToSearchIndex(pcr.ID, pcr.Name)
        }
    }

    // Policy
    if v, ok := request.Data["policy.add_member"].(string); ok {
        pcr.Policy.AddMember = nested.PolicyGroup(v)
        switch pcr.Policy.AddMember {
        case nested.PLACE_POLICY_CREATORS, nested.PLACE_POLICY_EVERYONE:
        default:
            pcr.Policy.AddMember = nested.PLACE_POLICY_CREATORS
        }
    } else {
        pcr.Policy.AddMember = nested.PLACE_POLICY_CREATORS
    }
    if v, ok := request.Data["policy.add_post"].(string); ok {
        pcr.Policy.AddPost = nested.PolicyGroup(v)
        switch pcr.Policy.AddPost {
        case nested.PLACE_POLICY_CREATORS, nested.PLACE_POLICY_EVERYONE:
        default:
            pcr.Policy.AddPost = nested.PLACE_POLICY_CREATORS
        }
    } else {
        pcr.Policy.AddPost = nested.PLACE_POLICY_CREATORS
    }
    if v, ok := request.Data["policy.add_place"].(string); ok {
        pcr.Policy.AddPlace = nested.PolicyGroup(v)
        switch pcr.Policy.AddPlace {
        case nested.PLACE_POLICY_CREATORS, nested.PLACE_POLICY_EVERYONE:
        default:
            pcr.Policy.AddPlace = nested.PLACE_POLICY_CREATORS
        }
    } else {
        pcr.Policy.AddPlace = nested.PLACE_POLICY_CREATORS
    }

    pcr.GrandParentID = pcr.ID
    pcr.AccountID = requester.ID
    place := s.Worker().Model().Place.CreateGrandPlace(pcr)
    if place == nil {
        response.Error(nested.ERR_UNKNOWN, []string{"cannot_create_place"})
        return
    }
    // Add the creator of the place
    s.Worker().Model().Place.AddKeyholder(pcr.ID, requester.ID)
    s.Worker().Model().Place.Promote(pcr.ID, requester.ID)
    s.Worker().Model().Account.SetLimit(requester.ID, "grand_places", requester.Limits.GrandPlaces-1)

    // Enable Notification by default
    s.Worker().Model().Account.SetPlaceNotification(requester.ID, place.ID, true)

    // Add the place to feed
    s.Worker().Model().Account.AddPlaceToBookmarks(requester.ID, place.ID)

    response.OkWithData(nested.M{
        "_id":             place.ID,
        "name":            place.Name,
        "description":     place.Description,
        "picture":         place.Picture,
        "grand_parent_id": place.GrandParentID,
        "privacy":         place.Privacy,
        "policy":          place.Policy,
        "member_type":     nested.MEMBER_TYPE_CREATOR,
        "limits":          place.Limit,
        "counters":        place.Counter,
        "unread_posts":    s.Worker().Model().Place.CountUnreadPosts([]string{place.ID}, requester.ID),
    })
    return
}

// @Command:	place/add_locked_place
// @Input:	place_id				string	*
// @Input:	place_name			string	*
// @Input:	place_description	string	+
// @Input:	privacy.receptive	string	*	(external | internal | off)
// @Input:	privacy.search		bool		*
// @Input:	policy.add_member	string	*	(creators | everyone)
// @Input:	policy.add_post		string	*	(creators | everyone)
// @Input:	policy.add_place		string	*	(creators | everyone)
func (s *PlaceService) createLockedPlace(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    pcr := nested.PlaceCreateRequest{}

    if v, ok := request.Data["place_id"].(string); ok {
        pcr.ID = strings.ToLower(v)
        if pcr.ID == "" || len(pcr.ID) > nested.DEFAULT_MAX_PLACE_ID {
            response.Error(nested.ERR_INVALID, []string{"place_id"})
            return
        }
        // check if place is a subplace
        if pos := strings.LastIndex(pcr.ID, "."); pos == -1 {
            response.Error(nested.ERR_INVALID, []string{"place_id"})
            return
        } else {
            localPlaceID := string(pcr.ID[pos+1:])
            // check if place id is a valid place id
            if matched, err := regexp.MatchString(nested.DEFAULT_REGEX_PLACE_ID, localPlaceID); err != nil {
                response.Error(nested.ERR_UNKNOWN, []string{err.Error()})
                return
            } else if !matched {
                response.Error(nested.ERR_INVALID, []string{"place_id"})
                return
            }
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"place_id"})
        return
    }
    if v, ok := request.Data["place_name"].(string); ok {
        pcr.Name = v
        if pcr.Name == "" || len(pcr.Name) > nested.DEFAULT_MAX_PLACE_NAME {
            response.Error(nested.ERR_INVALID, []string{"place_name"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"place_name"})
        return
    }
    if v, ok := request.Data["place_description"].(string); ok {
        pcr.Description = v
    }

    // Privacy validation checks
    if v, ok := request.Data["privacy.receptive"].(string); ok {
        pcr.Privacy.Receptive = nested.PrivacyReceptive(v)
        switch pcr.Privacy.Receptive {
        case nested.PLACE_RECEPTIVE_EXTERNAL, nested.PLACE_RECEPTIVE_INTERNAL, nested.PLACE_RECEPTIVE_OFF:
        default:
            pcr.Privacy.Receptive = nested.PLACE_RECEPTIVE_OFF
        }
    } else {
        pcr.Privacy.Receptive = nested.PLACE_RECEPTIVE_OFF
    }
    if v, ok := request.Data["privacy.search"].(bool); ok {
        pcr.Privacy.Search = v
        if v {
            s.Worker().Model().Search.AddPlaceToSearchIndex(pcr.ID, pcr.Name)
        }
    }

    // Policy validation checks
    if v, ok := request.Data["policy.add_member"].(string); ok {
        pcr.Policy.AddMember = nested.PolicyGroup(v)
        switch pcr.Policy.AddMember {
        case nested.PLACE_POLICY_CREATORS, nested.PLACE_POLICY_EVERYONE:
        default:
            pcr.Policy.AddMember = nested.PLACE_POLICY_CREATORS
        }
    } else {
        pcr.Policy.AddMember = nested.PLACE_POLICY_CREATORS
    }
    if v, ok := request.Data["policy.add_post"].(string); ok {
        pcr.Policy.AddPost = nested.PolicyGroup(v)
        switch pcr.Policy.AddPost {
        case nested.PLACE_POLICY_CREATORS, nested.PLACE_POLICY_EVERYONE:
        default:
            pcr.Policy.AddPost = nested.PLACE_POLICY_CREATORS
        }
    } else {
        pcr.Policy.AddPost = nested.PLACE_POLICY_CREATORS
    }
    if v, ok := request.Data["policy.add_place"].(string); ok {
        pcr.Policy.AddPlace = nested.PolicyGroup(v)
        switch pcr.Policy.AddPlace {
        case nested.PLACE_POLICY_CREATORS, nested.PLACE_POLICY_EVERYONE:
        default:
            pcr.Policy.AddPlace = nested.PLACE_POLICY_CREATORS
        }
    } else {
        pcr.Policy.AddPlace = nested.PLACE_POLICY_CREATORS
    }

    // check parent's limitations and access permissions
    parent := s.Worker().Model().Place.GetByID(s.Worker().Model().Place.GetParentID(pcr.ID), nil)
    if parent == nil {
        response.Error(nested.ERR_INVALID, []string{"place_id"})
        return
    }
    if parent.Level >= nested.DEFAULT_PLACE_MAX_LEVEL {
        response.Error(nested.ERR_LIMIT, []string{"level"})
        return
    }
    if parent.HasChildLimit() {
        response.Error(nested.ERR_LIMIT, []string{"place"})
        return
    }

    // check if user has the right to create place
    access := parent.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_ADD_PLACE] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    pcr.GrandParentID = parent.GrandParentID
    pcr.AccountID = requester.ID
    grandPlace := s.Worker().Model().Place.GetByID(parent.GrandParentID, nil)
    var place *nested.Place
    if grandPlace.IsPersonal() {
        place = s.Worker().Model().Place.CreatePersonalPlace(pcr)
        if place == nil {
            response.Error(nested.ERR_UNKNOWN, []string{})
            return
        }
    } else {
        place = s.Worker().Model().Place.CreateLockedPlace(pcr)
        if place == nil {
            response.Error(nested.ERR_UNKNOWN, []string{})
            return
        }
    }
    // Add the creator of the place
    s.Worker().Model().Place.AddKeyholder(pcr.ID, requester.ID)
    s.Worker().Model().Place.Promote(pcr.ID, requester.ID)

    // Enable Notification by default
    s.Worker().Model().Account.SetPlaceNotification(requester.ID, place.ID, true)

    // Add place to the user's feed
    s.Worker().Model().Account.AddPlaceToBookmarks(requester.ID, place.ID)

    response.OkWithData(nested.M{
        "_id":             place.ID,
        "name":            place.Name,
        "description":     place.Description,
        "picture":         place.Picture,
        "grand_parent_id": place.GrandParentID,
        "privacy":         place.Privacy,
        "policy":          place.Policy,
        "member_type":     nested.MEMBER_TYPE_CREATOR,
        "limits":          place.Limit,
        "counters":        place.Counter,
        "unread_posts":    s.Worker().Model().Place.CountUnreadPosts([]string{place.ID}, requester.ID),
    })
    return
}

// @Command:	place/add_unlocked_place
// @Input:	place_id				string	*
// @Input:	place_name			string	*
// @Input:	place_description	string	+
func (s *PlaceService) createUnlockedPlace(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    pcr := nested.PlaceCreateRequest{}
    if v, ok := request.Data["place_id"].(string); ok {
        pcr.ID = strings.ToLower(v)
        if pcr.ID == "" || len(pcr.ID) > nested.DEFAULT_MAX_PLACE_ID {
            response.Error(nested.ERR_INVALID, []string{"place_id"})
            return
        }
        // check if place is a subplace
        if pos := strings.LastIndex(pcr.ID, "."); pos == -1 {
            response.Error(nested.ERR_INVALID, []string{"place_id"})
            return
        } else {
            localPlaceID := string(pcr.ID[pos+1:])
            // check if place id is a valid place id
            if matched, err := regexp.MatchString(nested.DEFAULT_REGEX_PLACE_ID, localPlaceID); err != nil {
                response.Error(nested.ERR_UNKNOWN, []string{err.Error()})
                return
            } else if !matched {
                response.Error(nested.ERR_INVALID, []string{"place_id"})
                return
            }
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"place_id"})
        return
    }
    if v, ok := request.Data["place_name"].(string); ok {
        pcr.Name = v
        if pcr.Name == "" || len(pcr.Name) > nested.DEFAULT_MAX_PLACE_NAME {
            response.Error(nested.ERR_INVALID, []string{"place_name"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"place_name"})
        return
    }
    if v, ok := request.Data["place_description"].(string); ok {
        pcr.Description = v
    }

    // check parent's limitations and access permissions
    parent := s.Worker().Model().Place.GetByID(s.Worker().Model().Place.GetParentID(pcr.ID), nil)
    if parent == nil {
        response.Error(nested.ERR_INVALID, []string{"place_id"})
        return
    }
    if parent.HasChildLimit() {
        response.Error(nested.ERR_LIMIT, []string{"place"})
        return
    }
    if !parent.IsGrandPlace() {
        response.Error(nested.ERR_ACCESS, []string{"open_places_only_on_level_1"})
        return
    }

    // check if user has the right to create place
    access := parent.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_ADD_PLACE] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    pcr.GrandParentID = parent.GrandParentID
    pcr.AccountID = requester.ID
    grandPlace := s.Worker().Model().Place.GetByID(parent.GrandParentID, nil)
    var place *nested.Place
    if grandPlace.IsPersonal() {
        response.Error(nested.ERR_ACCESS, []string{"no_open_place_in_personal"})
        return
    }
    place = s.Worker().Model().Place.CreateUnlockedPlace(pcr)
    if place == nil {
        response.Error(nested.ERR_UNKNOWN, []string{})
        return
    }

    // Add the creator of the place
    s.Worker().Model().Place.AddKeyholder(pcr.ID, requester.ID)
    s.Worker().Model().Place.Promote(pcr.ID, requester.ID)

    // Enable Notification by default
    s.Worker().Model().Account.SetPlaceNotification(requester.ID, place.ID, true)

    // Add place to the user's feed
    s.Worker().Model().Account.AddPlaceToBookmarks(requester.ID, place.ID)

    response.OkWithData(nested.M{
        "_id":             place.ID,
        "name":            place.Name,
        "description":     place.Description,
        "picture":         place.Picture,
        "grand_parent_id": place.GrandParentID,
        "privacy":         place.Privacy,
        "policy":          place.Policy,
        "member_type":     nested.MEMBER_TYPE_CREATOR,
        "limits":          place.Limit,
        "counters":        place.Counter,
        "unread_posts":    s.Worker().Model().Place.CountUnreadPosts([]string{place.ID}, requester.ID),
    })

}

// @Command:	place/demote_member
// @Input:	place_id				string	*
// @Input:	member_id			string	*
func (s *PlaceService) demoteMember(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var memberID string
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    if v, ok := request.Data["member_id"].(string); ok {
        memberID = v
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"member_id"})
        return
    }
    if !place.IsCreator(requester.ID) {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    if !place.IsCreator(memberID) {
        response.Error(nested.ERR_INVALID, []string{"member_id"})
        return
    }
    if place.HasKeyholderLimit() {
        response.Error(nested.ERR_LIMIT, []string{"member_id"})
        return
    }

    s.Worker().Model().Place.Demote(place.ID, memberID)

    s.Worker().Pusher().PlaceMemberDemoted(place, requester.ID, memberID)

    response.Ok()

}

// @Command:	place/get_access
// @Input:	place_id				string	*	(comma separated)
func (s *PlaceService) getPlaceAccess(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var places []nested.Place
    if v, ok := request.Data["place_id"].(string); ok {
        placeIDs := strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
        places = s.Worker().Model().Place.GetPlacesByIDs(placeIDs)
    } else {
        if v, ok := request.Data["place_ids"].(string); ok {
            placeIDs := strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
            places = s.Worker().Model().Place.GetPlacesByIDs(placeIDs)
        } else {
            response.Error(nested.ERR_INVALID, []string{"place_id"})
            return
        }
    }

    var r []nested.M
    for _, place := range places {
        access := place.GetAccess(requester.ID)
        a := make([]string, 0, 10)
        a = a[:0]
        for k, v := range access {
            if v {
                a = append(a, k)
            }
        }
        r = append(r, nested.M{
            "_id":      place.ID,
            "place_id": place.ID,
            "access":   a,
        })
    }
    response.OkWithData(nested.M{"places": r})
}

// @Command:	place/get
// @Input:	place_id				string	*
func (s *PlaceService) getPlaceInfo(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    response.OkWithData(s.Worker().Map().Place(requester, *place, place.GetAccess(requester.ID)))
}

// @Command:	place/get_many
// @Input:	place_id				string	*	(comma separated)
// @Input:	member_id			string	*
func (s *PlaceService) getManyPlacesInfo(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var places []nested.Place
    if v, ok := request.Data["place_id"].(string); ok {
        placeIDs := strings.Split(v, ",")
        places = s.Worker().Model().Place.GetPlacesByIDs(placeIDs)
        if len(places) == 0 {
            response.OkWithData(nested.M{"places": []nested.M{}})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"place_id"})
        return
    }
    r := make([]nested.M, 0, len(places))
    for _, place := range places {
        r = append(r, s.Worker().Map().Place(requester, place, place.GetAccess(requester.ID)))
    }
    response.OkWithData(nested.M{"places": r})
}

// @Command:	place/get_activities
// @Input:	place_id				string	*
// @Input:	details				bool		+
func (s *PlaceService) getPlaceActivities(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    var details bool
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }

    if v, ok := request.Data["details"].(bool); ok {
        details = v
    }

    if !place.HasReadAccess(requester.ID) {
        response.Error(nested.ERR_ACCESS, []string{""})
        return
    }

    pg := s.Worker().Argument().GetPagination(request)
    ta := s.Worker().Model().PlaceActivity.GetActivitiesByPlace(place.ID, pg)
    d := make([]nested.M, 0, pg.GetLimit())
    for _, v := range ta {
        d = append(d, s.Worker().Map().PlaceActivity(requester, v, details))
    }
    response.OkWithData(nested.M{
        "skip":       pg.GetSkip(),
        "limit":      pg.GetLimit(),
        "activities": d,
    })
}

// @Command:	place/get_files
// @Input:	place_id		string	*
// @Input:	filter		string	+ AUD | DOC | IMG | VID | OTH | all
// @Input:	filename		string	+
func (s *PlaceService) getPlaceFiles(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var filter, filename string
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    if ! place.HasReadAccess(requester.ID) {
        response.Error(nested.ERR_ACCESS, []string{""})
        return
    }
    if v, ok := request.Data["filter"].(string); ok {
        switch v {
        case nested.FILE_TYPE_AUDIO, nested.FILE_TYPE_DOCUMENT, nested.FILE_TYPE_IMAGE, nested.FILE_TYPE_VIDEO, nested.FILE_TYPE_OTHER:
            filter = v
        default:
            filter = nested.FILE_TYPE_ALL
        }

    }
    if v, ok := request.Data["filename"].(string); ok {
        filename = v
    }
    access := place.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_READ_POST] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    pg := s.Worker().Argument().GetPagination(request)
    result := s.Worker().Model().File.GetFilesByPlace(place.ID, filter, filename, pg)
    r := make([]nested.M, 0, len(result))
    for _, f := range result {
            d := s.Worker().Map().FileInfo(f.File)
            d["post_id"] = f.PostId.Hex()
            r = append(r, d)
    }
    response.OkWithData(nested.M{"files": r})
}

// @Command:	place/get_unread_posts
// @Input:	place_id		string	*
// @Input:	subs			bool		+	(default: FALSE)
func (s *PlaceService) getPlaceUnreadPosts(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var subPlaces bool
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }

    if v, ok := request.Data["subs"].(bool); ok {
        subPlaces = v
    }

    if !place.HasReadAccess(requester.ID) {
        response.Error(nested.ERR_UNAVAILABLE, []string{"place_id"})
        return
    }

    pg := s.Worker().Argument().GetPagination(request)
    posts := s.Worker().Model().Post.GetUnreadPostsByPlace(place.ID, requester.ID, subPlaces, pg)
    r := make([]nested.M, 0, len(posts))
    for _, post := range posts {
        r = append(r, s.Worker().Map().Post(requester, post, true))
    }
    response.OkWithData(nested.M{
        "skip":  pg.GetSkip(),
        "limit": pg.GetLimit(),
        "posts": r,
    })
    return
}

// @Command:	place/get_notification
// @Input:	place_id		string	*
func (s *PlaceService) getPlaceNotification(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    if s.Worker().Model().Group.ItemExists(place.Groups["_ntfy"], requester.ID) {
        response.OkWithData(nested.M{"state": true})
    } else {
        response.OkWithData(nested.M{"state": false})
    }
    return
}

// @Command:	place/get_creators
// @Input:	place_id		string	*
func (s *PlaceService) getPlaceCreators(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    access := place.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_SEE_MEMBERS] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    pg := s.Worker().Argument().GetPagination(request)
    iStart := pg.GetSkip()
    iEnd := iStart + pg.GetLimit()
    if iEnd > len(place.CreatorIDs) {
        iEnd = len(place.CreatorIDs)
    }

    var r []nested.M
    for _, v := range place.CreatorIDs[iStart:iEnd] {
        m := s.Worker().Model().Account.GetByID(v, nil)
        r = append(r, s.Worker().Map().Account(*m, false))
    }

    response.OkWithData(nested.M{
        "total":    place.Counter.Creators,
        "skip":     pg.GetSkip(),
        "limit":    pg.GetLimit(),
        "creators": r,
    })
}

// @Command:	place/get_key_holders
// @Input:	place_id		string	*
func (s *PlaceService) getPlaceKeyholders(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    access := place.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_SEE_MEMBERS] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    pg := s.Worker().Argument().GetPagination(request)
    iStart := pg.GetSkip()
    iEnd := iStart + pg.GetLimit()
    if iEnd > len(place.KeyholderIDs) {
        iEnd = len(place.KeyholderIDs)
    }

    var r []nested.M
    for _, v := range place.KeyholderIDs[iStart:iEnd] {
        m := s.Worker().Model().Account.GetByID(v, nil)
        r = append(r, s.Worker().Map().Account(*m, false))
    }
    response.OkWithData(nested.M{
        "total":       place.Counter.Keyholders,
        "skip":        pg.GetSkip(),
        "limit":       pg.GetLimit(),
        "key_holders": r,
    })
}

// @Command:	place/get_members
// @Input:	place_id		string	*
func (s *PlaceService) getPlaceMembers(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    access := place.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_SEE_MEMBERS] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    // TODO:: use s.Worker().Model().Account.GetAccountsByIDs instead
    rKeyholders := make([]nested.M, 0, len(place.KeyholderIDs))
    for _, v := range place.KeyholderIDs {
        m := s.Worker().Model().Account.GetByID(v, nil)
        rKeyholders = append(rKeyholders, s.Worker().Map().Account(*m, false))
    }
    rCreators := make([]nested.M, 0, len(place.CreatorIDs))
    for _, v := range place.CreatorIDs {
        m := s.Worker().Model().Account.GetByID(v, nil)
        rCreators = append(rCreators, s.Worker().Map().Account(*m, false))
    }

    response.OkWithData(nested.M{
        "key_holders": rKeyholders,
        "creators":    rCreators,
    })
}

// @Command:	place/get_sub_places
// @Input:	place_id		string	*
func (s *PlaceService) getSubPlaces(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    mapPlaceIDs := nested.M{}
    if place.IsGrandPlace() {
        for _, placeID := range place.UnlockedChildrenIDs {
            mapPlaceIDs[placeID] = true
        }
        for _, placeID := range requester.AccessPlaceIDs {
            if dotIndex := strings.Index(placeID, "."); dotIndex == -1 {
                continue
            } else {
                grandParentID := string(placeID[:dotIndex])
                if grandParentID == place.ID {
                    mapPlaceIDs[placeID] = true
                }
            }
        }
    } else {
        for _, placeID := range requester.AccessPlaceIDs {
            if dotIndex := strings.LastIndex(placeID, "."); dotIndex == -1 {
                continue
            } else {
                parentID := string(placeID[:dotIndex])
                if parentID == place.ID {
                    mapPlaceIDs[placeID] = true
                }
            }
        }
    }
    placeIDs := mapPlaceIDs.KeysToArray()
    places := s.Worker().Model().Place.GetPlacesByIDs(placeIDs)
    var r []nested.M
    for _, place := range places {
        r = append(r, s.Worker().Map().Place(requester, place, place.GetAccess(requester.ID)))
    }
    response.OkWithData(nested.M{"places": r})
}

// @Command:	place/get_mutual_places
// @Input:	account_id		string	*
func (s *PlaceService) getMutualPlaces(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var accountID string
    if v, ok := request.Data["account_id"].(string); ok {
        accountID = v
        if !s.Worker().Model().Account.Exists(accountID) {
            response.Error(nested.ERR_INVALID, []string{"account_id"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"account_id"})
        return
    }
    placeIDs := s.Worker().Model().Account.GetMutualPlaceIDs(requester.ID, accountID)
    r := make([]nested.M, 0, len(placeIDs))
    iStart := 0
    iLength := nested.DEFAULT_MAX_RESULT_LIMIT
    iEnd := iStart + iLength
    if iEnd > len(placeIDs) {
        iEnd = len(placeIDs)
    }
    for {
        for _, place := range s.Worker().Model().Place.GetPlacesByIDs(placeIDs[iStart:iEnd]) {
            r = append(r, s.Worker().Map().Place(requester, place, place.GetAccess(requester.ID)))
        }
        iStart += iLength
        iEnd = iStart + iLength
        if iStart >= len(placeIDs) {
            break
        }
        if iEnd > len(placeIDs) {
            iEnd = len(placeIDs)
        }
    }
    response.OkWithData(nested.M{"places": r})
}

// @Command:	place/get_posts
// @Input:	place_id		string	*
// @Input:	by_update	bool		+
func (s *PlaceService) getPlacePosts(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var sort_item string
    var place *nested.Place
    var posts []nested.Post
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    sort_item = nested.POST_SORT_TIMESTAMP
    if v, ok := request.Data["by_update"].(bool); ok {
        if v {
            sort_item = nested.POST_SORT_LAST_UPDATE
        }
    }

    // user must have read access in place
    if !place.HasReadAccess(requester.ID) {
        response.Error(nested.ERR_ACCESS, []string{""})
        return
    }

    pg := s.Worker().Argument().GetPagination(request)
    if place.ID == requester.ID {
        posts = s.Worker().Model().Post.GetPostsOfPlaces([]string{place.ID, "*"}, sort_item, pg)
    } else {
        posts = s.Worker().Model().Post.GetPostsByPlace(place.ID, sort_item, pg)
    }
    r := make([]nested.M, 0, len(posts))

    for _, post := range posts {
        r = append(r, s.Worker().Map().Post(requester, post, true))
    }
    response.OkWithData(nested.M{
        "skip":  pg.GetSkip(),
        "limit": pg.GetLimit(),
        "posts": r,
    })
    return
}

// @Command:	place/invite_member
// @Input:	place_id			string	*
// @Input:	member_id		string	*	(comma separated)
func (s *PlaceService) invitePlaceMember(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var memberIDs []string
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    if v, ok := request.Data["member_id"].(string); ok {
        memberIDs = strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"member_id"})
        return
    }

    // user can only invite people to grand place otherwise they must be added
    if !place.IsGrandPlace() {
        response.Error(nested.ERR_INVALID, []string{"cmd"})
        return
    }

    // check if user has the right permission
    access := place.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_ADD_MEMBERS] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    invalidIDs := make([]string, 0)
    for _, m := range memberIDs {
        if !s.Worker().Model().Account.Exists(m) {
            invalidIDs = append(invalidIDs, m)
            continue
        }

        if place.IsMember(m) {
            invalidIDs = append(invalidIDs, m)
            continue
        }

        if !place.HasKeyholderLimit() {
            s.Worker().Model().Place.AddKeyholder(place.ID, m)

            // Enables notification by default
            s.Worker().Model().Account.SetPlaceNotification(m, place.ID, true)

            // Add the place to the added user's feed list
            s.Worker().Model().Account.AddPlaceToBookmarks(m, place.ID)

            place.Counter.Keyholders += 1

            s.Worker().Pusher().PlaceJoined(place, requester.ID, m)
        } else {
            invalidIDs = append(invalidIDs, m)
        }
    }
    response.OkWithData(nested.M{"invalid_ids": invalidIDs})
}

// @Command:	place/leave
// @Input:	place_id		string	*
func (s *PlaceService) leavePlace(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }

    if place.IsPersonal() {
        response.Error(nested.ERR_ACCESS, []string{"cannot_leave_personal_place"})
        return
    }

    // if user is leaving grand place then they must leave all the sub places first.
    // if user cannot leave any of the sub-places then they must fix the issue by removing the place
    // or adding another creator to the sub-place
    if place.IsGrandPlace() {
        for _, subPlaceID := range requester.AccessPlaceIDs {
            if s.Worker().Model().Place.IsSubPlace(place.ID, subPlaceID) {
                subPlace := s.Worker().Model().Place.GetByID(subPlaceID, nil)
                if subPlace != nil {
                    if subPlace.IsCreator(requester.ID) {
                        s.Worker().Model().Place.RemoveCreator(subPlaceID, requester.ID, requester.ID)
                    } else if subPlace.IsKeyholder(requester.ID) {
                        s.Worker().Model().Place.RemoveKeyHolder(subPlaceID, requester.ID, requester.ID)
                    }
                }
            }
        }
    }

    if place.IsCreator(requester.ID) {
        s.Worker().Model().Place.RemoveCreator(place.ID, requester.ID, requester.ID)
    } else if place.IsKeyholder(requester.ID) {
        s.Worker().Model().Place.RemoveKeyHolder(place.ID, requester.ID, requester.ID)
    } else {
        response.Error(nested.ERR_INVALID, []string{"you_are_not_member"})
        return
    }
    response.Ok()
    return
}

// @Command:	place/mark_all_read
// @Input:	place_id		string	*
func (s *PlaceService) markAllPostsAsRead(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    s.Worker().Model().Post.MarkAsReadByPlace(place.ID, requester.ID)
    response.Ok()
}

// @Command:	place/available
// @Input:	place_id		string	*
func (s *PlaceService) placeIDAvailable(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    if placeID, ok := request.Data["place_id"].(string); ok {
        if s.Worker().Model().Place.Available(strings.ToLower(placeID)) {
            response.Ok()
        } else {
            response.Error(nested.ERR_UNAVAILABLE, []string{"place_id"})
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"place_id"})
        return
    }
    return
}

// @Command:	place/promote_member
// @Input:	place_id		string	*
// @Input:	member_id	string	*
func (s *PlaceService) promoteMember(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var memberID string
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    if v, ok := request.Data["member_id"].(string); ok {
        memberID = v
        if requester.ID == memberID || !place.IsCreator(requester.ID) {
            response.Error(nested.ERR_ACCESS, []string{})
            return
        }
    }
    if !place.IsKeyholder(memberID) {
        response.Error(nested.ERR_INVALID, []string{"member_id"})
        return
    }
    if place.HasCreatorLimit() {
        response.Error(nested.ERR_LIMIT, []string{"creators"})
        return
    }
    s.Worker().Model().Place.Promote(place.ID, memberID)

    s.Worker().Pusher().PlaceMemberPromoted(place, requester.ID, memberID)

    response.Ok()
}

// @Command:	place/pin_post
// @Input:	place_id		string	*
// @Input: post_id          string *
func (s *PlaceService) pinPost(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    var post *nested.Post
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    if post = s.Worker().Argument().GetPost(request, response); post == nil {
        return
    }
    // Only creators of the place or system admins can do it
    if !place.IsCreator(requester.ID) && !requester.Authority.Admin {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    if !post.IsInPlace(place.ID) {
        response.Error(nested.ERR_UNAVAILABLE, []string{"post_id"})
        return
    }
    s.Worker().Model().Place.PinPost(place.ID, post.ID)
    response.Ok()
}

// @Command:	place/remove
// @Input:	place_id		string	*
func (s *PlaceService) remove(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }

    // check if user has the right permission
    access := place.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_REMOVE_PLACE] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    if place.Counter.Children > 0 {
        response.Error(nested.ERR_ACCESS, []string{"remove_children_first"})
        return
    }
    if s.Worker().Model().Place.Remove(place.ID, requester.ID) {
        s.Worker().Model().Account.IncrementLimit(place.MainCreatorID, "grand_places", 1)
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{})
    }

    // Remove Place from search index
    s.Worker().Model().Search.RemovePlaceFromSearchIndex(place.ID)

}

// @Command:	place/remove_favorite
// @Input:	place_id		string	*
func (s *PlaceService) removePlaceFromFavorites(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    access := place.GetAccess(requester.ID)
    if access[nested.PLACE_ACCESS_READ_POST] {
        s.Worker().Model().Account.RemovePlaceFromBookmarks(requester.ID, place.ID)
        response.Ok()
    } else {
        response.Error(nested.ERR_ACCESS, []string{})
    }
    return
}

// @Command:	place/remove_member
// @Input:	place_id		string	*
// @Input:	member_id	string	*
func (s *PlaceService) removeMember(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var memberID string
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    if v, ok := request.Data["member_id"].(string); ok {
        memberID = v
    }
    if memberID == requester.ID {
        response.Error(nested.ERR_ACCESS, []string{"cant remove yourself, leave instead"})
        return
    }
    access := place.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_REMOVE_MEMBERS] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }

    // If you are leaving a grand place first must check through all the sub-places and
    // check if user is allowed to leave that.
    if place.IsGrandPlace() {
        for _, pid := range s.Worker().Model().Account.GetAccessPlaceIDs(memberID) {
            if s.Worker().Model().Place.IsSubPlace(place.ID, pid) {
                subPlace := s.Worker().Model().Place.GetByID(pid, nil)
                if subPlace != nil {
                    switch {
                    case subPlace.IsCreator(memberID):
                        s.Worker().Model().Place.RemoveCreator(pid, memberID, requester.ID)
                    default:
                        s.Worker().Model().Place.RemoveKeyHolder(pid, memberID, requester.ID)
                    }
                }
            }
        }
    }
    // remove the user from placeID
    switch {
    case place.IsCreator(memberID):
        s.Worker().Model().Place.RemoveCreator(place.ID, memberID, requester.ID)
    case place.IsKeyholder(memberID):
        s.Worker().Model().Place.RemoveKeyHolder(place.ID, memberID, requester.ID)
    default:
        response.Error(nested.ERR_INVALID, []string{"member_id"})
    }
    response.Ok()
    return
}

// @Command:	place/remove_picture
// @Input:	place_id		string	*
func (s *PlaceService) removePicture(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    access := place.GetAccess(requester.ID)
    if access[nested.PLACE_ACCESS_CONTROL] {
        s.Worker().Model().Place.SetPicture(place.ID, nested.Picture{})
        response.Ok()
    } else {
        response.Error(nested.ERR_ACCESS, []string{})
    }
    return
}

// @Command:	place/add_favorite
// @Input:	place_id		string	*
func (s *PlaceService) setPlaceAsFavorite(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    access := place.GetAccess(requester.ID)
    if access[nested.PLACE_ACCESS_READ_POST] {
        s.Worker().Model().Account.AddPlaceToBookmarks(requester.ID, place.ID)
        response.Ok()
    } else {
        response.Error(nested.ERR_ACCESS, []string{})
    }
    return
}

// @Command:	place/set_notification
// @Input:	place_id		string	*
// @Input:	state		bool		*
func (s *PlaceService) setPlaceNotification(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var state bool
    var place *nested.Place
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    if v, ok := request.Data["state"].(bool); ok {
        state = v
    }
    // user must have READ access in the place
    access := place.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_READ_POST] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    if state {
        s.Worker().Model().Group.AddItems(place.Groups["_ntfy"], []string{requester.ID})
    } else {
        s.Worker().Model().Group.RemoveItems(place.Groups["_ntfy"], []string{requester.ID})
    }
    response.Ok()
}

// @Command:	place/set_picture
// @Input:	place_id			string	*
// @Input:	universal_id		string	*
func (s *PlaceService) setPicture(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var uniID nested.UniversalID
    var place *nested.Place
    pic := nested.Picture{}
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    if v, ok := request.Data["universal_id"].(string); ok {
        uniID = nested.UniversalID(v)
        fileInfo := s.Worker().Model().File.GetByID(uniID, nil)
        if fileInfo == nil {
            response.Error(nested.ERR_UNAVAILABLE, []string{"universal_id"})
            return
        }
        pic = fileInfo.Thumbnails
    }
    access := place.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_CONTROL] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    s.Worker().Model().Place.SetPicture(place.ID, pic)
    response.Ok()
    return
}

// @Command:	place/update
// @Input:	place_id					string	*
// @Input:	place_name				string	+
// @Input:	place_desc				string	+
// @Input:	privacy.search			bool		+
// @Input:	privacy.receptive		string	+	(external | internal | off)
// @Input:	policy.add_post			string	+	(creators | everyone)
// @Input:	policy.add_member		string	+	(creators | everyone)
// @Input:	policy.add_place			string	+	(creators | everyone)
func (s *PlaceService) update(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    placeUpdateRequest := nested.M{}
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    if v, ok := request.Data["place_name"].(string); ok {
        placeName := strings.TrimSpace(v)
        if len(placeName) > 0 {
            placeUpdateRequest["name"] = placeName
        }
    }
    if v, ok := request.Data["place_desc"].(string); ok && v != "" {
        placeUpdateRequest["description"] = v
    }
    if v, ok := request.Data["policy.add_member"].(string); ok {
        if !place.IsPersonal() {
            switch nested.PolicyGroup(v) {
            case nested.PLACE_POLICY_CREATORS, nested.PLACE_POLICY_EVERYONE:
                placeUpdateRequest["policy.add_member"] = v
            }
        }

    }
    if place.Privacy.Locked == true {
        if v, ok := request.Data["privacy.search"].(bool); ok {
            placeUpdateRequest["privacy.search"] = v
            if v {
                s.Worker().Model().Search.AddPlaceToSearchIndex(requester.ID, place.Name)
            } else {
                s.Worker().Model().Search.RemovePlaceFromSearchIndex(place.ID)
            }
        }
        if v, ok := request.Data["privacy.receptive"].(string); ok {
            if place.IsGrandPlace() {
                switch nested.PrivacyReceptive(v) {
                case nested.PLACE_RECEPTIVE_EXTERNAL:
                    placeUpdateRequest["privacy.receptive"] = v
                case nested.PLACE_RECEPTIVE_INTERNAL, nested.PLACE_RECEPTIVE_OFF:
                    placeUpdateRequest["privacy.receptive"] = v
                    s.Worker().Model().Search.RemovePlaceFromSearchIndex(requester.ID)
                default:
                    response.Error(nested.ERR_INVALID, []string{"privacy.receptive"})
                    return
                }
            } else {
                switch nested.PrivacyReceptive(v) {
                case nested.PLACE_RECEPTIVE_EXTERNAL, nested.PLACE_RECEPTIVE_INTERNAL:
                    placeUpdateRequest["privacy.receptive"] = v
                case nested.PLACE_RECEPTIVE_OFF:
                    placeUpdateRequest["privacy.receptive"] = v
                    placeUpdateRequest["privacy.search"] = false
                    s.Worker().Model().Search.RemovePlaceFromSearchIndex(place.Name)
                default:
                    response.Error(nested.ERR_INVALID, []string{"privacy.receptive"})
                    return
                }
            }

        }
        if v, ok := request.Data["policy.add_post"].(string); ok {
            switch nested.PolicyGroup(v) {
            case nested.PLACE_POLICY_CREATORS, nested.PLACE_POLICY_EVERYONE:
                placeUpdateRequest["policy.add_post"] = v

            }
        }
        if v, ok := request.Data["policy.add_place"].(string); ok {
            if !place.IsPersonal() {
                switch nested.PolicyGroup(v) {
                case nested.PLACE_POLICY_CREATORS, nested.PLACE_POLICY_EVERYONE:
                    placeUpdateRequest["policy.add_place"] = v

                }
            }
        }
    }
    access := place.GetAccess(requester.ID)
    if !access[nested.PLACE_ACCESS_CONTROL] {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    if len(placeUpdateRequest) > 0 {
        s.Worker().Model().Place.Update(place.ID, placeUpdateRequest)
        s.Worker().Pusher().PlaceSettingsUpdated(place, requester.ID)
    }
    response.OkWithData(nested.M{"applied": placeUpdateRequest})
    return
}

// @Command:	place/unpin_post
// @Input:	place_id		string	*
// @Input: post_id          string *
func (s *PlaceService) unpinPost(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var place *nested.Place
    var post *nested.Post
    if place = s.Worker().Argument().GetPlace(request, response); place == nil {
        return
    }
    if post = s.Worker().Argument().GetPost(request, response); post == nil {
        return
    }
    // Only creators of the place or system admins can do it
    if !place.IsCreator(requester.ID) && !requester.Authority.Admin {
        response.Error(nested.ERR_ACCESS, []string{})
        return
    }
    if !post.IsInPlace(place.ID) {
        response.Error(nested.ERR_UNAVAILABLE, []string{"post_id"})
        return
    }
    s.Worker().Model().Place.UnpinPost(place.ID, post.ID)
    response.Ok()
}

// @Command:	place/get_blocked_ids
// @Input:	place_id		string	 *
// @Input:  accounts        []string *
func (s *PlaceService) getBlockedIDs(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var IDs []string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if !place.HasReadAccess(requester.ID) {
		response.Error(nested.ERR_ACCESS, []string{""})
		return
	}
	IDs = s.Worker().Model().Place.GetPlaceBlockedIDs(place.ID)
	response.OkWithData(nested.M{"ids": IDs})
		return
}

// @Command:	place/block_ids
// @Input:	place_id		string	 *
// @Input:  IDs             []string *
func (s *PlaceService) blockIDs(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var IDs []string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if v, ok := request.Data["IDs"].(string); ok {
		IDs = strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"IDs"})
		return
	}
	for _,id := range IDs {
		if id == requester.ID {
			response.Error(nested.ERR_ACCESS, []string{"cant block yourself"})
			return
		}
	}
	// Only creators of the place or system admins can do it
	if !place.IsCreator(requester.ID) && !requester.Authority.Admin {
		response.Error(nested.ERR_ACCESS, []string{})
		return
	}
	if s.Worker().Model().Place.AddToBlacklist(place.ID, IDs) {
		response.Ok()
	} else {
		response.Error(nested.ERR_UNKNOWN, []string{})
		return
	}
}

// @Command:	place/unblock_ids
// @Input:	place_id		string	 *
// @Input:  ids             []string *
func (s *PlaceService) unblockIDs(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var IDs []string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if v, ok := request.Data["ids"].(string); ok {
		IDs = strings.SplitN(v, ",", nested.DEFAULT_MAX_RESULT_LIMIT)
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"ids"})
		return
	}
	// Only creators of the place or system admins can do it
	if !place.IsCreator(requester.ID) && !requester.Authority.Admin {
		response.Error(nested.ERR_ACCESS, []string{})
		return
	}
	if s.Worker().Model().Place.RemoveFromBlacklist(place.ID, IDs) {
		response.Ok()
	} else {
		response.Error(nested.ERR_UNKNOWN, []string{})
		return
	}
}


