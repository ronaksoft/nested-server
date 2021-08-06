package nestedServicePost

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/config"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"strings"

	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
	"github.com/globalsign/mgo/bson"
)

// @Command:	post/add_label
// @Input:	post_id			string	*
// @Input:	label_id			string	*
func (s *PostService) addLabelToPost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var label *nested.Label
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	if label = s.Worker().Argument().GetLabel(request, response); label == nil {
		return
	}

	// If label is not public and user is not member of the label then he/she has no permission
	// to add this label to posts
	if !label.Public && !label.IsMember(requester.ID) {
		response.Error(global.ErrAccess, []string{"not_member_of_label"})
		return
	}

	// If user has no access to the post, then he/she cannot add label to post
	if !post.HasAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{"no_access_to_post"})
		return
	}

	if post.Counters.Labels >= global.DefaultPostMaxLabels {
		response.Error(global.ErrLimit, []string{"number_of_labels"})
		return
	}

	if post.AddLabel(requester.ID, label.ID) {

		// handle push messages (activity)
		go s.Worker().Pusher().PostLabelAdded(post, label)

		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{""})
	}
	return
}

// @Command:	post/add_comment
// @Input:	post_id			string	*
// @Input:	txt				string	*
// @Input:	attachment_id	string	*
func (s *PostService) addComment(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
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

	if !post.HasAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{})
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
	c := _Model.Post.AddComment(post.ID, requester.ID, txt, attachmentID)
	if c == nil {
		response.Error(global.ErrUnknown, []string{"internal_error"})
	}

	// mark post as read
	if post.SenderID != requester.ID {
		_Model.Post.MarkAsRead(post.ID, requester.ID)
	}

	// handle push messages (notification and activity)
	go s.Worker().Pusher().PostCommentAdded(post, c)

	response.OkWithData(tools.M{"comment_id": c.ID})

	return
}

// @Command:	post/attach_place
// @Input:	post_id			string	*
// @Input:	place_id			string	*	(comma separated)
func (s *PostService) attachPlace(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	var placeIDs []string
	if v, ok := request.Data["place_id"].(string); ok {
		placeIDs = strings.Split(v, ",")
	} else {
		response.Error(global.ErrIncomplete, []string{"old_place_id"})
		return
	}
	if len(post.PlaceIDs)+len(placeIDs) > global.DefaultPostMaxTargets {
		response.Error(global.ErrLimit, []string{"targets"})
		return
	}

	// User must be sender of the post and have at-least READ ACCESS to the post
	if !post.HasAccess(requester.ID) || post.SenderID != requester.ID {
		response.Error(global.ErrAccess, []string{})
		return
	}

	attachedPlaceIDs := make([]string, 0, len(placeIDs))
	notAttachedPlaceIDs := make([]string, 0, len(placeIDs))
	for _, placeID := range placeIDs {
		// Post must already be in oldPlaceID and must not be in newPlaceID
		if post.IsInPlace(placeID) {
			notAttachedPlaceIDs = append(notAttachedPlaceIDs, placeID)
			continue
		}
		// User must have at least WRITE ACCESS to the new place
		place := _Model.Place.GetByID(placeID, nil)
		if place == nil {
			notAttachedPlaceIDs = append(notAttachedPlaceIDs, placeID)
			continue
		}
		access := place.GetAccess(requester.ID)
		if !access[nested.PlaceAccessWritePost] {
			notAttachedPlaceIDs = append(notAttachedPlaceIDs, placeID)
			continue
		}
		if _Model.Post.AttachPlace(post.ID, placeID, requester.ID) {
			attachedPlaceIDs = append(attachedPlaceIDs, placeID)
		} else {
			notAttachedPlaceIDs = append(notAttachedPlaceIDs, placeID)
		}
	}

	// Send Push Notifications
	s.Worker().Pusher().PostAttached(post, attachedPlaceIDs)

	response.OkWithData(tools.M{
		"attached":     attachedPlaceIDs,
		"not_attached": notAttachedPlaceIDs,
	})
}

