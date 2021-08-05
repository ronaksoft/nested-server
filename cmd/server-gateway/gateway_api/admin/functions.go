package nestedServiceAdmin

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"html/template"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/nested"
)

// @Command: admin/set_message_template
// @Input:  msg_id          string      *
// @Input:  msg_body        string      *
// @Input:  msg_subject     string      *
func (s *AdminService) setMessageTemplate(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var msgID, msgSubject, msgBody string
	if v, ok := request.Data["msg_id"].(string); ok && len(v) > 0 {
		msgID = v
	} else {
		response.Error(global.ErrIncomplete, []string{"msg_id"})
		return
	}
	if v, ok := request.Data["msg_body"].(string); ok && len(v) > 0 {
		msgBody = v
	} else {
		response.Error(global.ErrIncomplete, []string{"msg_body"})
		return
	}
	if v, ok := request.Data["msg_subject"].(string); ok && len(v) > 0 {
		msgSubject = v
	} else {
		response.Error(global.ErrIncomplete, []string{"msg_subject"})
		return
	}
	if s.Worker().Model().System.SetMessageTemplate(msgID, msgSubject, msgBody) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
}

// @Command: admin/get_message_templates
func (s *AdminService) getMessageTemplates(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	response.OkWithData(tools.M{
		"message_templates": s.Worker().Model().System.GetMessageTemplates(),
	})
}

// @Command: admin/remove_message_template
// @Input:  msg_id          string      *
func (s *AdminService) removeMessageTemplates(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var msgID string
	if v, ok := request.Data["msg_id"].(string); ok && len(v) > 0 {
		msgID = v
	} else {
		response.Error(global.ErrIncomplete, []string{"msg_id"})
		return
	}
	s.Worker().Model().System.RemoveMessageTemplate(msgID)
	response.Ok()
}

// @Command: admin/health_check
// @Input:	check_state		bool		+
func (s *AdminService) checkSystemHealth(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var checkState bool
	if v, ok := request.Data["check_state"].(bool); ok {
		checkState = v
	}
	HEALTH_CHECK_IS_RUNNING := s.Worker().Server().GetFlags().HealthCheckRunning
	if checkState {
		response.OkWithData(tools.M{"running_health_check": HEALTH_CHECK_IS_RUNNING})
		return
	}

	if !HEALTH_CHECK_IS_RUNNING {
		go func() {
			s.Worker().Server().SetHealthCheckState(true)
			s.Worker().Model().ModelCheckHealth()
			s.Worker().Server().SetHealthCheckState(false)
		}()
		response.Ok()
	} else {
		response.Error(global.ErrAccess, []string{"already_running"})
	}
	return
}

// @Command:	admin/create_post
// @Input:  subject			string	+
// @Input:  trequest, responseets			string 	+	(comma separated)
// @Input:  attaches			string 	+	(comma separated)
// @Input:  content_type		string	+	(text/plain | text/html)
// @Input:  iframe_url         string +
func (s *AdminService) createPost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var targets []string
	var attachments []string
	var subject, body, content_type, iframeUrl string
	var labels []nested.Label
	if v, ok := request.Data["targets"].(string); ok {
		targets = strings.SplitN(v, ",", global.DefaultPostMaxTargets)
		if len(targets) == 0 {
			response.Error(global.ErrInvalid, []string{"targets"})
			return
		}
	} else {
		response.Error(global.ErrInvalid, []string{"targets"})
		return
	}
	if v, ok := request.Data["label_id"].(string); ok {
		labelIDs := strings.SplitN(v, ",", global.DefaultPostMaxLabels)
		labels = s.Worker().Model().Label.GetByIDs(labelIDs)
	} else {
		labels = []nested.Label{}
	}
	if v, ok := request.Data["attaches"].(string); ok && v != "" {
		attachments = strings.SplitN(v, ",", global.DefaultPostMaxAttachments)
	} else {
		attachments = []string{}
	}
	if v, ok := request.Data["content_type"].(string); ok {
		switch v {
		case nested.ContentTypeTextHtml, nested.ContentTypeTextPlain:
			content_type = v
		default:
			content_type = nested.ContentTypeTextPlain
		}
	} else {
		content_type = nested.ContentTypeTextPlain
	}
	if v, ok := request.Data["subject"].(string); ok {
		subject = v
	}
	if v, ok := request.Data["body"].(string); ok {
		body = v
	}
	if v, ok := request.Data["iframe_url"].(string); ok {
		iframeUrl = v
	}

	if "" == strings.Trim(subject, " ") && "" == strings.Trim(body, " ") && len(attachments) == 0 {
		response.Error(global.ErrIncomplete, []string{"subject", "body"})
		return
	}
	// Separate places and emails
	mPlaces := make(map[string]bool)
	mEmails := make(map[string]bool)
	for _, v := range targets {
		if idx := strings.Index(v, "@"); idx != -1 {
			domains := strings.Split(s.Worker().Config().GetString("DOMAINS"), ",")
			isInternal := false
			for _, domain := range domains {
				if strings.HasSuffix(strings.ToLower(v), fmt.Sprintf("@%s", domain)) {
					mPlaces[v[:idx]] = true
					isInternal = true
					break
				}
			}
			if isInternal == false {
				mEmails[v] = true
			}
		} else if s.Worker().Model().Place.Exists(v) {
			mPlaces[v] = true
		}
	}
	notValidPlaces := make([]string, 0, global.DefaultPostMaxTargets)
	for k := range mPlaces {
		place := s.Worker().Model().Place.GetByID(k, nil)
		if place == nil {
			notValidPlaces = append(notValidPlaces, k)
			delete(mPlaces, k)
			continue
		}
	}

	if len(mPlaces) == 0 && len(mEmails) == 0 {
		response.Error(global.ErrInvalid, []string{"targets"})
		return
	}

	for i, v := range attachments {
		if v == "" || !s.Worker().Model().File.Exists(nested.UniversalID(v)) {
			if len(attachments) > 1 {
				attachments[i] = attachments[len(attachments)-1]
				attachments = attachments[:len(attachments)-1]
			} else {
				attachments = attachments[:0]
				break
			}
		}
	}

	// Let's create the post
	var places, emails []string
	for k := range mPlaces {
		places = append(places, k)
	}
	for k := range mEmails {
		emails = append(emails, k)
	}

	pcr := nested.PostCreateRequest{
		PlaceIDs:    places,
		Recipients:  emails,
		ContentType: content_type,
		SenderID:    requester.ID,
		SystemData: nested.PostSystemData{
			NoComment: true,
		},
	}

	// Make attachments unique and add them to PostCreateRequest
	mapAttachments := tools.MB{}
	for _, attachID := range attachments {
		mapAttachments[attachID] = true
	}

	for attachID := range mapAttachments {
		pcr.AttachmentIDs = append(pcr.AttachmentIDs, nested.UniversalID(attachID))
	}

	// Set Body for PostCreateRequest
	pcr.Body = body
	pcr.IFrameUrl = iframeUrl

	// check if subject does not exceed the limit
	if len(subject) > 255 {
		pcr.Subject = string(subject[:255])
	} else {
		pcr.Subject = subject
	}

	post := s.Worker().Model().Post.AddPost(pcr)
	if post == nil {
		response.Error(global.ErrUnknown, []string{})
		return
	}

	for _, label := range labels {
		if label.Public || label.IsMember(requester.ID) {
			post.AddLabel(requester.ID, label.ID)
		}
	}

	s.Worker().Pusher().PostAdded(post)

	// Send Emails
	if len(emails) > 0 {
		mailReq := api.MailRequest{
			Host:     requester.Mail.OutgoingSMTPHost,
			Port:     requester.Mail.OutgoingSMTPPort,
			Username: requester.Mail.OutgoingSMTPUser,
			Password: requester.Mail.OutgoingSMTPPass,
			PostID:   post.ID,
		}
		s.Worker().Mailer().SendRequest(mailReq)
	}

	response.OkWithData(tools.M{
		"post_id":        post.ID,
		"invalid_places": notValidPlaces,
	})

}

