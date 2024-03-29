package nestedServicePlace

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"regexp"
	"strings"

	"git.ronaksoft.com/nested/server/nested"
)

// @Command: place/add_member
// @Input:	place_id		string	*
// @Input:	member_id		string	*	(comma separated)
func (s *PlaceService) addPlaceMember(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	var memberIDs []string
	if placeID, ok := request.Data["place_id"].(string); ok {
		place = s.Worker().Model().Place.GetByID(placeID, nil)
		if place == nil {
			response.Error(global.ErrUnavailable, []string{"place_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_id"})
		return
	}
	if v, ok := request.Data["member_id"].(string); ok {
		memberIDs = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	} else {
		response.Error(global.ErrIncomplete, []string{"member_id"})
		return
	}
	// for grand places use invite
	if place.IsGrandPlace() {
		response.Error(global.ErrInvalid, []string{"cmd"})
		return
	}
	// check users right access
	access := place.GetAccess(requester.ID)
	if !access[nested.PlaceAccessAddMembers] {
		response.Error(global.ErrAccess, []string{})
		return
	}
	grandPlace := place.GetGrandParent()
	var invalidIDs []string
	for _, m := range memberIDs {
		if grandPlace.IsMember(m) && !place.IsMember(m) {
			if !place.HasKeyholderLimit() {
				s.Worker().Model().Place.AddKeyHolder(place.ID, m)

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
	response.OkWithData(tools.M{"invalid_ids": invalidIDs})
}

// @Command: place/count_unread_posts
// @Input:	place_id		string	*
// @Input:	subs			bool	+
func (s *PlaceService) countPlaceUnreadPosts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var placeIDs []string
	var withSubPlaces bool
	if v, ok := request.Data["place_id"].(string); ok {
		placeIDs = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	} else {
		response.Error(global.ErrInvalid, []string{"place_id"})
		return
	}
	if v, ok := request.Data["subs"].(bool); ok {
		withSubPlaces = v
	}
	r := make([]tools.M, 0)
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
			r = append(r, tools.M{
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
			r = append(r, tools.M{
				"place_id": place.ID,
				"count":    s.Worker().Model().Place.CountUnreadPosts([]string{place.ID}, requester.ID),
			})
		}
	}
	response.OkWithData(tools.M{"counts": r})
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
func (s *PlaceService) createGrandPlace(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	pcr := nested.PlaceCreateRequest{}
	if requester.Limits.GrandPlaces == 0 {
		response.Error(global.ErrLimit, []string{"no_grand_places"})
		return
	}
	if v, ok := request.Data["place_id"].(string); ok {
		pcr.ID = strings.ToLower(v)
		if pcr.ID == "" || len(pcr.ID) > global.DefaultMaxPlaceID {
			response.Error(global.ErrInvalid, []string{"place_id"})
			return
		}
		if matched, err := regexp.MatchString(global.DefaultRegexGrandPlaceID, pcr.ID); err != nil {
			response.Error(global.ErrUnknown, []string{err.Error()})
			return
		} else if !matched {
			response.Error(global.ErrInvalid, []string{"place_id"})
			return
		}
		if !s.Worker().Model().Place.Available(pcr.ID) {
			response.Error(global.ErrDuplicate, []string{"place_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_id"})
		return
	}
	if v, ok := request.Data["place_name"].(string); ok {
		pcr.Name = v
		if pcr.Name == "" || len(pcr.Name) > global.DefaultMaxPlaceName {
			response.Error(global.ErrInvalid, []string{"place_name"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_name"})
		return
	}
	if v, ok := request.Data["place_description"].(string); ok {
		pcr.Description = v
	}

	// Privacy
	if v, ok := request.Data["privacy.receptive"].(string); ok {
		pcr.Privacy.Receptive = nested.PrivacyReceptive(v)
		switch pcr.Privacy.Receptive {
		case nested.PlaceReceptiveExternal, nested.PlaceReceptiveOff:
		default:
			pcr.Privacy.Receptive = nested.PlaceReceptiveOff
		}
	} else {
		pcr.Privacy.Receptive = nested.PlaceReceptiveOff
	}
	if v, ok := request.Data["privacy.search"].(bool); ok {
		pcr.Privacy.Search = v
		if v {
			s.Worker().Model().Search.AddPlaceToSearchIndex(pcr.ID, pcr.Name, pcr.Picture)
		}
	}

	// Policy
	if v, ok := request.Data["policy.add_member"].(string); ok {
		pcr.Policy.AddMember = nested.PolicyGroup(v)
		switch pcr.Policy.AddMember {
		case nested.PlacePolicyCreators, nested.PlacePolicyEveryone:
		default:
			pcr.Policy.AddMember = nested.PlacePolicyCreators
		}
	} else {
		pcr.Policy.AddMember = nested.PlacePolicyCreators
	}
	if v, ok := request.Data["policy.add_post"].(string); ok {
		pcr.Policy.AddPost = nested.PolicyGroup(v)
		switch pcr.Policy.AddPost {
		case nested.PlacePolicyCreators, nested.PlacePolicyEveryone:
		default:
			pcr.Policy.AddPost = nested.PlacePolicyCreators
		}
	} else {
		pcr.Policy.AddPost = nested.PlacePolicyCreators
	}
	if v, ok := request.Data["policy.add_place"].(string); ok {
		pcr.Policy.AddPlace = nested.PolicyGroup(v)
		switch pcr.Policy.AddPlace {
		case nested.PlacePolicyCreators, nested.PlacePolicyEveryone:
		default:
			pcr.Policy.AddPlace = nested.PlacePolicyCreators
		}
	} else {
		pcr.Policy.AddPlace = nested.PlacePolicyCreators
	}

	pcr.GrandParentID = pcr.ID
	pcr.AccountID = requester.ID
	place := s.Worker().Model().Place.CreateGrandPlace(pcr)
	if place == nil {
		response.Error(global.ErrUnknown, []string{"cannot_create_place"})
		return
	}
	// Add the creator of the place
	s.Worker().Model().Place.AddKeyHolder(pcr.ID, requester.ID)
	s.Worker().Model().Place.Promote(pcr.ID, requester.ID)
	s.Worker().Model().Account.SetLimit(requester.ID, "grand_places", requester.Limits.GrandPlaces-1)

	// Enable Notification by default
	s.Worker().Model().Account.SetPlaceNotification(requester.ID, place.ID, true)

	// Add the place to feed
	s.Worker().Model().Account.AddPlaceToBookmarks(requester.ID, place.ID)

	response.OkWithData(tools.M{
		"_id":             place.ID,
		"name":            place.Name,
		"description":     place.Description,
		"picture":         place.Picture,
		"grand_parent_id": place.GrandParentID,
		"privacy":         place.Privacy,
		"policy":          place.Policy,
		"member_type":     nested.MemberTypeCreator,
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
func (s *PlaceService) createLockedPlace(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	pcr := nested.PlaceCreateRequest{}

	if v, ok := request.Data["place_id"].(string); ok {
		pcr.ID = strings.ToLower(v)
		if pcr.ID == "" || len(pcr.ID) > global.DefaultMaxPlaceID {
			response.Error(global.ErrInvalid, []string{"place_id"})
			return
		}
		// check if place is a subplace
		if pos := strings.LastIndex(pcr.ID, "."); pos == -1 {
			response.Error(global.ErrInvalid, []string{"place_id"})
			return
		} else {
			localPlaceID := string(pcr.ID[pos+1:])
			// check if place id is a valid place id
			if matched, err := regexp.MatchString(global.DefaultRegexPlaceID, localPlaceID); err != nil {
				response.Error(global.ErrUnknown, []string{err.Error()})
				return
			} else if !matched {
				response.Error(global.ErrInvalid, []string{"place_id"})
				return
			}
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_id"})
		return
	}
	if v, ok := request.Data["place_name"].(string); ok {
		pcr.Name = v
		if pcr.Name == "" || len(pcr.Name) > global.DefaultMaxPlaceName {
			response.Error(global.ErrInvalid, []string{"place_name"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_name"})
		return
	}
	if v, ok := request.Data["place_description"].(string); ok {
		pcr.Description = v
	}

	// Privacy validation checks
	if v, ok := request.Data["privacy.receptive"].(string); ok {
		pcr.Privacy.Receptive = nested.PrivacyReceptive(v)
		switch pcr.Privacy.Receptive {
		case nested.PlaceReceptiveExternal, nested.PlaceReceptiveInternal, nested.PlaceReceptiveOff:
		default:
			pcr.Privacy.Receptive = nested.PlaceReceptiveOff
		}
	} else {
		pcr.Privacy.Receptive = nested.PlaceReceptiveOff
	}
	if v, ok := request.Data["privacy.search"].(bool); ok {
		pcr.Privacy.Search = v
		if v {
			s.Worker().Model().Search.AddPlaceToSearchIndex(pcr.ID, pcr.Name, pcr.Picture)
		}
	}

	// Policy validation checks
	if v, ok := request.Data["policy.add_member"].(string); ok {
		pcr.Policy.AddMember = nested.PolicyGroup(v)
		switch pcr.Policy.AddMember {
		case nested.PlacePolicyCreators, nested.PlacePolicyEveryone:
		default:
			pcr.Policy.AddMember = nested.PlacePolicyCreators
		}
	} else {
		pcr.Policy.AddMember = nested.PlacePolicyCreators
	}
	if v, ok := request.Data["policy.add_post"].(string); ok {
		pcr.Policy.AddPost = nested.PolicyGroup(v)
		switch pcr.Policy.AddPost {
		case nested.PlacePolicyCreators, nested.PlacePolicyEveryone:
		default:
			pcr.Policy.AddPost = nested.PlacePolicyCreators
		}
	} else {
		pcr.Policy.AddPost = nested.PlacePolicyCreators
	}
	if v, ok := request.Data["policy.add_place"].(string); ok {
		pcr.Policy.AddPlace = nested.PolicyGroup(v)
		switch pcr.Policy.AddPlace {
		case nested.PlacePolicyCreators, nested.PlacePolicyEveryone:
		default:
			pcr.Policy.AddPlace = nested.PlacePolicyCreators
		}
	} else {
		pcr.Policy.AddPlace = nested.PlacePolicyCreators
	}

	// check parent's limitations and access permissions
	parent := s.Worker().Model().Place.GetByID(s.Worker().Model().Place.GetParentID(pcr.ID), nil)
	if parent == nil {
		response.Error(global.ErrInvalid, []string{"place_id"})
		return
	}
	if parent.Level >= global.DefaultPlaceMaxLevel {
		response.Error(global.ErrLimit, []string{"level"})
		return
	}
	if parent.HasChildLimit() {
		response.Error(global.ErrLimit, []string{"place"})
		return
	}

	// check if user has the right to create place
	access := parent.GetAccess(requester.ID)
	if !access[nested.PlaceAccessAddPlace] {
		response.Error(global.ErrAccess, []string{})
		return
	}

	pcr.GrandParentID = parent.GrandParentID
	pcr.AccountID = requester.ID
	grandPlace := s.Worker().Model().Place.GetByID(parent.GrandParentID, nil)
	var place *nested.Place
	if grandPlace.IsPersonal() {
		place = s.Worker().Model().Place.CreatePersonalPlace(pcr)
		if place == nil {
			response.Error(global.ErrUnknown, []string{})
			return
		}
	} else {
		place = s.Worker().Model().Place.CreateLockedPlace(pcr)
		if place == nil {
			response.Error(global.ErrUnknown, []string{})
			return
		}
	}
	// Add the creator of the place
	s.Worker().Model().Place.AddKeyHolder(pcr.ID, requester.ID)
	s.Worker().Model().Place.Promote(pcr.ID, requester.ID)

	// Enable Notification by default
	s.Worker().Model().Account.SetPlaceNotification(requester.ID, place.ID, true)

	// Add place to the user's feed
	s.Worker().Model().Account.AddPlaceToBookmarks(requester.ID, place.ID)

	response.OkWithData(tools.M{
		"_id":             place.ID,
		"name":            place.Name,
		"description":     place.Description,
		"picture":         place.Picture,
		"grand_parent_id": place.GrandParentID,
		"privacy":         place.Privacy,
		"policy":          place.Policy,
		"member_type":     nested.MemberTypeCreator,
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
func (s *PlaceService) createUnlockedPlace(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	pcr := nested.PlaceCreateRequest{}
	if v, ok := request.Data["place_id"].(string); ok {
		pcr.ID = strings.ToLower(v)
		if pcr.ID == "" || len(pcr.ID) > global.DefaultMaxPlaceID {
			response.Error(global.ErrInvalid, []string{"place_id"})
			return
		}
		// check if place is a sub-place
		if pos := strings.LastIndex(pcr.ID, "."); pos == -1 {
			response.Error(global.ErrInvalid, []string{"place_id"})
			return
		} else {
			localPlaceID := string(pcr.ID[pos+1:])
			// check if place id is a valid place id
			if matched, err := regexp.MatchString(global.DefaultRegexPlaceID, localPlaceID); err != nil {
				response.Error(global.ErrUnknown, []string{err.Error()})
				return
			} else if !matched {
				response.Error(global.ErrInvalid, []string{"place_id"})
				return
			}
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_id"})
		return
	}
	if v, ok := request.Data["place_name"].(string); ok {
		pcr.Name = v
		if pcr.Name == "" || len(pcr.Name) > global.DefaultMaxPlaceName {
			response.Error(global.ErrInvalid, []string{"place_name"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_name"})
		return
	}
	if v, ok := request.Data["place_description"].(string); ok {
		pcr.Description = v
	}

	// check parent's limitations and access permissions
	parent := s.Worker().Model().Place.GetByID(s.Worker().Model().Place.GetParentID(pcr.ID), nil)
	if parent == nil {
		response.Error(global.ErrInvalid, []string{"place_id"})
		return
	}
	if parent.HasChildLimit() {
		response.Error(global.ErrLimit, []string{"place"})
		return
	}
	if !parent.IsGrandPlace() {
		response.Error(global.ErrAccess, []string{"open_places_only_on_level_1"})
		return
	}

	// check if user has the right to create place
	access := parent.GetAccess(requester.ID)
	if !access[nested.PlaceAccessAddPlace] {
		response.Error(global.ErrAccess, []string{})
		return
	}

	pcr.GrandParentID = parent.GrandParentID
	pcr.AccountID = requester.ID
	grandPlace := s.Worker().Model().Place.GetByID(parent.GrandParentID, nil)
	var place *nested.Place
	if grandPlace.IsPersonal() {
		response.Error(global.ErrAccess, []string{"no_open_place_in_personal"})
		return
	}
	place = s.Worker().Model().Place.CreateUnlockedPlace(pcr)
	if place == nil {
		response.Error(global.ErrUnknown, []string{})
		return
	}

	// Add the creator of the place
	s.Worker().Model().Place.AddKeyHolder(pcr.ID, requester.ID)
	s.Worker().Model().Place.Promote(pcr.ID, requester.ID)

	// Enable Notification by default
	s.Worker().Model().Account.SetPlaceNotification(requester.ID, place.ID, true)

	// Add place to the user's feed
	s.Worker().Model().Account.AddPlaceToBookmarks(requester.ID, place.ID)

	response.OkWithData(tools.M{
		"_id":             place.ID,
		"name":            place.Name,
		"description":     place.Description,
		"picture":         place.Picture,
		"grand_parent_id": place.GrandParentID,
		"privacy":         place.Privacy,
		"policy":          place.Policy,
		"member_type":     nested.MemberTypeCreator,
		"limits":          place.Limit,
		"counters":        place.Counter,
		"unread_posts":    s.Worker().Model().Place.CountUnreadPosts([]string{place.ID}, requester.ID),
	})

}

// @Command:	place/demote_member
// @Input:	place_id				string	*
// @Input:	member_id			string	*
func (s *PlaceService) demoteMember(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var memberID string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if v, ok := request.Data["member_id"].(string); ok {
		memberID = v
	} else {
		response.Error(global.ErrIncomplete, []string{"member_id"})
		return
	}
	if !place.IsCreator(requester.ID) {
		response.Error(global.ErrAccess, []string{})
		return
	}
	if !place.IsCreator(memberID) {
		response.Error(global.ErrInvalid, []string{"member_id"})
		return
	}
	if place.HasKeyholderLimit() {
		response.Error(global.ErrLimit, []string{"member_id"})
		return
	}

	s.Worker().Model().Place.Demote(place.ID, memberID)

	s.Worker().Pusher().PlaceMemberDemoted(place, requester.ID, memberID)

	response.Ok()

}

// @Command:	place/get_access
// @Input:	place_id				string	*	(comma separated)
func (s *PlaceService) getPlaceAccess(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var places []nested.Place
	if v, ok := request.Data["place_id"].(string); ok {
		placeIDs := strings.SplitN(v, ",", global.DefaultMaxResultLimit)
		places = s.Worker().Model().Place.GetPlacesByIDs(placeIDs)
	} else {
		if v, ok := request.Data["place_ids"].(string); ok {
			placeIDs := strings.SplitN(v, ",", global.DefaultMaxResultLimit)
			places = s.Worker().Model().Place.GetPlacesByIDs(placeIDs)
		} else {
			response.Error(global.ErrInvalid, []string{"place_id"})
			return
		}
	}

	var r []tools.M
	for _, place := range places {
		access := place.GetAccess(requester.ID)
		a := make([]string, 0, 10)
		a = a[:0]
		for k, v := range access {
			if v {
				a = append(a, k)
			}
		}
		r = append(r, tools.M{
			"_id":      place.ID,
			"place_id": place.ID,
			"access":   a,
		})
	}
	response.OkWithData(tools.M{"places": r})
}

// @Command:	place/get
// @Input:	place_id				string	*
func (s *PlaceService) getPlaceInfo(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	response.OkWithData(s.Worker().Map().Place(requester, *place, place.GetAccess(requester.ID)))
}

// @Command:	place/get_many
// @Input:	place_id				string	*	(comma separated)
// @Input:	member_id			string	*
func (s *PlaceService) getManyPlacesInfo(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var places []nested.Place
	if v, ok := request.Data["place_id"].(string); ok {
		placeIDs := strings.Split(v, ",")
		places = s.Worker().Model().Place.GetPlacesByIDs(placeIDs)
		if len(places) == 0 {
			response.OkWithData(tools.M{"places": []tools.M{}})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_id"})
		return
	}
	r := make([]tools.M, 0, len(places))
	for _, place := range places {
		r = append(r, s.Worker().Map().Place(requester, place, place.GetAccess(requester.ID)))
	}
	response.OkWithData(tools.M{"places": r})
}

// @Command:	place/get_activities
// @Input:	place_id				string	*
// @Input:	details				bool		+
func (s *PlaceService) getPlaceActivities(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	var details bool
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}

	if v, ok := request.Data["details"].(bool); ok {
		details = v
	}

	if !place.HasReadAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{""})
		return
	}

	pg := s.Worker().Argument().GetPagination(request)
	ta := s.Worker().Model().PlaceActivity.GetActivitiesByPlace(place.ID, pg)
	d := make([]tools.M, 0, pg.GetLimit())
	for _, v := range ta {
		d = append(d, s.Worker().Map().PlaceActivity(requester, v, details))
	}
	response.OkWithData(tools.M{
		"skip":       pg.GetSkip(),
		"limit":      pg.GetLimit(),
		"activities": d,
	})
}

// @Command:	place/get_files
// @Input:	place_id		string	*
// @Input:	filter		string	+ AUD | DOC | IMG | VID | OTH | all
// @Input:	filename		string	+
func (s *PlaceService) getPlaceFiles(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var filter, filename string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if !place.HasReadAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{""})
		return
	}
	if v, ok := request.Data["filter"].(string); ok {
		switch v {
		case nested.FileTypeAudio, nested.FileTypeDocument, nested.FileTypeImage, nested.FileTypeVideo, nested.FileTypeOther:
			filter = v
		default:
			filter = nested.FileTypeAll
		}

	}
	if v, ok := request.Data["filename"].(string); ok {
		filename = v
	}
	access := place.GetAccess(requester.ID)
	if !access[nested.PlaceAccessReadPost] {
		response.Error(global.ErrAccess, []string{})
		return
	}
	pg := s.Worker().Argument().GetPagination(request)
	result := s.Worker().Model().File.GetFilesByPlace(place.ID, filter, filename, pg)
	r := make([]tools.M, 0, len(result))
	for _, f := range result {
		d := s.Worker().Map().FileInfo(f.File)
		d["post_id"] = f.PostId.Hex()
		r = append(r, d)
	}
	response.OkWithData(tools.M{"files": r})
}

// @Command:	place/get_unread_posts
// @Input:	place_id		string	*
// @Input:	subs			bool		+	(default: FALSE)
func (s *PlaceService) getPlaceUnreadPosts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var subPlaces bool
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}

	if v, ok := request.Data["subs"].(bool); ok {
		subPlaces = v
	}

	if !place.HasReadAccess(requester.ID) {
		response.Error(global.ErrUnavailable, []string{"place_id"})
		return
	}

	pg := s.Worker().Argument().GetPagination(request)
	posts := s.Worker().Model().Post.GetUnreadPostsByPlace(place.ID, requester.ID, subPlaces, pg)
	r := make([]tools.M, 0, len(posts))
	for _, post := range posts {
		r = append(r, s.Worker().Map().Post(requester, post, true))
	}
	response.OkWithData(tools.M{
		"skip":  pg.GetSkip(),
		"limit": pg.GetLimit(),
		"posts": r,
	})
	return
}

// @Command:	place/get_notification
// @Input:	place_id		string	*
func (s *PlaceService) getPlaceNotification(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if s.Worker().Model().Group.ItemExists(place.Groups["_ntfy"], requester.ID) {
		response.OkWithData(tools.M{"state": true})
	} else {
		response.OkWithData(tools.M{"state": false})
	}
	return
}

// @Command:	place/get_creators
// @Input:	place_id		string	*
func (s *PlaceService) getPlaceCreators(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	access := place.GetAccess(requester.ID)
	if !access[nested.PlaceAccessSeeMembers] {
		response.Error(global.ErrAccess, []string{})
		return
	}
	pg := s.Worker().Argument().GetPagination(request)
	iStart := pg.GetSkip()
	iEnd := iStart + pg.GetLimit()
	if iEnd > len(place.CreatorIDs) {
		iEnd = len(place.CreatorIDs)
	}

	var r []tools.M
	for _, v := range place.CreatorIDs[iStart:iEnd] {
		m := s.Worker().Model().Account.GetByID(v, nil)
		r = append(r, s.Worker().Map().Account(*m, false))
	}

	response.OkWithData(tools.M{
		"total":    place.Counter.Creators,
		"skip":     pg.GetSkip(),
		"limit":    pg.GetLimit(),
		"creators": r,
	})
}

// @Command:	place/get_key_holders
// @Input:	place_id		string	*
func (s *PlaceService) getPlaceKeyholders(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	access := place.GetAccess(requester.ID)
	if !access[nested.PlaceAccessSeeMembers] {
		response.Error(global.ErrAccess, []string{})
		return
	}
	pg := s.Worker().Argument().GetPagination(request)
	iStart := pg.GetSkip()
	iEnd := iStart + pg.GetLimit()
	if iEnd > len(place.KeyholderIDs) {
		iEnd = len(place.KeyholderIDs)
	}

	var r []tools.M
	for _, v := range place.KeyholderIDs[iStart:iEnd] {
		m := s.Worker().Model().Account.GetByID(v, nil)
		r = append(r, s.Worker().Map().Account(*m, false))
	}
	response.OkWithData(tools.M{
		"total":       place.Counter.Keyholders,
		"skip":        pg.GetSkip(),
		"limit":       pg.GetLimit(),
		"key_holders": r,
	})
}

// @Command:	place/get_members
// @Input:	place_id		string	*
func (s *PlaceService) getPlaceMembers(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	access := place.GetAccess(requester.ID)
	if !access[nested.PlaceAccessSeeMembers] {
		response.Error(global.ErrAccess, []string{})
		return
	}

	// TODO:: use s.Worker().Model().Account.GetAccountsByIDs instead
	rKeyholders := make([]tools.M, 0, len(place.KeyholderIDs))
	for _, v := range place.KeyholderIDs {
		m := s.Worker().Model().Account.GetByID(v, nil)
		rKeyholders = append(rKeyholders, s.Worker().Map().Account(*m, false))
	}
	rCreators := make([]tools.M, 0, len(place.CreatorIDs))
	for _, v := range place.CreatorIDs {
		m := s.Worker().Model().Account.GetByID(v, nil)
		rCreators = append(rCreators, s.Worker().Map().Account(*m, false))
	}

	response.OkWithData(tools.M{
		"key_holders": rKeyholders,
		"creators":    rCreators,
	})
}

// @Command:	place/get_sub_places
// @Input:	place_id		string	*
func (s *PlaceService) getSubPlaces(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	mapPlaceIDs := tools.M{}
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
	var r []tools.M
	for _, place := range places {
		r = append(r, s.Worker().Map().Place(requester, place, place.GetAccess(requester.ID)))
	}
	response.OkWithData(tools.M{"places": r})
}

// @Command:	place/get_mutual_places
// @Input:	account_id		string	*
func (s *PlaceService) getMutualPlaces(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var accountID string
	if v, ok := request.Data["account_id"].(string); ok {
		accountID = v
		if !s.Worker().Model().Account.Exists(accountID) {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}
	placeIDs := s.Worker().Model().Account.GetMutualPlaceIDs(requester.ID, accountID)
	r := make([]tools.M, 0, len(placeIDs))
	iStart := 0
	iLength := global.DefaultMaxResultLimit
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
	response.OkWithData(tools.M{"places": r})
}

// @Command:	place/get_posts
// @Input:	place_id		string	*
// @Input:	by_update	bool		+
func (s *PlaceService) getPlacePosts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var sortItem string
	var place *nested.Place
	var posts []nested.Post
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	sortItem = nested.PostSortTimestamp
	if v, ok := request.Data["by_update"].(bool); ok {
		if v {
			sortItem = nested.PostSortLastUpdate
		}
	}

	// user must have read access in place
	if !place.HasReadAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{""})
		return
	}

	pg := s.Worker().Argument().GetPagination(request)
	if place.ID == requester.ID {
		posts = s.Worker().Model().Post.GetPostsOfPlaces([]string{place.ID, "*"}, sortItem, pg)
	} else {
		posts = s.Worker().Model().Post.GetPostsByPlace(place.ID, sortItem, pg)
	}
	r := make([]tools.M, 0, len(posts))

	for _, post := range posts {
		r = append(r, s.Worker().Map().Post(requester, post, true))
	}
	response.OkWithData(tools.M{
		"skip":  pg.GetSkip(),
		"limit": pg.GetLimit(),
		"posts": r,
	})
	return
}

// @Command:	place/invite_member
// @Input:	place_id			string	*
// @Input:	member_id		string	*	(comma separated)
func (s *PlaceService) invitePlaceMember(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var memberIDs []string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if v, ok := request.Data["member_id"].(string); ok {
		memberIDs = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	} else {
		response.Error(global.ErrIncomplete, []string{"member_id"})
		return
	}

	// user can only invite people to grand place otherwise they must be added
	if !place.IsGrandPlace() {
		response.Error(global.ErrInvalid, []string{"cmd"})
		return
	}

	// check if user has the right permission
	access := place.GetAccess(requester.ID)
	if !access[nested.PlaceAccessAddMembers] {
		response.Error(global.ErrAccess, []string{})
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
			s.Worker().Model().Place.AddKeyHolder(place.ID, m)

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
	response.OkWithData(tools.M{"invalid_ids": invalidIDs})
}

// @Command:	place/leave
// @Input:	place_id		string	*
func (s *PlaceService) leavePlace(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}

	if place.IsPersonal() {
		response.Error(global.ErrAccess, []string{"cannot_leave_personal_place"})
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
		response.Error(global.ErrInvalid, []string{"you_are_not_member"})
		return
	}
	response.Ok()
	return
}

// @Command:	place/mark_all_read
// @Input:	place_id		string	*
func (s *PlaceService) markAllPostsAsRead(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	s.Worker().Model().Post.MarkAsReadByPlace(place.ID, requester.ID)
	response.Ok()
}

// @Command:	place/available
// @Input:	place_id		string	*
func (s *PlaceService) placeIDAvailable(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	if placeID, ok := request.Data["place_id"].(string); ok {
		if s.Worker().Model().Place.Available(strings.ToLower(placeID)) {
			response.Ok()
		} else {
			response.Error(global.ErrUnavailable, []string{"place_id"})
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_id"})
		return
	}
	return
}

// @Command:	place/promote_member
// @Input:	place_id		string	*
// @Input:	member_id	string	*
func (s *PlaceService) promoteMember(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var memberID string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if v, ok := request.Data["member_id"].(string); ok {
		memberID = v
		if requester.ID == memberID || !place.IsCreator(requester.ID) {
			response.Error(global.ErrAccess, []string{})
			return
		}
	}
	if !place.IsKeyholder(memberID) {
		response.Error(global.ErrInvalid, []string{"member_id"})
		return
	}
	if place.HasCreatorLimit() {
		response.Error(global.ErrLimit, []string{"creators"})
		return
	}
	s.Worker().Model().Place.Promote(place.ID, memberID)

	s.Worker().Pusher().PlaceMemberPromoted(place, requester.ID, memberID)

	response.Ok()
}

// @Command:	place/pin_post
// @Input:	place_id		string	*
// @Input: post_id          string *
func (s *PlaceService) pinPost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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
		response.Error(global.ErrAccess, []string{})
		return
	}
	if !post.IsInPlace(place.ID) {
		response.Error(global.ErrUnavailable, []string{"post_id"})
		return
	}
	s.Worker().Model().Place.PinPost(place.ID, post.ID)
	response.Ok()
}

// @Command:	place/remove
// @Input:	place_id		string	*
func (s *PlaceService) remove(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}

	// check if user has the right permission
	access := place.GetAccess(requester.ID)
	if !access[nested.PlaceAccessRemovePlace] {
		response.Error(global.ErrAccess, []string{})
		return
	}
	if place.Counter.Children > 0 {
		response.Error(global.ErrAccess, []string{"remove_children_first"})
		return
	}
	s.Worker().Model().Place.RemoveDefaultPlaces([]string{place.ID})

	if s.Worker().Model().Place.Remove(place.ID, requester.ID) {
		s.Worker().Model().Account.IncrementLimit(place.MainCreatorID, "grand_places", 1)
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}

	// Remove Place from search index
	s.Worker().Model().Search.RemovePlaceFromSearchIndex(place.ID)

}

// @Command: place/remove_all_posts
// @Input: place_id			string *
func (s *PlaceService) removeAllPosts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}

	access := place.GetAccess(requester.ID)
	if access[nested.PlaceAccessRemovePost] || requester.Authority.Admin {
		s.Worker().Model().Post.RemoveByPlaceID(requester.ID, place.ID)
		response.Ok()
	} else {
		response.Error(global.ErrAccess, []string{})
	}
	return
}

// @Command:	place/remove_favorite
// @Input:	place_id		string	*
func (s *PlaceService) removePlaceFromFavorites(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	access := place.GetAccess(requester.ID)
	if access[nested.PlaceAccessReadPost] {
		s.Worker().Model().Account.RemovePlaceFromBookmarks(requester.ID, place.ID)
		response.Ok()
	} else {
		response.Error(global.ErrAccess, []string{})
	}
	return
}

// @Command:	place/remove_member
// @Input:	place_id		string	*
// @Input:	member_id	string	*
func (s *PlaceService) removeMember(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var memberID string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if v, ok := request.Data["member_id"].(string); ok {
		memberID = v
	}
	if memberID == requester.ID {
		response.Error(global.ErrAccess, []string{"cant remove yourself, leave instead"})
		return
	}
	access := place.GetAccess(requester.ID)
	if !access[nested.PlaceAccessRemoveMembers] {
		response.Error(global.ErrAccess, []string{})
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
		response.Error(global.ErrInvalid, []string{"member_id"})
	}
	response.Ok()
	return
}

// @Command:	place/remove_picture
// @Input:	place_id		string	*
func (s *PlaceService) removePicture(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	access := place.GetAccess(requester.ID)
	if access[nested.PlaceAccessControl] {
		s.Worker().Model().Place.SetPicture(place.ID, nested.Picture{})
		response.Ok()
	} else {
		response.Error(global.ErrAccess, []string{})
	}
	return
}

// @Command:	place/add_favorite
// @Input:	place_id		string	*
func (s *PlaceService) setPlaceAsFavorite(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	access := place.GetAccess(requester.ID)
	if access[nested.PlaceAccessReadPost] {
		s.Worker().Model().Account.AddPlaceToBookmarks(requester.ID, place.ID)
		response.Ok()
	} else {
		response.Error(global.ErrAccess, []string{})
	}
	return
}

// @Command:	place/set_notification
// @Input:	place_id		string	*
// @Input:	state		bool		*
func (s *PlaceService) setPlaceNotification(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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
	if !access[nested.PlaceAccessReadPost] {
		response.Error(global.ErrAccess, []string{})
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
func (s *PlaceService) setPicture(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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
			response.Error(global.ErrUnavailable, []string{"universal_id"})
			return
		}
		pic = fileInfo.Thumbnails
	}
	access := place.GetAccess(requester.ID)
	if !access[nested.PlaceAccessControl] {
		response.Error(global.ErrAccess, []string{})
		return
	}
	s.Worker().Model().Place.SetPicture(place.ID, pic)
	if place.Privacy.Search {
		s.Worker().Model().Search.AddPlaceToSearchIndex(place.ID, place.Name, pic)
	}
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
func (s *PlaceService) update(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	placeUpdateRequest := tools.M{}
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
			case nested.PlacePolicyCreators, nested.PlacePolicyEveryone:
				placeUpdateRequest["policy.add_member"] = v
			}
		}

	}
	if place.Privacy.Locked == true {
		if v, ok := request.Data["privacy.search"].(bool); ok {
			placeUpdateRequest["privacy.search"] = v
			if v {
				s.Worker().Model().Search.AddPlaceToSearchIndex(place.ID, place.Name, place.Picture)
			} else {
				s.Worker().Model().Search.RemovePlaceFromSearchIndex(place.ID)
			}
		}
		if v, ok := request.Data["privacy.receptive"].(string); ok {
			if place.IsGrandPlace() {
				switch nested.PrivacyReceptive(v) {
				case nested.PlaceReceptiveExternal:
					placeUpdateRequest["privacy.receptive"] = v
				case nested.PlaceReceptiveInternal, nested.PlaceReceptiveOff:
					placeUpdateRequest["privacy.receptive"] = v
					s.Worker().Model().Search.RemovePlaceFromSearchIndex(place.ID)
				default:
					response.Error(global.ErrInvalid, []string{"privacy.receptive"})
					return
				}
			} else {
				switch nested.PrivacyReceptive(v) {
				case nested.PlaceReceptiveExternal, nested.PlaceReceptiveInternal:
					placeUpdateRequest["privacy.receptive"] = v
				case nested.PlaceReceptiveOff:
					placeUpdateRequest["privacy.receptive"] = v
					placeUpdateRequest["privacy.search"] = false
					s.Worker().Model().Search.RemovePlaceFromSearchIndex(place.ID)
				default:
					response.Error(global.ErrInvalid, []string{"privacy.receptive"})
					return
				}
			}

		}
		if v, ok := request.Data["policy.add_post"].(string); ok {
			switch nested.PolicyGroup(v) {
			case nested.PlacePolicyCreators, nested.PlacePolicyEveryone:
				placeUpdateRequest["policy.add_post"] = v

			}
		}
		if v, ok := request.Data["policy.add_place"].(string); ok {
			if !place.IsPersonal() {
				switch nested.PolicyGroup(v) {
				case nested.PlacePolicyCreators, nested.PlacePolicyEveryone:
					placeUpdateRequest["policy.add_place"] = v

				}
			}
		}
	}
	access := place.GetAccess(requester.ID)
	if !access[nested.PlaceAccessControl] {
		response.Error(global.ErrAccess, []string{})
		return
	}
	if len(placeUpdateRequest) > 0 {
		s.Worker().Model().Place.Update(place.ID, placeUpdateRequest)
		s.Worker().Pusher().PlaceSettingsUpdated(place, requester.ID)
	}
	response.OkWithData(tools.M{"applied": placeUpdateRequest})
	return
}

// @Command:	place/unpin_post
// @Input:	place_id		string	*
// @Input: post_id          string *
func (s *PlaceService) unpinPost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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
		response.Error(global.ErrAccess, []string{})
		return
	}
	if !post.IsInPlace(place.ID) {
		response.Error(global.ErrUnavailable, []string{"post_id"})
		return
	}
	s.Worker().Model().Place.UnpinPost(place.ID, post.ID)
	response.Ok()
}

// @Command:	place/get_blocked_addresses
// @Input:	    place_id		string	 *
func (s *PlaceService) getBlockedAddresses(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var Addresses []string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if !place.HasReadAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{""})
		return
	}
	Addresses = s.Worker().Model().Place.GetPlaceBlockedAddresses(place.ID)
	response.OkWithData(tools.M{"addresses": Addresses})
	return
}

// @Command:	place/add_to_blacklist
// @Input:	place_id		string	 *
// @Input:  addresses       []string *
func (s *PlaceService) addToBlackList(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var addresses []string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if v, ok := request.Data["addresses"].(string); ok {
		addresses = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	} else {
		response.Error(global.ErrIncomplete, []string{"addresses"})
		return
	}
	for _, id := range addresses {
		if id == requester.ID {
			response.Error(global.ErrAccess, []string{"cant block yourself"})
			return
		}
	}
	// Only creators of the place or system admins can do it
	if !place.IsCreator(requester.ID) && !requester.Authority.Admin {
		response.Error(global.ErrAccess, []string{})
		return
	}
	if s.Worker().Model().Place.AddToBlacklist(place.ID, addresses) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
		return
	}
}

// @Command:	place/remove_from_blacklist
// @Input:	place_id		string	 *
// @Input:  addresses       []string *
func (s *PlaceService) removeFromBlacklist(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var addresses []string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if v, ok := request.Data["addresses"].(string); ok {
		addresses = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	} else {
		response.Error(global.ErrIncomplete, []string{"addresses"})
		return
	}
	// Only creators of the place or system admins can do it
	if !place.IsCreator(requester.ID) && !requester.Authority.Admin {
		response.Error(global.ErrAccess, []string{})
		return
	}
	if s.Worker().Model().Place.RemoveFromBlacklist(place.ID, addresses) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
		return
	}
}