// @Command:	post/add
// @Input:	subject			string	+
// @Input:	targets			string 	+	(comma separated)
// @Input:	attaches			string 	+	(comma separated)
// @Input:  label_id            string + (comma separated)
// @Input:	content_type		string	+	(text/plain | text/html)
// @Input:	reply_to			string 	+	(post_id)
// @Input:	forward_from		string 	+	(post_id)
// @Input:  body                string  *
// @Input:	no_comment		bool		+
func (s *PostService) createPost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var targets []string
	var attachments []string
	var subject, body, contentType, iframeUrl string
	var replyTo, forwardFrom bson.ObjectId
	var noComment bool
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
		labels = _Model.Label.GetByIDs(labelIDs)
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
			contentType = v
		default:
			contentType = nested.ContentTypeTextPlain
		}
	} else {
		contentType = nested.ContentTypeTextPlain
	}
	if v, ok := request.Data["subject"].(string); ok {
		subject = v
	}
	if v, ok := request.Data["body"].(string); ok {
		body = v
	}
	if v, ok := request.Data["reply_to"].(string); ok {
		if bson.IsObjectIdHex(v) {
			replyTo = bson.ObjectIdHex(v)
		}
	}
	if v, ok := request.Data["forward_from"].(string); ok {
		if bson.IsObjectIdHex(v) {
			forwardFrom = bson.ObjectIdHex(v)
		}
	}
	if v, ok := request.Data["no_comment"].(bool); ok {
		noComment = v
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
			domains := strings.Split(config.GetString(config.Domains), ",")
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
	hasReadAccess := false
	noWriteAccessPlaces := make([]string, 0, global.DefaultPostMaxTargets)
	notValidPlaces := make([]string, 0, global.DefaultPostMaxTargets)
	for k := range mPlaces {
		place := _Model.Place.GetByID(k, nil)
		if place == nil {
			notValidPlaces = append(notValidPlaces, k)
			delete(mPlaces, k)
			continue
		}
		// Check read access for each target place
		if !hasReadAccess && place.HasReadAccess(requester.ID) {
			hasReadAccess = true
		}
		// Check write access for each target place
		if !place.HasWriteAccess(requester.ID) {
			noWriteAccessPlaces = append(noWriteAccessPlaces, k)
			delete(mPlaces, k)
		}
	}

	if len(mPlaces) == 0 && len(mEmails) == 0 {
		response.Error(global.ErrInvalid, []string{"targets"})
		return
	}

	// If user has no read access to any of the target places then add his/her personal place to target places
	if !hasReadAccess {
		mPlaces[requester.ID] = true
	}
	for i, v := range attachments {
		if v == "" || !_Model.File.Exists(nested.UniversalID(v)) {
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
		ReplyTo:     replyTo,
		ForwardFrom: forwardFrom,
		ContentType: contentType,
		SenderID:    requester.ID,
		SystemData: nested.PostSystemData{
			NoComment: noComment,
		},
		IFrameUrl: iframeUrl,
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
	// check if subject does not exceed the limit
	if len(subject) > 255 {
		pcr.Subject = string(subject[:255])
	} else {
		pcr.Subject = subject
	}

	post := _Model.Post.AddPost(pcr)
	if post == nil {
		response.Error(global.ErrUnknown, []string{})
		return
	}

	for _, label := range labels {
		if label.Public || label.IsMember(requester.ID) {
			post.AddLabel(requester.ID, label.ID)
		}
	}

	// Push Notification and syncs
	s.Worker().Pusher().PostAdded(post)

	// Send Emails
	if len(emails) > 0 {
		mailReq := api.MailRequest{}
		if requester.Mail.Active {
			mailReq.Host = requester.Mail.OutgoingSMTPHost
			mailReq.Port = requester.Mail.OutgoingSMTPPort
			mailReq.Username = requester.Mail.OutgoingSMTPUser
			mailReq.Password = nested.Decrypt(nested.EMAIL_ENCRYPT_KEY, requester.Mail.OutgoingSMTPPass)
			mailReq.PostID = post.ID
		} else {
			mailReq.Host = ""
			mailReq.PostID = post.ID
		}
		s.Worker().Mailer().SendRequest(mailReq)
	}

	// Remove places from connection list if user no longer has access to write to it.
	if len(noWriteAccessPlaces) != 0 {
		_Model.Account.RemovePlaceConnection(requester.ID, noWriteAccessPlaces)
	}
	response.OkWithData(tools.M{
		"post_id":          post.ID,
		"no_permit_places": noWriteAccessPlaces,
		"invalid_places":   notValidPlaces,
	})

}

// @Command:	post/get
// @Input:	post_id			string	*
// @Input:	mark_as_read		bool		+
func (s *PostService) getPost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	var markAsRead bool
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	if v, ok := request.Data["mark_as_read"].(bool); ok {
		markAsRead = v
	}
	if !post.HasAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{})
		return
	}

	// mark post as read if asked so
	if markAsRead {
		post.MarkAsRead(requester.ID)
	}
	response.OkWithData(s.Worker().Map().Post(requester, *post, false))
}