// @Command:	admin/add_comment
// @Input:  post_id			string	+
// @Input:  txt     			string 	+
// @Input:  attachment_id	    string	+
func (s *AdminService) addComment(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	var txt string
	var attachmentID nested.UniversalID
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	if v, ok := request.Data["txt"].(string); ok {
		txt = v
	}

	// if post does not allow commenting return error
	if post.SystemData.NoComment {
		response.Error(global.ErrAccess, []string{"no_comment"})
		return
	}

	if v, ok := request.Data["attachment_id"].(string); ok && strings.HasPrefix(v, "VOC") {
		attachmentID = nested.UniversalID(v)
		attachment := s.Worker().Model().File.GetByID(attachmentID, nil)
		if attachment == nil {
			response.Error(global.ErrInvalid, []string{"attachment_id"})
			return
		}
		txt = "[VOICE COMMENT]"
	} else {
		// comment with empty text is not allowed
		if txt == "" {
			response.Error(global.ErrInvalid, []string{"txt"})
			return
		}
	}

	// create the comment object
	c := s.Worker().Model().Post.AddComment(post.ID, requester.ID, txt, attachmentID)
	if c == nil {
		response.Error(global.ErrUnknown, []string{"internal_error"})
	}

	// mark post as read
	s.Worker().Model().Post.MarkAsRead(post.ID, requester.ID)

	// handle push messages (notification and activity)
	go s.Worker().Pusher().PostCommentAdded(post, c)

	response.OkWithData(tools.M{"comment_id": c.ID})
}

// @Command: admin/promote
// @Input:	account_id		string		*
func (s *AdminService) promoteAccount(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var accountID string
	if v, ok := request.Data["account_id"].(string); ok {
		accountID = v
		if account := s.Worker().Model().Account.GetByID(accountID, nil); account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}
	s.Worker().Model().Account.SetAdmin(accountID, true)
	response.Ok()
}

// @Command: admin/demote
// @Input:	account_id		string		*
func (s *AdminService) demoteAccount(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var accountID string
	if v, ok := request.Data["account_id"].(string); ok {
		accountID = v
		if account := s.Worker().Model().Account.GetByID(accountID, nil); account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}
	s.Worker().Model().Account.SetAdmin(accountID, false)
	response.Ok()
}

// @Command:	admin/create_grand_place
// @Input:	place_id				string	*
// @Input:	place_name			string	*
// @Input:	place_description	string	+
// @Input:	privacy.receptive	string	*	(external | off)
// @Input:	privacy.search		bool		*
// @Input:	policy.add_member	string	*	(creators | everyone)
// @Input:	policy.add_post		string	*	(creators | everyone)
// @Input:	policy.add_place		string	*	(creators | everyone)
func (s *AdminService) createGrandPlace(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	pcr := nested.PlaceCreateRequest{}
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

	response.OkWithData(tools.M{
		"_id":             place.ID,
		"name":            place.Name,
		"description":     place.Description,
		"picture":         place.Picture,
		"grand_parent_id": place.GrandParentID,
		"privacy":         place.Privacy,
		"policy":          place.Policy,
		"limits":          place.Limit,
		"counters":        place.Counter,
	})
	return
}

// @Command:	admin/create_place
// @Input:	place_id				string	*
// @Input:	place_name			string	*
// @Input:	place_description	string	+
// @Input:	privacy.receptive	string	*	(external | internal | off)
// @Input:	privacy.search		bool		*
// @Input:	policy.add_member	string	*	(creators | everyone)
// @Input:	policy.add_post		string	*	(creators | everyone)
// @Input:	policy.add_place		string	*	(creators | everyone)
func (s *AdminService) createPlace(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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
				log.Println(err.Error())
				response.Error(global.ErrUnknown, []string{})
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

	pcr.GrandParentID = parent.GrandParentID
	pcr.AccountID = requester.ID
	grandPlace := s.Worker().Model().Place.GetByID(parent.GrandParentID, nil)
	var place *nested.Place
	if grandPlace.IsPersonal() {
		pcr.AccountID = grandPlace.MainCreatorID
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

	response.OkWithData(tools.M{
		"_id":             place.ID,
		"name":            place.Name,
		"description":     place.Description,
		"picture":         place.Picture,
		"grand_parent_id": place.GrandParentID,
		"privacy":         place.Privacy,
		"policy":          place.Policy,
		"limits":          place.Limit,
		"counters":        place.Counter,
	})
	return
}

// @Command: admin/place_update
// @Input:	place_id					string		*
// @Input:	place_description				string		+
// @Input:	place_name						string		+
// @Input:	limits.key_holders		int			+
// @Input:	limits.creators			int			+
// @Input:	limits.size				int			+
// @Input:	limits.childs			int			+
// @Input:	privacy.search			bool		+
// @Input:	privacy.receptive		string	+	(external | internal | off)
// @Input:	policy.add_post			string	+	(creators | everyone)
// @Input:	policy.add_member		string	+	(creators | everyone)
// @Input:	policy.add_place			string	+	(creators | everyone)
func (s *AdminService) updatePlace(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	placeUpdate := tools.M{}
	placeLimitsUpdate := nested.MI{}
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}

	if placeName, ok := request.Data["place_name"].(string); ok {
		if len(placeName) > 0 && len(placeName) < global.DefaultMaxPlaceName {
			placeUpdate["name"] = placeName
			place.Name = placeName
		}
	}

	if placeDescription, ok := request.Data["place_description"].(string); ok {
		placeUpdate["description"] = placeDescription
	}

	if v, ok := request.Data["policy.add_member"].(string); ok {
		if !place.IsPersonal() {
			switch nested.PolicyGroup(v) {
			case nested.PlacePolicyCreators, nested.PlacePolicyEveryone:
				placeUpdate["policy.add_member"] = v
			}
		}

	}
	if place.Privacy.Locked == true {
		if v, ok := request.Data["privacy.search"].(bool); ok {
			placeUpdate["privacy.search"] = v
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
					placeUpdate["privacy.receptive"] = v
				case nested.PlaceReceptiveInternal, nested.PlaceReceptiveOff:
					placeUpdate["privacy.receptive"] = v
					s.Worker().Model().Search.RemovePlaceFromSearchIndex(place.ID)
				default:
					response.Error(global.ErrInvalid, []string{"privacy.receptive"})
					return
				}
			} else {
				switch nested.PrivacyReceptive(v) {
				case nested.PlaceReceptiveExternal, nested.PlaceReceptiveInternal:
					placeUpdate["privacy.receptive"] = v
				case nested.PlaceReceptiveOff:
					placeUpdate["privacy.receptive"] = v
					placeUpdate["privacy.search"] = false
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
				placeUpdate["policy.add_post"] = v

			}
		}
		if v, ok := request.Data["policy.add_place"].(string); ok {
			if !place.IsPersonal() {
				switch nested.PolicyGroup(v) {
				case nested.PlacePolicyCreators, nested.PlacePolicyEveryone:
					placeUpdate["policy.add_place"] = v

				}
			}
		}
	}
	if limitsKeyHolder, ok := request.Data["limits.key_holders"].(float64); ok {
		placeLimitsUpdate["limits.key_holders"] = int(limitsKeyHolder)
	} else if v, ok := request.Data["limits.key_holders"].(string); ok {
		placeLimitsUpdate["limits.key_holders"], _ = strconv.Atoi(v)
	}
	if limitCreator, ok := request.Data["limits.creators"].(float64); ok {
		placeLimitsUpdate["limits.creators"] = int(limitCreator)
	} else if v, ok := request.Data["limits.creators"].(string); ok {
		placeLimitsUpdate["limits.creators"], _ = strconv.Atoi(v)
	}
	if limitChildren, ok := request.Data["limits.childs"].(float64); ok {
		placeLimitsUpdate["limits.childs"] = int(limitChildren)
	} else if v, ok := request.Data["limits.childs"].(string); ok {
		placeLimitsUpdate["limits.childs"], _ = strconv.Atoi(v)
	}
	if limitSize, ok := request.Data["limits.size"].(float64); ok {
		placeLimitsUpdate["limits.size"] = int(limitSize)
	} else if v, ok := request.Data["limits.size"].(string); ok {
		placeLimitsUpdate["limits.size"], _ = strconv.Atoi(v)
	}
	if len(placeUpdate) > 0 {
		if !s.Worker().Model().Place.Update(place.ID, placeUpdate) {
			response.Error(global.ErrUnknown, []string{"place_update_error"})
			return
		}
	}
	if len(placeLimitsUpdate) > 0 {
		if !s.Worker().Model().Place.UpdateLimits(place.ID, placeLimitsUpdate) {
			response.Error(global.ErrUnknown, []string{"place_limit_update_error"})
			return
		}
	}
	response.Ok()
}

// @Command:	admin/place_list
// @Input:	filter				string			+	(grand_places | locked_places | unlocked_places | personal_places | shared_places | all)
// @Input:	keyword 				string			+
// @Input:	grand_parent_id 		string			+
// @Input: sort					string			+	(key_holders | creators | children | place_type)
// @Pagination
func (s *AdminService) listPlaces(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keyword, filter, grandParentID, sort string
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}
	if v, ok := request.Data["filter"].(string); ok {
		filter = v
	}
	if v, ok := request.Data["grand_parent_id"].(string); ok {
		if s.Worker().Model().Place.Exists(v) {
			grandParentID = v
		}
	}
	if v, ok := request.Data["sort"].(string); ok {
		sortDescending := strings.HasPrefix(v, "-")
		switch strings.ToLower(strings.Trim(v, "-")) {
		case "key_holders":
			sort = "counters.key_holders"
		case "creators":
			sort = "counters.creators"
		case "children":
			sort = "counters.childs"
		case "place_type":
			sort = "type"
		}
		if sortDescending {
			sort = fmt.Sprintf("-%s", sort)
		}
	}
	places := s.Worker().Model().Search.Places(keyword, filter, sort, grandParentID, s.Worker().Argument().GetPagination(request))
	response.OkWithData(tools.M{"places": places})
	return
}

// @Command:	admin/place_remove
// @Input:	place_id		string		*
func (s *AdminService) removePlace(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}

	if place.Counter.Children > 0 {
		response.Error(global.ErrAccess, []string{"remove_children_first"})
		return
	}

	if s.Worker().Model().Place.Remove(place.ID, requester.ID) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
		return
	}

	// Remove Place from search index
	s.Worker().Model().Search.RemovePlaceFromSearchIndex(place.ID)
}

// @Command:	admin/place_add_member
// @Input:	place_id		string		*
// @Input: account_id		string		* (comma separated)
func (s *AdminService) addPlaceMember(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	var accountIDs []string
	var ignoredAccountIDs []string
	if placeID, ok := request.Data["place_id"].(string); ok {
		place = s.Worker().Model().Place.GetByID(placeID, nil)
		if place == nil {
			response.Error(global.ErrInvalid, []string{"place_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_id"})
		return
	}
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}

	if v, ok := request.Data["account_id"].(string); ok {
		accountIDs = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}

	// no one can join personal places
	if place.IsPersonal() {
		response.Error(global.ErrAccess, []string{"personal_place"})
		return
	}

	// if placeID is not grand place and accountID is not member of the grand place return error
	grandPlace := s.Worker().Model().Place.GetByID(place.GrandParentID, nil)
	for _, accountID := range accountIDs {
		if !s.Worker().Model().Account.Exists(accountID) {
			continue
		}

		if !place.IsGrandPlace() && !grandPlace.IsMember(accountID) {
			ignoredAccountIDs = append(ignoredAccountIDs, accountID)
			continue
		}

		if place.IsMember(accountID) {
			ignoredAccountIDs = append(ignoredAccountIDs, accountID)
			continue
		}

		// if place is full
		if place.HasKeyholderLimit() {
			ignoredAccountIDs = append(ignoredAccountIDs, accountID)
			continue
		}

		s.Worker().Model().Place.AddKeyHolder(place.ID, accountID)

		// Enables notification by default
		s.Worker().Model().Account.SetPlaceNotification(accountID, place.ID, true)

		// Add the place to the added user's feed list
		s.Worker().Model().Account.AddPlaceToBookmarks(accountID, place.ID)
	}

	response.OkWithData(tools.M{
		"ignored_account_ids": ignoredAccountIDs,
	})
}