// @Command: post/edit
// @Input: post_id          string *
// @Input: subject          string *
// @Input: body             string *
func (s *PostService) editPost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var subject, body string
	var post *nested.Post

	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	if v, ok := request.Data["subject"].(string); ok {
		subject = v
	}
	if v, ok := request.Data["body"].(string); ok {
		body = v
	}

	// check if subject does not exceed the limit
	if len(subject) > 255 {
		subject = string(subject[:255])
	}

	if post.SenderID == requester.ID && nested.Timestamp() < post.Timestamp+global.DefaultPostRetractTime {
		if post.Update(subject, body) {
			s.Worker().Pusher().PostEdited(post)
			response.Ok()
		} else {
			response.Error(global.ErrUnknown, []string{})
		}
	} else {
		response.Error(global.ErrAccess, []string{})
	}

}

// @Command:	post/get_many
// @Input:	post_id			string	*	(comma separated)
func (s *PostService) getManyPosts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	postIDs := make([]bson.ObjectId, 0, global.DefaultMaxResultLimit)
	noAccessPostIDs := make([]bson.ObjectId, 0)
	if v, ok := request.Data["post_id"].(string); ok {
		for _, pid := range strings.SplitN(v, ",", global.DefaultMaxResultLimit) {
			if bson.IsObjectIdHex(pid) {
				postID := bson.ObjectIdHex(pid)
				if _Model.Post.HasAccess(postID, requester.ID) {
					postIDs = append(postIDs, postID)
				} else {
					noAccessPostIDs = append(noAccessPostIDs, postID)
				}
			}
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"post_id"})
		return
	}
	posts := _Model.Post.GetPostsByIDs(postIDs)
	r := make([]tools.M, 0, len(posts))
	for _, post := range posts {
		r = append(r, s.Worker().Map().Post(requester, post, false))
	}
	response.OkWithData(tools.M{
		"posts":     r,
		"no_access": noAccessPostIDs,
	})

}

// @Command:	post/get_chain
// @Input:	post_id			string	*
func (s *PostService) getPostChain(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	limit := 10
	r := make([]tools.M, 0, limit)
	for limit > 0 {
		var postID bson.ObjectId
		if post == nil {
			break
		}
		if post.HasAccess(requester.ID) {
			r = append(r, s.Worker().Map().Post(requester, *post, false))
		}
		if post.ReplyTo.Valid() {
			postID = post.ReplyTo
		} else if post.ForwardFrom.Valid() {
			postID = post.ForwardFrom
		} else {
			break
		}
		post = _Model.Post.GetPostByID(postID)
		limit--
	}
	response.OkWithData(tools.M{
		"posts": r,
	})
}

// @Command:	post/get_counters
// @Input:	post_id			string	*
func (s *PostService) getPostCounters(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	response.OkWithData(tools.M{"counters": post.Counters})
	return
}

// @Command:	post/get_activities
// @Input:	post_id				string	    *
// @Input:	details				bool		+
func (s *PostService) getPostActivities(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	var details bool
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}

	if v, ok := request.Data["details"].(bool); ok {
		details = v
	}

	if !post.HasAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{""})
		return
	}

	pg := s.Worker().Argument().GetPagination(request)
	ta := s.Worker().Model().PostActivity.GetActivitiesByPostID(post.ID, pg, []global.PostAction{})
	d := make([]tools.M, 0, pg.GetLimit())
	for _, v := range ta {
		d = append(d, s.Worker().Map().PostActivity(requester, v, details))
	}
	response.OkWithData(tools.M{
		"skip":       pg.GetSkip(),
		"limit":      pg.GetLimit(),
		"activities": d,
	})
}

// @Command:	post/get_comments
// @Input:	post_id			string	*
// @Pagination
func (s *PostService) getCommentsByPost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}

	// check if user has the right access to comment on the post
	if post.HasAccess(requester.ID) {
		pg := s.Worker().Argument().GetPagination(request)
		comments := _Model.Post.GetCommentsByPostID(post.ID, pg)
		r := make([]tools.M, 0, len(comments))
		for _, c := range comments {
			r = append(r, s.Worker().Map().Comment(c))
		}
		response.OkWithData(tools.M{
			"total_comments": post.Counters.Comments,
			"comments":       r,
		})
	} else {
		response.Error(global.ErrAccess, []string{})
	}
	return
}