// @Command:	admin/place_promote_member
// @Input:	place_id		string	*
// @Input:	account_id		string	*
func (s *AdminService) promotePlaceMember(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var accountID string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if v, ok := request.Data["account_id"].(string); ok {
		accountID = v
	}
	if !place.IsKeyholder(accountID) {
		response.Error(global.ErrInvalid, []string{"account_id"})
		return
	}
	if place.HasCreatorLimit() {
		response.Error(global.ErrLimit, []string{"creators"})
		return
	}

	s.Worker().Model().Place.Promote(place.ID, accountID)

	// Push notification and sync messages
	s.Worker().Pusher().PlaceMemberPromoted(place, requester.ID, accountID)

	response.Ok()

}

// @Command:	admin/place_demote_member
// @Input:	place_id		string	*
// @Input:	account_id		string	*
func (s *AdminService) demotePlaceMember(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var accountID string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if v, ok := request.Data["account_id"].(string); ok {
		accountID = v
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}

	if !place.IsCreator(accountID) {
		response.Error(global.ErrInvalid, []string{"account_id"})
		return
	}

	s.Worker().Model().Place.Demote(place.ID, accountID)

	s.Worker().Pusher().PlaceMemberDemoted(place, requester.ID, accountID)

	response.Ok()
}

// @Command:	admin/place_remove_member
// @Input:	place_id		string	*
// @Input:	account_id		string	*
func (s *AdminService) removePlaceMember(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var accountID string
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	if v, ok := request.Data["account_id"].(string); ok {
		accountID = v
	}

	// If you are leaving a grand place first must check through all the sub-places and
	// check if user is allowed to leave that.
	if place.IsGrandPlace() {
		for _, pid := range s.Worker().Model().Account.GetAccessPlaceIDs(accountID) {
			if s.Worker().Model().Place.IsSubPlace(place.ID, pid) {
				subPlace := s.Worker().Model().Place.GetByID(pid, nil)
				if subPlace != nil {
					switch {
					case subPlace.IsCreator(accountID):
						s.Worker().Model().Place.RemoveCreator(pid, accountID, requester.ID)
					default:
						s.Worker().Model().Place.RemoveKeyHolder(pid, accountID, requester.ID)
					}
				}
			}
		}
	}

	// remove the user from placeID
	switch {
	case place.IsCreator(accountID):
		s.Worker().Model().Place.RemoveCreator(place.ID, accountID, requester.ID)
	case place.IsKeyholder(accountID):
		s.Worker().Model().Place.RemoveKeyHolder(place.ID, accountID, requester.ID)
	default:
		response.Error(global.ErrInvalid, []string{"account_id"})
	}

	response.Ok()
	return
}

// @Command:	admin/place_list_members
// @Input:	place_id		string		*
func (s *AdminService) listPlaceMembers(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	creatorIDs := place.CreatorIDs
	keyholderIDs := place.KeyholderIDs
	r := make([]tools.M, 0, len(creatorIDs)+len(keyholderIDs))
	iStart := 0
	iLength := global.DefaultMaxResultLimit
	iEnd := iStart + iLength
	if iEnd > len(creatorIDs) {
		iEnd = len(creatorIDs)
	}
	for {
		for _, member := range s.Worker().Model().Account.GetAccountsByIDs(creatorIDs[iStart:iEnd]) {
			r = append(r, s.Worker().Map().Account(member, true))
		}
		iStart += iLength
		iEnd = iStart + iLength
		if iStart >= len(creatorIDs) {
			break
		}
		if iEnd > len(creatorIDs) {
			iEnd = len(creatorIDs)
		}
	}

	iStart = 0
	iLength = global.DefaultMaxResultLimit
	iEnd = iStart + iLength
	if iEnd > len(keyholderIDs) {
		iEnd = len(keyholderIDs)
	}
	for {
		for _, member := range s.Worker().Model().Account.GetAccountsByIDs(keyholderIDs[iStart:iEnd]) {
			r = append(r, s.Worker().Map().Account(member, false))
		}
		iStart += iLength
		iEnd = iStart + iLength
		if iStart >= len(keyholderIDs) {
			break
		}
		if iEnd > len(keyholderIDs) {
			iEnd = len(keyholderIDs)
		}
	}

	response.OkWithData(tools.M{"accounts": r})

}

// @Command:	admin/place_set_picture
// @Input:	place_id			string	*
// @Input:	universal_id		string	*
func (s *AdminService) setPlaceProfilePicture(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}
	pic := nested.Picture{}
	if v, ok := request.Data["universal_id"].(string); ok {
		fileInfo := s.Worker().Model().File.GetByID(nested.UniversalID(v), nil)
		if fileInfo == nil {
			response.Error(global.ErrUnavailable, []string{"universal_id"})
			return
		}
		pic = fileInfo.Thumbnails
	}
	s.Worker().Model().Place.SetPicture(place.ID, pic)
	response.Ok()
	return
}

// @Command:	admin/account_register
// @Input:	uid			string	*
// @Input:	pass		string	*
// @Input:	fname		string	*
// @Input:	lname		string	*
// @Input:	gender		string	+	(m | f | o | x)
// @Input:	dob			string 	+	(YYYY-MM-DD)
// @Input:	country		string 	+	(2 character)
// @Input:	email	 	string	+
// @Input:	phone		string  *	(format example: 98912345678)
// @Input:   send_sms    bool       +
func (s *AdminService) createAccount(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	baseURL := s.Worker().Config().GetString("WEBAPP_BASE_URL")
	var uid, pass, fname, lname, gender, dob, country, email, phone string
	var passAutoGenerated, sendSms bool

	// Check License Limit
	counters := s.Worker().Model().System.GetCounters()
	maxActiveUsers := s.Worker().Model().License.Get().MaxActiveUsers
	if maxActiveUsers != 0 && counters[global.SystemCountersEnabledAccounts] >= maxActiveUsers {
		response.Error(global.ErrLimit, []string{"license_users_limit"})
		return
	}

	if v, ok := request.Data["uid"].(string); ok {
		uid = strings.ToLower(strings.Trim(v, " "))
	}
	if v, ok := request.Data["pass"].(string); ok && len(v) > 0 {
		pass = v
	} else {
		passAutoGenerated = true
		md5Hasher := md5.New()
		md5Hasher.Write([]byte(nested.RandomPassword(6)))
		pass = hex.EncodeToString(md5Hasher.Sum(nil))
	}
	if v, ok := request.Data["fname"].(string); ok {
		fname = strings.Trim(v, " ")
	}
	if v, ok := request.Data["lname"].(string); ok {
		lname = strings.Trim(v, " ")
	}
	if v, ok := request.Data["gender"].(string); ok {
		gender = v
	}
	if v, ok := request.Data["dob"].(string); ok {
		if _, err := time.Parse("2006-01-02", v); err == nil {
			dob = v
		}
	}
	if v, ok := request.Data["country"].(string); ok {
		country = v
	}
	if v, ok := request.Data["email"].(string); ok {
		email = strings.Trim(v, " ")
		if b, err := regexp.MatchString(global.DefaultRegexEmail, email); err != nil || !b {
			response.Error(global.ErrInvalid, []string{"email"})
			return
		}

	}
	if v, ok := request.Data["phone"].(string); ok {
		phone = v
		phone = strings.TrimLeft(phone, "+0")
	}
	if v, ok := request.Data["send_sms"].(bool); ok {
		sendSms = v
	}

	// check if username match the regular expression
	if matched, err := regexp.MatchString(global.DefaultRegexAccountID, uid); err != nil {
		response.Error(global.ErrUnknown, []string{err.Error()})
		return
	} else if !matched {
		response.Error(global.ErrInvalid, []string{"uid"})
		return
	}

	// check if username is not taken already
	if s.Worker().Model().Account.Exists(uid) || s.Worker().Model().Place.Exists(uid) {
		response.Error(global.ErrDuplicate, []string{"uid"})
		return
	}

	// check if phone is not taken already
	systemConstants := s.Worker().Model().System.GetStringConstants()
	if phone != systemConstants[global.SystemConstantsMagicNumber] && s.Worker().Model().Account.PhoneExists(phone) {
		response.Error(global.ErrDuplicate, []string{"phone"})
		return
	}

	// check if email is not taken already
	if email != "" && s.Worker().Model().Account.EmailExists(email) {
		response.Error(global.ErrDuplicate, []string{"email"})
		return
	}

	// check that fname and lname cannot both be empty text
	if fname == "" && lname == "" {
		response.Error(global.ErrInvalid, []string{"fname", "lname"})
		return
	}

	if !s.Worker().Model().Account.CreateUser(uid, pass, phone, country, fname, lname, email, dob, gender) {
		response.Error(global.ErrUnknown, []string{""})
		return
	}

	// create personal place for the new account
	pcr := nested.PlaceCreateRequest{
		ID:            uid,
		GrandParentID: uid,
		AccountID:     uid,
		Name:          fmt.Sprintf("%s %s", fname, lname),
		Description:   fmt.Sprintf("Personal place for %s", uid),
	}
	pcr.Policy.AddMember = nested.PlacePolicyNoOne
	pcr.Policy.AddPlace = nested.PlacePolicyCreators
	pcr.Policy.AddPost = nested.PlacePolicyEveryone
	pcr.Privacy.Locked = true
	pcr.Privacy.Receptive = nested.PlaceReceptiveExternal
	pcr.Privacy.Search = true
	s.Worker().Model().Place.CreatePersonalPlace(pcr)

	// add the new user to his/her new personal place
	s.Worker().Model().Place.AddKeyHolder(pcr.ID, pcr.AccountID)
	s.Worker().Model().Place.Promote(pcr.ID, pcr.AccountID)

	// add the personal place to his/her favorite place
	s.Worker().Model().Account.AddPlaceToBookmarks(pcr.AccountID, pcr.ID)

	// set notification on for place
	s.Worker().Model().Account.SetPlaceNotification(pcr.AccountID, pcr.ID, true)

	// add user's account & place to search index
	s.Worker().Model().Search.AddPlaceToSearchIndex(uid, fmt.Sprintf("%s %s", fname, lname), pcr.Picture)

	if placeIDs := s.Worker().Model().Place.GetDefaultPlaces(); len(placeIDs) > 0 {
		for _, placeID := range placeIDs {
			place := s.Worker().Model().Place.GetByID(placeID, nil)
			if place == nil {
				continue
			}
			grandPlace := place.GetGrandParent()
			// if user is already a member of the place then skip
			if place.IsMember(uid) {
				continue
			}
			// if user is not a keyHolder or Creator of place grandPlace, then make him to be
			if !grandPlace.IsMember(uid) {
				s.Worker().Model().Place.AddKeyHolder(grandPlace.ID, uid)
				// Enables notification by default
				s.Worker().Model().Account.SetPlaceNotification(uid, grandPlace.ID, true)

				// Add the place to the added user's feed list
				s.Worker().Model().Account.AddPlaceToBookmarks(uid, grandPlace.ID)

				// Handle push notifications and activities
				s.Worker().Pusher().PlaceJoined(grandPlace, requester.ID, uid)
			}
			// if place is a grandPlace then skip going deeper
			if place.IsGrandPlace() {
				continue
			}
			//if !place.HasKeyholderLimit() {
			s.Worker().Model().Place.AddKeyHolder(place.ID, uid)

			// Enables notification by default
			s.Worker().Model().Account.SetPlaceNotification(uid, place.ID, true)

			// Add the place to the added user's feed list
			s.Worker().Model().Account.AddPlaceToBookmarks(uid, place.ID)

			// Handle push notifications and activities
			s.Worker().Pusher().PlaceJoined(place, requester.ID, uid)

			place.Counter.Keyholders += 1

		}
	}

	// prepare welcome message and invitations
	go s.prepareWelcome(uid)

	// ADP Sms Panel Initialization
	adp := NewADP(
		s.Worker().Config().GetString("ADP_USERNAME"),
		s.Worker().Config().GetString("ADP_PASSWORD"),
		s.Worker().Config().GetString("ADP_MESSAGE_URL"),
	)
	// Force user to change his/her password at next login
	s.Worker().Model().Account.ForcePasswordChange(uid, true)

	if passAutoGenerated {
		// Create a Login Token
		loginToken := s.Worker().Model().Token.CreateLoginToken(uid)

		// Send SMS
		go func() {
			if len(baseURL) > 0 {
				if _, err := adp.SendSms(
					phone,
					fmt.Sprintf("Welcome to Nested, login to your account click on: %s/t/?%s",
						s.Worker().Config().GetString("WEBAPP_BASE_URL"),
						loginToken,
					),
				); err != nil {
					log.Println("Send SMS Error:", err.Error())
				}
			}
		}()

		response.OkWithData(tools.M{
			"token": loginToken,
		})
	} else {
		if sendSms && len(baseURL) > 0 {
			// Create a Login Token
			loginToken := s.Worker().Model().Token.CreateLoginToken(uid)

			// Send SMS
			go func() {
				adp.SendSms(
					phone,
					fmt.Sprintf("Welcome to Nested, login to your account click on: %s/t/?%s",
						// uid,
						s.Worker().Config().GetString("WEBAPP_BASE_URL"),
						loginToken,
					),
				)
			}()

		}
		response.Ok()
	}
	return
}