// @Command:	post/get_comment
// @Input:	post_id			string	*
// @Input:	comment_id		string	*
func (s *PostService) getCommentByID(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	var comment *nested.Comment
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	if comment = s.Worker().Argument().GetComment(request, response); comment == nil {
		return
	}

	response.OkWithData(s.Worker().Map().Comment(*comment))
}

// @Command:	post/get_many_comments
// @Input:	post_id			string	*
// @Input:	comment_id		string	*	(comma separated)
func (s *PostService) getManyCommentsByIDs(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	commentIDs := make([]bson.ObjectId, 0, global.DefaultMaxResultLimit)
	noAccessCommentIDs := make([]bson.ObjectId, 0)
	if v, ok := request.Data["comment_id"].(string); ok {
		for _, cid := range strings.Split(v, ",") {
			if bson.IsObjectIdHex(cid) {
				commentID := bson.ObjectIdHex(cid)
				if _Model.Post.CommentHasAccess(commentID, requester.ID) {
					commentIDs = append(commentIDs, commentID)
				} else {
					noAccessCommentIDs = append(noAccessCommentIDs, commentID)
				}
			}
		}
		if len(commentIDs) == 0 {
			response.Error(global.ErrInvalid, []string{"comment_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"comment_id"})
		return
	}

	comments := _Model.Post.GetCommentsByIDs(commentIDs)
	r := make([]tools.M, 0, len(comments))
	for _, comment := range comments {
		r = append(r, s.Worker().Map().Comment(comment))
	}
	response.OkWithData(tools.M{
		"comments":  r,
		"no_access": noAccessCommentIDs,
	})
}

// @Command:	post/mark_as_read
// @Input:	post_id			string	*
func (s *PostService) markPostAsRead(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}

	if !post.HasAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{"post_id"})
		return
	}
	post.MarkAsRead(requester.ID)
	notificationIDs := _Model.Notification.MarkAsReadByPostID(post.ID, requester.ID)
	for _, notificationID := range notificationIDs {
		notification := _Model.Notification.GetByID(notificationID)
		if notification != nil && notification.AccountID == requester.ID {
			_Model.Notification.MarkAsRead(notificationID, requester.ID)
			go s.Worker().Pusher().ClearNotification(requester, notification)
		}
	}
	response.Ok()
}

// @Command:	post/add_to_bookmarks
// @Input:	post_id			string	*
func (s *PostService) addToBookmarks(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}

	// check if user has access to the post
	if !post.HasAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{"post_id"})
		return
	}

	_Model.Post.BookmarkPost(requester.ID, post.ID)
	response.Ok()
}

// @Command:	post/remove_comment
// @Input:	post_id			string	*
// @Input:	comment_id		string	*
func (s *PostService) removeComment(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	var comment *nested.Comment
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	if comment = s.Worker().Argument().GetComment(request, response); comment == nil {
		return
	}

	// if user is the sender of the comment he/she can remove in the retract time period
	if comment.SenderID == requester.ID && comment.Timestamp+global.DefaultPostRetractTime > nested.Timestamp() {
		_Model.Post.RemoveComment(requester.ID, comment.ID)
		s.Worker().Pusher().PostCommentRemoved(post, comment)
		response.Ok()
		return
	}
	// if user is creator of one of the places the post is attached to
	for _, placeID := range post.PlaceIDs {
		place := _Model.Place.GetByID(placeID, nil)
		if place == nil {
			continue
		}
		access := place.GetAccess(requester.ID)
		if access[nested.PlaceAccessRemovePost] {
			_Model.Post.HideComment(comment.ID, requester.ID)
			s.Worker().Pusher().PostCommentRemoved(post, comment)
			response.Ok()
			return
		}
	}
	response.Error(global.ErrAccess, []string{})

	return

}

// @Command:	post/set_notification
// @Input:	post_id			string	*
// @Input:	state			bool		+
func (s *PostService) setPostNotification(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}

	// check if user has access to post
	if !post.HasAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{"post_id"})
		return
	}
	if v, ok := request.Data["state"].(bool); ok {
		if v {
			_Model.Post.AddAccountToWatcherList(post.ID, requester.ID)
		} else {
			_Model.Post.RemoveAccountFromWatcherList(post.ID, requester.ID)
		}
	}

	response.Ok()
}

// @Command:	post/remove
// @Input:	post_id			string	*
// @Input:	place_id			string	*
func (s *PostService) removePost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var place *nested.Place
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}

	if place = s.Worker().Argument().GetPlace(request, response); place == nil {
		return
	}

	access := place.GetAccess(requester.ID)
	if access[nested.PlaceAccessRemovePost] || requester.Authority.Admin {
		_Model.Post.Remove(requester.ID, post.ID, place.ID)
		response.Ok()
	} else {
		response.Error(global.ErrAccess, []string{})
	}
	return

}