// @Command: admin/account_list
// @Input:	keyword		string	+
// @Input:	filter		string	+	(users_enabled | users_disabled | users | devices | all)
// @Input:	sort 		string	+	(joined_on | birthday | user_id | email)
// @Pagination
func (s *AdminService) listAccounts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keyword, filter, sort string
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}
	if v, ok := request.Data["filter"].(string); ok {
		filter = v
	}
	if v, ok := request.Data["sort"].(string); ok {
		sortDescending := strings.HasPrefix(v, "-")
		switch strings.ToLower(strings.Trim(v, "-")) {
		case "joined_on":
			sort = "joined_on"
		case "birthday":
			sort = "dob"
		case "user_id":
			sort = "_id"
		case "email":
			sort = "email"
		}
		if sortDescending {
			sort = fmt.Sprintf("-%s", sort)
		}
	}
	accounts := s.Worker().Model().Search.Accounts(keyword, filter, sort, s.Worker().Argument().GetPagination(request))
	r := make([]tools.M, 0, len(accounts))
	for _, account := range accounts {
		r = append(r, s.Worker().Map().Account(account, true))
	}
	response.OkWithData(tools.M{"accounts": r})
	return
}

// @Command:	admin/account_disable
// @Input:	account_id		string	*
func (s *AdminService) disableAccount(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var account *nested.Account
	if accountID, ok := request.Data["account_id"].(string); ok {
		account = s.Worker().Model().Account.GetByID(accountID, nil)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}
	s.Worker().Model().Account.Disable(account.ID)
	response.Ok()
	return
}

// @Command:	admin/account_enable
// @Input:	account_id		string	*
func (s *AdminService) enableAccount(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var account *nested.Account

	// Check License Limit
	counters := s.Worker().Model().System.GetCounters()
	maxActiveUsers := s.Worker().Model().License.Get().MaxActiveUsers
	if maxActiveUsers != 0 && counters[global.SystemCountersEnabledAccounts] >= maxActiveUsers {
		response.Error(global.ErrLimit, []string{"license_users_limit"})
		return
	}

	if accountID, ok := request.Data["account_id"].(string); ok {
		account = s.Worker().Model().Account.GetByID(accountID, nil)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}
	s.Worker().Model().Account.Enable(account.ID)
	response.Ok()
	return
}

// @Command:	admin/account_set_pass
// @Input:	account_id		string	*
// @Input:	new_pass			string	*
func (s *AdminService) setAccountPassword(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var newPass string
	var account *nested.Account
	if accountID, ok := request.Data["account_id"].(string); ok {
		account = s.Worker().Model().Account.GetByID(accountID, nil)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}

	if str, ok := request.Data["new_pass"].(string); ok {
		newPass = str
	} else {
		response.Error(global.ErrInvalid, []string{"new_pass"})
	}

	if s.Worker().Model().Account.SetPassword(account.ID, newPass) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
}

// @Command:	admin/account_list_places
// @Input:	account_id		string	*
func (s *AdminService) listPlacesOfAccount(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var account *nested.Account
	if accountID, ok := request.Data["account_id"].(string); ok {
		account = s.Worker().Model().Account.GetByID(accountID, nil)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}

	placeIDs := account.AccessPlaceIDs
	r := make([]tools.M, 0, len(placeIDs))
	iStart := 0
	iLength := global.DefaultMaxResultLimit
	iEnd := iStart + iLength
	if iEnd > len(placeIDs) {
		iEnd = len(placeIDs)
	}
	for {
		for _, place := range s.Worker().Model().Place.GetPlacesByIDs(placeIDs[iStart:iEnd]) {
			r = append(r, s.Worker().Map().Place(requester, place, place.GetAccess(account.ID)))
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

// @Command:	admin/account_update
// @Input:	account_id:					string		*
// @Input:	fname:						string		+
// @Input:	lname:						string		+
// @Input:	gender:						string		+	(m | f)
// @Input:	dob:						    string		+	(YYYY-MM-DD)
// @Input:	email:						string		+
// @Input:	phone: 						string		+
// @Input:	searchable:					bool		    +
// @Input:	change_profile:				bool		    +
// @Input:	change_picture:				bool		    +
// @Input:	force_password				bool		    +
// @Input:	limits.grand_places		    int		    +
// @Input:	authority.label_editor		bool		    +
func (s *AdminService) updateAccount(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var account *nested.Account
	accountUpdateRequest := nested.AccountUpdateRequest{}
	accountLimitsUpdateRequest := nested.MI{}
	placeUpdateRequest := tools.M{}

	if accountID, ok := request.Data["account_id"].(string); ok {
		account = s.Worker().Model().Account.GetByID(accountID, nil)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}

	if fname, ok := request.Data["fname"].(string); ok && fname != "" {
		accountUpdateRequest.FirstName = fname
		if len(fname) > global.DefaultMaxAccountName {
			accountUpdateRequest.FirstName = fname[:global.DefaultMaxAccountName]
		}
	}
	if lname, ok := request.Data["lname"].(string); ok && lname != "" {
		accountUpdateRequest.LastName = lname
		if len(lname) > global.DefaultMaxAccountName {
			accountUpdateRequest.LastName = lname[:global.DefaultMaxAccountName]
		}
	}
	if gender, ok := request.Data["gender"].(string); ok && gender != "" {
		switch gender {
		case "m", "male", "man", "boy":
			gender = "m"
		case "f", "female", "woman", "girl":
			gender = "f"
		case "o", "other":
			gender = "o"
		default:
			gender = "x"
		}
		accountUpdateRequest.Gender = gender
	}
	if dob, ok := request.Data["dob"].(string); ok {
		if _, err := time.Parse("2006-01-02", dob); err == nil {
			accountUpdateRequest.DateOfBirth = dob
		}
	}
	if email, ok := request.Data["email"].(string); ok {
		email = strings.Trim(email, " ")
		if b, err := regexp.MatchString(global.DefaultRegexEmail, email); err == nil && b {
			accountUpdateRequest.Email = email
		}
	}
	if phone, ok := request.Data["phone"].(string); ok {
		phone = strings.Trim(phone, " ")
		if b, _ := regexp.MatchString(`^[\d]{12,15}$`, phone); b {
			s.Worker().Model().Account.SetPhone(account.ID, phone)
		}

	}
	if searchable, ok := request.Data["searchable"].(bool); ok {
		if searchable {
			s.Worker().Model().Search.AddPlaceToSearchIndex(account.ID, fmt.Sprintf("%s %s", account.FirstName, account.LastName), account.Picture)
			placeUpdateRequest["privacy.search"] = true
		} else {
			s.Worker().Model().Search.RemovePlaceFromSearchIndex(account.ID)
			placeUpdateRequest["privacy.search"] = false
		}
		s.Worker().Model().Account.SetPrivacy(account.ID, "searchable", searchable)
	}
	if changeProfile, ok := request.Data["change_profile"].(bool); ok {
		s.Worker().Model().Account.SetPrivacy(account.ID, "change_profile", changeProfile)
	}
	if changePicture, ok := request.Data["change_picture"].(bool); ok {
		s.Worker().Model().Account.SetPrivacy(account.ID, "change_picture", changePicture)
	}
	if forcePasswordChange, ok := request.Data["force_password"].(bool); ok {
		s.Worker().Model().Account.ForcePasswordChange(account.ID, forcePasswordChange)
	}
	if limitGrandPlace, ok := request.Data["limits.grand_places"].(float64); ok {
		accountLimitsUpdateRequest["limits.grand_places"] = int(limitGrandPlace)
	}
	if authorityLabelEditor, ok := request.Data["authority.label_editor"].(bool); ok {
		account.Authority.LabelEditor = authorityLabelEditor
	}
	// Update accountID and its limits
	s.Worker().Model().Account.Update(account.ID, accountUpdateRequest)
	s.Worker().Model().Account.UpdateLimits(account.ID, accountLimitsUpdateRequest)
	s.Worker().Model().Account.UpdateAuthority(account.ID, account.Authority)

	// Update the personal place of the accountID
	s.Worker().Model().Place.Update(account.ID, placeUpdateRequest)

	if account.Privacy.Searchable && (accountUpdateRequest.FirstName != "" || accountUpdateRequest.LastName != "") {
		s.Worker().Model().Search.AddPlaceToSearchIndex(account.ID, fmt.Sprintf("%s %s", account.FirstName, account.LastName), account.Picture)
	}

	response.OkWithData(tools.M{
		"applied_limit_updates": accountLimitsUpdateRequest,
		"applied_updates":       accountUpdateRequest,
	})
	return
}

// @Command: admin/account_set_picture
// @Input:	account_id		string		*
// @Input:	universal_id		string		*
func (s *AdminService) setAccountProfilePicture(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var account *nested.Account
	var uni_id nested.UniversalID
	if accountID, ok := request.Data["account_id"].(string); ok {
		account = s.Worker().Model().Account.GetByID(accountID, nil)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}

	if v, ok := request.Data["universal_id"].(string); ok {
		uni_id = nested.UniversalID(v)
		if !s.Worker().Model().File.Exists(uni_id) {
			response.Error(global.ErrUnavailable, []string{"universal_id"})
			return
		}
	}
	f := s.Worker().Model().File.GetByID(uni_id, nil)
	s.Worker().Model().Account.SetPicture(account.ID, f.Thumbnails)
	response.Ok()
	return
}

// @Command: admin/account_remove_picture
// @Input:	account_id		string		*
func (s *AdminService) removeAccountProfilePicture(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var account *nested.Account
	if accountID, ok := request.Data["account_id"].(string); ok {
		account = s.Worker().Model().Account.GetByID(accountID, nil)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}

	pic := nested.Picture{}
	s.Worker().Model().Account.SetPicture(account.ID, pic)
	response.Ok()
	return
}

func (s *AdminService) prepareWelcome(accountID string) {
	account := s.Worker().Model().Account.GetByID(accountID, nil)
	var fillData struct {
		AccountFirstName string
		AccountLastName  string
	}
	fillData.AccountFirstName = account.FirstName
	fillData.AccountLastName = account.LastName

	msgTemplates := s.Worker().Model().System.GetMessageTemplates()
	var body bytes.Buffer
	t, _ := template.New("Welcome").Parse(msgTemplates["WELCOME_MSG"].Body)
	t.Execute(&body, fillData)

	pcr := nested.PostCreateRequest{
		SenderID:    "nested",
		Subject:     msgTemplates["WELCOME_MSG"].Subject,
		Body:        body.String(),
		ContentType: nested.ContentTypeTextHtml,
		PlaceIDs:    []string{accountID},
		SystemData: nested.PostSystemData{
			NoComment: true,
		},
	}

	s.Worker().Model().Post.AddPost(pcr)
}

// @Command:	admin/create_post_for_all_accounts
// @Input:  subject			string	+
// @Input:  attaches			string 	+	(comma separated)
// @Input:  content_type		string	+	(text/plain | text/html)
// @Input:  iframe_url         string +
func (s *AdminService) createPostForAllAccounts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var targets []string
	var attachments []string
	var subject, body, content_type, iframeUrl, filter string
	var labels []nested.Label
	if v, ok := request.Data["filter"].(string); ok {
		filter = v
	}
	if v, ok := request.Data["label_id"].(string); ok {
		labelIDs := strings.SplitN(v, ",", global.DefaultPostMaxLabels)
		labels = s.Worker().Model().Label.GetByIDs(labelIDs)
	} else {
		labels = []nested.Label{}
	}
	if v, ok := request.Data["attaches"].(string); ok && v != "" {
		attachments = strings.SplitN(v, ",", global.DefaultPostMaxAttachments)
	} else {
		attachments = []string{}
	}
	if v, ok := request.Data["content_type"].(string); ok {
		switch v {
		case nested.ContentTypeTextHtml, nested.ContentTypeTextPlain:
			content_type = v
		default:
			content_type = nested.ContentTypeTextPlain
		}
	} else {
		content_type = nested.ContentTypeTextPlain
	}
	if v, ok := request.Data["subject"].(string); ok {
		subject = v
	}
	if v, ok := request.Data["body"].(string); ok {
		body = v
	}
	if v, ok := request.Data["iframe_url"].(string); ok {
		iframeUrl = v
	}

	if "" == strings.Trim(subject, " ") && "" == strings.Trim(body, " ") && len(attachments) == 0 {
		response.Error(global.ErrIncomplete, []string{"subject", "body"})
		return
	}
	targets = s.Worker().Model().Search.AccountIDs(filter)

	if len(targets) == 0 {
		response.Error(global.ErrInvalid, []string{"targets"})
		return
	}

	for i, v := range attachments {
		if v == "" || !s.Worker().Model().File.Exists(nested.UniversalID(v)) {
			if len(attachments) > 1 {
				attachments[i] = attachments[len(attachments)-1]
				attachments = attachments[:len(attachments)-1]
			} else {
				attachments = attachments[:0]
				break
			}
		}
	}

	pcr := nested.PostCreateRequest{
		PlaceIDs:    targets,
		ContentType: content_type,
		SenderID:    requester.ID,
		SystemData: nested.PostSystemData{
			NoComment: true,
		},
	}

	// Make attachments unique and add them to PostCreateRequest
	mapAttachments := tools.MB{}
	for _, attachID := range attachments {
		mapAttachments[attachID] = true
	}

	for attachID := range mapAttachments {
		pcr.AttachmentIDs = append(pcr.AttachmentIDs, nested.UniversalID(attachID))
	}

	// Set Body for PostCreateRequest
	pcr.Body = body
	pcr.IFrameUrl = iframeUrl

	// check if subject does not exceed the limit
	if len(subject) > 255 {
		pcr.Subject = string(subject[:255])
	} else {
		pcr.Subject = subject
	}

	post := s.Worker().Model().Post.AddPost(pcr)
	if post == nil {
		response.Error(global.ErrUnknown, []string{})
		return
	}

	for _, label := range labels {
		if label.Public || label.IsMember(requester.ID) {
			post.AddLabel(requester.ID, label.ID)
		}
	}

	s.Worker().Pusher().PostAdded(post)

	response.OkWithData(tools.M{
		"post_id": post.ID,
	})
}

// @Command:	admin/default_places_add
// @Input:  	place_ids			string	+
func (s *AdminService) addDefaultPlaces(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var places []string
	ids := s.Worker().Model().Place.GetDefaultPlaces()
	if v, ok := request.Data["place_ids"].(string); ok {
		placeIDs := strings.SplitN(v, ",", -1)
		for _, id := range placeIDs {
			if place := s.Worker().Model().Place.GetByID(id, nil); place != nil {
				// no one can join personal places
				if place.IsPersonal() {
					response.Error(global.ErrAccess, []string{"personal_place"})
					return
				}
				exist := false
				for _, pid := range ids {
					if pid == id {
						exist = true
						continue
					}
				}
				if exist == false {
					places = append(places, id)
				}
			}
		}
	} else {
		response.Error(global.ErrInvalid, []string{"place_ids"})
		return
	}
	if len(places) < 1 {
		response.Error(global.ErrInvalid, []string{"place_ids"})
		return
	}
	if success := s.Worker().Model().Place.AddDefaultPlaces(places); !success {
		response.Error(global.ErrUnknown, []string{""})
		return
	}
	response.OkWithData(tools.M{})
}

// @Command:	admin/default_places_get
func (s *AdminService) getDefaultPlaces(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	pg := s.Worker().Argument().GetPagination(request)
	if placeIDs, total := s.Worker().Model().Place.GetDefaultPlacesWithPagination(pg); placeIDs == nil {
		response.Error(global.ErrUnknown, []string{""})
		return
	} else {
		places := s.Worker().Model().Place.GetPlacesByIDs(placeIDs)
		response.OkWithData(tools.M{"places": places, "total": total})
	}
}

// @Command:	admin/default_places_remove
// @Input:  	place_ids			string	+
func (s *AdminService) removeDefaultPlaces(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var placeIDs []string
	if v, ok := request.Data["place_ids"].(string); ok {
		ids := strings.SplitN(v, ",", -1)
		for _, id := range ids {
			if place := s.Worker().Model().Place.GetByID(id, nil); place != nil {
				placeIDs = append(placeIDs, id)
			}
		}
	} else {
		response.Error(global.ErrInvalid, []string{"places"})
		return
	}
	if success := s.Worker().Model().Place.RemoveDefaultPlaces(placeIDs); !success {
		response.Error(global.ErrUnknown, []string{""})
		return
	} else {
		response.Ok()
	}
}

// @Command:	admin/default_places_set_users
// @Input:  	account_ids			string	+
func (s *AdminService) defaultPlacesSetUsers(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var accountIDs []string
	if v, ok := request.Data["account_ids"].(string); ok {
		ids := strings.SplitN(v, ",", -1)
		for _, id := range ids {
			if account := s.Worker().Model().Account.GetByID(id, nil); account != nil {
				accountIDs = append(accountIDs, id)
			}
		}
	} else {
		response.Error(global.ErrInvalid, []string{"account_ids"})
		return
	}
	if placeIDs := s.Worker().Model().Place.GetDefaultPlaces(); len(placeIDs) < 1 {
		response.Error(global.ErrInvalid, []string{"place_ids"})
		return
	} else {
		log.Println("defaultPlacesSetUsers::placeIDs ", placeIDs)
		for _, placeID := range placeIDs {
			for _, uid := range accountIDs {
				place := s.Worker().Model().Place.GetByID(placeID, nil)
				if place == nil {
					continue
				}
				grandPlace := place.GetGrandParent()
				// if user is already a member of the place then skip
				if place.IsMember(uid) {
					continue
				}
				// if user is not a keyHolder or Creator of place grandPlace, then make him to be
				if !grandPlace.IsMember(uid) {
					s.Worker().Model().Place.AddKeyHolder(grandPlace.ID, uid)
					// Enables notification by default
					s.Worker().Model().Account.SetPlaceNotification(uid, grandPlace.ID, true)

					// Add the place to the added user's feed list
					s.Worker().Model().Account.AddPlaceToBookmarks(uid, grandPlace.ID)

					// Handle push notifications and activities
					s.Worker().Pusher().PlaceJoined(grandPlace, requester.ID, uid)
				}
				// if place is a grandPlace then skip going deeper
				if place.IsGrandPlace() {
					continue
				}
				//if !place.HasKeyholderLimit() {
				s.Worker().Model().Place.AddKeyHolder(place.ID, uid)

				// Enables notification by default
				s.Worker().Model().Account.SetPlaceNotification(uid, place.ID, true)

				// Add the place to the added user's feed list
				s.Worker().Model().Account.AddPlaceToBookmarks(uid, place.ID)

				// Handle push notifications and activities
				s.Worker().Pusher().PlaceJoined(place, requester.ID, uid)

				place.Counter.Keyholders += 1
			}
		}
		response.Ok()
	}
}