// @Command:	post/remove_label
// @Input:	post_id			string	*
// @Input:	label_id			string	*
func (s *PostService) removeLabelFromPost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var label *nested.Label
	var post *nested.Post
	if label = s.Worker().Argument().GetLabel(request, response); label == nil {
		return
	}
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}

	// If label is not public and user is not member of the label then he/she has no permission
	// to add this label to posts
	if !label.Public && !label.IsMember(requester.ID) {
		response.Error(global.ErrAccess, []string{"not_member_of_label"})
		return
	}

	// If user has no access to the post, then he/she cannot add label to post
	if !post.HasAccess(requester.ID) {
		response.Error(global.ErrAccess, []string{"no_access_to_post"})
		return
	}

	if post.RemoveLabel(requester.ID, label.ID) {
		// handle push messages (activity)
		go s.Worker().Pusher().PostLabelRemoved(post, label)

		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
}

// @Command:	post/move
// @Input:	post_id			string	*
// @Input:	old_place_id		string	*
// @Input:	new_place_id		string	*
func (s *PostService) movePost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	var oldPlace, newPlace *nested.Place
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	if oldPlaceID, ok := request.Data["old_place_id"].(string); ok {
		oldPlace = _Model.Place.GetByID(oldPlaceID, nil)
		if oldPlace == nil {
			response.Error(global.ErrInvalid, []string{"old_place_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"old_place_id"})
		return
	}
	if newPlaceID, ok := request.Data["new_place_id"].(string); ok {
		newPlace = _Model.Place.GetByID(newPlaceID, nil)
		if newPlace == nil {
			response.Error(global.ErrInvalid, []string{"new_place_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"new_place_id"})
	}

	// Post must already be in oldPlaceID and must not be in newPlaceID
	if !post.IsInPlace(oldPlace.ID) || post.IsInPlace(newPlace.ID) {
		response.Error(global.ErrInvalid, []string{"old_place_id", "new_place_id"})
		return
	}

	// Get access for both places
	oldAccess := oldPlace.GetAccess(requester.ID)
	newAccess := newPlace.GetAccess(requester.ID)
	if !oldAccess[nested.PlaceAccessControl] || !newAccess[nested.PlaceAccessControl] {
		response.Error(global.ErrAccess, []string{"must_be_creator"})
		return
	}
	if _Model.Post.Move(post.ID, oldPlace.ID, newPlace.ID, requester.ID) {
		go s.Worker().Pusher().PostMovedTo(post, oldPlace, newPlace)
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
}

// @Command:	post/retract
// @Input:	post_id			string	*
func (s *PostService) retractPost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}

	// if user has the right permission to retract message
	if post.SenderID == requester.ID && nested.Timestamp() < post.Timestamp+global.DefaultPostRetractTime {
		for _, placeID := range post.PlaceIDs {
			if !_Model.Post.Remove(requester.ID, post.ID, placeID) {
				response.Error(global.ErrUnknown, []string{})
				return
			}
		}
		response.Ok()
	} else {
		response.Error(global.ErrAccess, []string{})
	}
	return

}

// @Command:	post/remove_from_bookmarks
// @Input:	post_id			string	*
func (s *PostService) removeFromBookmarks(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	_Model.Post.UnpinPost(requester.ID, post.ID)
	response.Ok()
}

// @Command:	post/who_read
// @Input:	post_id			string	*
// @Pagination
func (s *PostService) whoHaveReadThisPost(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	if post.SenderID != requester.ID {
		response.Error(global.ErrAccess, []string{"post_id"})
		return
	}
	pg := s.Worker().Argument().GetPagination(request)
	postReads := _Model.Post.GetAccountsWhoReadThis(post.ID, pg)
	var r []tools.M
	for _, pr := range postReads {
		account := _Model.Account.GetByID(pr.AccountID, nil)
		r = append(r, tools.M{
			"read_on":  pr.Timestamp,
			"place_id": pr.PlaceID,
			"account": tools.M{
				"_id":     account.ID,
				"fname":   account.FirstName,
				"lname":   account.LastName,
				"picture": account.Picture,
			},
		})
	}
	response.OkWithData(tools.M{
		"skip":       pg.GetSkip(),
		"limit":      pg.GetLimit(),
		"post_reads": r,
	})
}
