package api

import (
    "git.ronaksoftware.com/nested/server/model"
    "log"
    "github.com/globalsign/mgo/bson"
)

/*
    Creation Time: 2018 - Jul - 02
    Created by:  (ehsan)
    Maintainers:
        1.  (ehsan)
    Auditor: Ehsan N. Moosa
    Copyright Ronak Software Group 2018
*/

// Mapper converts model's structures into corresponding maps
type Mapper struct {
    worker *Worker
}

func NewMapper(worker *Worker) *Mapper {
    m := new(Mapper)
    m.worker = worker
    return m
}
func (m *Mapper) Account(account nested.Account, details bool) nested.M {
    r := nested.M{
        "_id":     account.ID,
        "fname":   account.FirstName,
        "lname":   account.LastName,
        "picture": account.Picture,
    }
    if details {
        r["counters"] = account.Counters
        r["admin"] = account.Authority.Admin
        r["disabled"] = account.Disabled
        r["dob"] = account.DateOfBirth
        r["phone"] = account.Phone
        r["email"] = account.Email
        r["privacy"] = account.Privacy
        r["gender"] = account.Gender
        r["joined_on"] = account.JoinedOn
        r["flags"] = account.Flags
        r["privacy"] = account.Privacy
        r["limits"] = account.Limits
        r["counters"] = account.Counters
        r["authority"] = account.Authority
        r["searchable"] = account.Privacy.Searchable
        r["bookmarked_places"] = account.BookmarkedPlaceIDs
		r["mail"] = nested.AccountMail{
			Active: account.Mail.Active,
			OutgoingSMTPHost: account.Mail.OutgoingSMTPHost,
			OutgoingSMTPUser: account.Mail.OutgoingSMTPUser,
			OutgoingSMTPPort: account.Mail.OutgoingSMTPPort,
		}
		r["access_place_ids"] = account.AccessPlaceIDs
    }
    return r
}
func (m *Mapper) Comment(comment nested.Comment) nested.M {
    s := m.worker.Model().Account.GetByID(comment.SenderID, nil)
    r := nested.M{
        "_id": comment.ID.Hex(),
        "sender": nested.M{
            "_id":     s.ID,
            "fname":   s.FirstName,
            "lname":   s.LastName,
            "picture": s.Picture,
        },
        "type":          comment.Type,
        "attachment_id": comment.AttachmentID,
        "text":          comment.Body,
        "timestamp":     comment.Timestamp,
        "post_id":       comment.PostID.Hex(),
        "removed_by":    comment.RemovedBy,
    }
    if len(comment.AttachmentID) > 0 {
        f := m.worker.Model().File.GetByID(comment.AttachmentID, nil)
        if f != nil {
            r["attachment"] = m.FileInfo(*f)
        }
    }
    return r
}
func (m *Mapper) Contact(requester *nested.Account, account nested.Account) nested.M {
    // TODO:: this is awful code fix it as soon as possible
    contacts := m.worker.Model().Contact.GetContacts(requester.ID)
    var isMutual, isFavorite, isContact bool
    for _, accountID := range contacts.MutualContacts {
        if accountID == account.ID {
            isMutual = true
            break
        }
    }
    for _, accountID := range contacts.FavoriteContacts {
        if accountID == account.ID {
            isFavorite = true
            isContact = true
            break
        }
    }
    if !isContact {
        for _, accountID := range contacts.Contacts {
            if accountID == account.ID {
                isContact = true
                break
            }
        }
    }
    r := nested.M{
        "_id":         account.ID,
        "fname":       account.FirstName,
        "lname":       account.LastName,
        "picture":     account.Picture,
        "is_contact":  isContact,
        "is_mutual":   isMutual,
        "is_favorite": isFavorite,
    }
    return r
}
func (m *Mapper) FileInfo(f nested.FileInfo) nested.M {
    // if UploadType is not set then set upload type as FILE
    if f.UploadType == "" {
        f.UploadType = nested.UPLOAD_TYPE_FILE
    }
    r := nested.M{
        "_id":         f.ID,
        "filename":    f.Filename,
        "type":        f.Type,
        "upload_type": f.UploadType,
        "mimetype":    f.MimeType,
        "size":        f.Size,
        "thumbs":      f.Thumbnails,
        "upload_time": f.UploadTimestamp,
        "width":       f.Width,
        "height":      f.Height,
        "meta":        f.Metadata,
    }
    return r
}
func (m *Mapper) Label(requester *nested.Account, label nested.Label, details bool) nested.M {
    r := nested.M{
        "_id":       label.ID,
        "title":     label.Title,
        "code":      label.ColourCode,
        "public":    label.Public,
        "is_member": label.IsMember(requester.ID),
    }

    if details {
        var members []nested.Account
        r["counters"] = label.Counters
        if len(label.Members) > 4 {
            members = m.worker.Model().Account.GetAccountsByIDs(label.Members[:4])
        } else {
            members = m.worker.Model().Account.GetAccountsByIDs(label.Members)
        }
        topMembers := []nested.M{}
        for _, member := range members {
            topMembers = append(topMembers, m.Account(member, false))
        }
        r["top_members"] = topMembers

    }
    return r
}
func (m *Mapper) LabelRequest(labelRequest nested.LabelRequest) nested.M {
    var label *nested.Label
    account := m.worker.Model().Account.GetByID(labelRequest.RequesterID, nil)
    r := nested.M{
        "_id":       labelRequest.ID,
        "title":     labelRequest.Title,
        "code":      labelRequest.ColourCode,
        "requester": m.Account(*account, false),
        "timestamp": labelRequest.Timestamp,
    }
    if len(labelRequest.LabelID) > 0 {
        label = m.worker.Model().Label.GetByID(labelRequest.LabelID)
        r["label"] = nested.M{
            "_id":   label.ID,
            "title": label.Title,
            "code":  label.ColourCode,
        }
    }
    return r
}
func (m *Mapper) Notification(requester *nested.Account, n nested.Notification) nested.M {
    r := nested.M{
        "_id":         n.ID,
        "type":        n.Type,
        "actor_id":    n.ActorID,
        "subject":     n.Subject,
        "read":        n.Read,
        "timestamp":   n.Timestamp,
        "last_update": n.LastUpdate,
    }
    s := m.worker.Model().Account.GetByID(n.ActorID, nil)
    if s != nil {
        r["actor"] = nested.M{
            "_id":     s.ID,
            "fname":   s.FirstName,
            "lname":   s.LastName,
            "picture": s.Picture,
        }
    }
    switch n.Type {
    case nested.NOTIFICATION_TYPE_MENTION:
        comment := m.worker.Model().Post.GetCommentByID(n.CommentID)
        r["post_id"] = n.PostID.Hex()
        if comment != nil {
            r["comment"] = m.worker.Map().Comment(*comment)
        }
        // TODO:: Deprecate it
        r["comment_id"] = n.CommentID.Hex()
        if comment != nil {
            r["comment_text"] = comment.Body
        }
    case nested.NOTIFICATION_TYPE_COMMENT:
        comment := m.worker.Model().Post.GetCommentByID(n.CommentID)
        otherCommenters := make([]nested.M, 0, len(n.Data.Others))
        r["post_id"] = n.PostID.Hex()
        r["comment"] = m.worker.Map().Comment(*comment)
        for _, account := range m.worker.Model().Account.GetAccountsByIDs(n.Data.Others) {
            otherCommenters = append(otherCommenters, m.worker.Map().Account(account, false))
        }
        r["others"] = otherCommenters

        // TODO:: Deprecate it
        r["comment_id"] = n.CommentID.Hex()
        r["data"] = n.Data
        if comment != nil {
            r["comment_text"] = comment.Body
        }
    case nested.NOTIFICATION_TYPE_JOINED_PLACE:
        p := m.worker.Model().Place.GetByID(n.PlaceID, nested.M{"name": 1, "picture": 1})
        if p != nil {
            r["place"] = m.worker.Map().Place(requester, *p, p.GetAccess(requester.ID))
            // TODO:: Deprecate it

            r["place_id"] = p.ID
            r["place_name"] = p.Name
            r["place_picture"] = p.Picture
        }
    case nested.NOTIFICATION_TYPE_DEMOTED, nested.NOTIFICATION_TYPE_PROMOTED, nested.NOTIFICATION_TYPE_PLACE_SETTINGS_CHANGED:
        p := m.worker.Model().Place.GetByID(n.PlaceID, nil)
        if p != nil {
            r["place"] = m.worker.Map().Place(requester, *p, p.GetAccess(requester.ID))
            // TODO:: Deprecate it

            r["place_id"] = p.ID
            r["place_name"] = p.Name
            r["place_picture"] = p.Picture
        }
    case nested.NOTIFICATION_TYPE_LABEL_REQUEST_APPROVED, nested.NOTIFICATION_TYPE_LABEL_REQUEST_REJECTED:
        label := m.worker.Model().Label.GetByID(n.LabelID)
        if label != nil {
            r["label"] = m.worker.Map().Label(requester, *label, false)
        }
    case nested.NOTIFICATION_TYPE_NEW_SESSION:
        r["client_id"] = n.ClientID
    case nested.NOTIFICATION_TYPE_TASK_REJECTED, nested.NOTIFICATION_TYPE_TASK_ACCEPTED,
        nested.NOTIFICATION_TYPE_TASK_COMPLETED, nested.NOTIFICATION_TYPE_TASK_ADD_TO_CANDIDATES,
        nested.NOTIFICATION_TYPE_TASK_ADD_TO_WATCHERS, nested.NOTIFICATION_TYPE_TASK_UPDATED,
        nested.NOTIFICATION_TYPE_TASK_OVER_DUE, nested.NOTIFICATION_TYPE_TASK_ASSIGNEE_CHANGED,
        nested.NOTIFICATION_TYPE_TASK_ADD_TO_EDITORS, nested.NOTIFICATION_TYPE_TASK_DUE_TIME_UPDATED,
        nested.NOTIFICATION_TYPE_TASK_ASSIGNED:
        r["task_id"] = n.TaskID.Hex()

        // HACK FOR ANDROID :(
        n.Data.Others = []string{"nested"}
        r["data"] = n.Data
    case nested.NOTIFICATION_TYPE_TASK_COMMENT, nested.NOTIFICATION_TYPE_TASK_MENTION:
        comment := m.worker.model.TaskActivity.GetActivityByID(n.Data.ActivityID)
        r["task_id"] = n.TaskID.Hex()
        r["data"] = n.Data
        r["comment_text"] = comment.CommentText

    }

    return r
}
func (m *Mapper) Place(requester *nested.Account, place nested.Place, access nested.MB) nested.M {
    if access == nil {
        return nested.M{
            "_id":         place.ID,
            "name":        place.Name,
            "description": place.Description,
            "picture":     place.Picture,
        }
    }
    a := make([]string, 0)
    for k, v := range access {
        if v {
            a = append(a, k)
        }
    }

    if !access[nested.PLACE_ACCESS_READ_POST] {
        r := nested.M{
            "_id":         place.ID,
            "type":        place.Type,
            "name":        place.Name,
            "description": place.Description,
            "picture":     place.Picture,
            "access":      a,
        }
        if place.Privacy.Receptive == nested.PLACE_RECEPTIVE_EXTERNAL {
            r["receptive"] = nested.PLACE_RECEPTIVE_EXTERNAL
        }
        return r
    }

    memberType := nested.MEMBER_TYPE_KEY_HOLDER
    if access[nested.PLACE_ACCESS_CONTROL] {
        memberType = nested.MEMBER_TYPE_CREATOR
    }
    r := nested.M{
        "_id":             place.ID,
        "type":            place.Type,
        "name":            place.Name,
        "description":     place.Description,
        "picture":         place.Picture,
        "grand_parent_id": place.GrandParentID,
        "privacy":         place.Privacy,
        "policy":          place.Policy,
        "access":          a,
        "member_type":     memberType,
        "limits":          place.Limit,
        "counters":        place.Counter,
        "favorite":        requester.IsBookmarked(place.ID),
        "notification":    m.worker.Model().Group.ItemExists(place.Groups["_ntfy"], requester.ID),
        "unread_posts":    m.worker.Model().Place.CountUnreadPosts([]string{place.ID}, requester.ID),
        "pinned_posts":    place.PinnedPosts,
    }
    return r
}
func (m *Mapper) PlaceActivity(requester *nested.Account, placeActivity nested.PlaceActivity, details bool) nested.M {
    if details {
        var post *nested.Post
        actor := m.worker.Model().Account.GetByID(placeActivity.Actor, nil)
        place := m.worker.Model().Place.GetByID(placeActivity.PlaceID, nil)
        r := nested.M{
            "_id":       placeActivity.ID.Hex(),
            "actor_id":  placeActivity.Actor,
            "action":    placeActivity.Action,
            "member_id": placeActivity.MemberID,
            "place_id":  placeActivity.PlaceID,
            "place":     m.Place(requester, *place, nil),
            "timestamp": placeActivity.LastUpdate,
        }
        if actor != nil {
            r["actor"] = m.Account(*actor, false)
        }
        switch placeActivity.Action {
        case nested.PLACE_ACTIVITY_ACTION_POST_ADD:
            post = m.worker.Model().Post.GetPostByID(placeActivity.PostID)
            if post != nil {
                if !post.Internal {
                    r["actor"] = nested.M{
                        "_id":     post.SenderID,
                        "fname":   post.EmailMetadata.Name,
                        "lname":   "",
                        "picture": post.EmailMetadata.Picture,
                    }
                }
                r["post"] = m.worker.Map().Post(requester, *post, true)
                // TODO:: Deprecate it
                r["post_id"] = post.ID.Hex()
                r["post_preview"] = post.Preview
                r["post_subject"] = post.Subject
            }
        case nested.PLACE_ACTIVITY_ACTION_MEMBER_REMOVE:
            member := m.worker.Model().Account.GetByID(placeActivity.MemberID, nil)
            if member != nil {
                r["member"] = m.Account(*member, false)
            }
        case nested.PLACE_ACTIVITY_ACTION_POST_MOVE_TO, nested.PLACE_ACTIVITY_ACTION_POST_MOVE_FROM:
            post = m.worker.Model().Post.GetPostByID(placeActivity.PostID)
            if post != nil {
                if !post.Internal {
                    r["actor"] = nested.M{
                        "_id":     post.SenderID,
                        "fname":   post.EmailMetadata.Name,
                        "lname":   "",
                        "picture": post.EmailMetadata.Picture,
                    }
                }
                r["post"] = m.worker.Map().Post(requester, *post, true)
            } else {
                log.Println("Post does not exists:", place.ID, placeActivity.PostID)
            }
            oldPlace := m.worker.Model().Place.GetByID(placeActivity.OldPlaceID, nil)
            newPlace := m.worker.Model().Place.GetByID(placeActivity.NewPlaceID, nil)
            if oldPlace != nil {
                r["old_place"] = m.Place(requester, *oldPlace, nil)
            }
            if newPlace != nil {
                r["new_place"] = m.Place(requester, *newPlace, nil)
            }
        }
        return r
    }
    r := nested.M{
        "_id":          placeActivity.ID.Hex(),
        "actor_id":     placeActivity.Actor,
        "action":       placeActivity.Action,
        "post_id":      placeActivity.PostID.Hex(),
        "label_id":     placeActivity.LabelID,
        "member_id":    placeActivity.MemberID,
        "comment_id":   placeActivity.CommentID.Hex(),
        "place_id":     placeActivity.PlaceID,
        "timestamp":    placeActivity.LastUpdate,
        "new_place_id": placeActivity.NewPlaceID,
        "old_place_id": placeActivity.OldPlaceID,
    }
    return r
}
func (m *Mapper) Post(requester *nested.Account, post nested.Post, preview bool) nested.M {
    isTrusted := true
    if !post.Internal {
        if !m.worker.Model().Account.IsRecipientTrusted(requester.ID, post.SenderID) {
            isTrusted = false
        }
    }
    s := new(nested.Account)
    r := nested.M{
        "_id":             post.ID.Hex(),
        "type":            post.Type,
        "subject":         post.Subject,
        "internal":        post.Internal,
        "is_trusted":      isTrusted,
        "ellipsis":        post.Ellipsis,
        "post_read":       m.worker.Model().Post.HasBeenReadBy(post.ID, requester.ID),
        "watched":         m.worker.Model().Post.HasBeenWatchedBy(post.ID, requester.ID),
        "pinned":          m.worker.Model().Post.IsPinned(requester.ID, post.ID),
        "post_recipients": post.Recipients,
        "timestamp":       post.Timestamp,
        "last_update":     post.LastUpdate,
        "content_type":    post.ContentType,
        "counters":        post.Counters,
        "recent_comments": post.RecentComments,
        "no_comment":      post.SystemData.NoComment,
    }

    if len(post.IFrameUrl) > 0 {
        r["iframe_url"] = post.IFrameUrl
    }
    // check if user can retract
    if post.SenderID == requester.ID && nested.Timestamp() < post.Timestamp+nested.DEFAULT_POST_RETRACT_TIME {
        r["wipe_access"] = true
    }

    // check if show body or preview
    if preview {
        r["preview"] = post.Preview
        // r["body"] = post.Body
    } else {
        r["preview"] = post.Preview
        r["body"] = post.Body
    }

    // preparing different presentations for internal and external posts
    if post.Internal {
        s = m.worker.Model().Account.GetByID(post.SenderID, nil)
        r["sender"] = m.Account(*s, false)
    } else {
        r["email_sender"] = nested.M{
            "_id":     post.SenderID,
            "name":    post.EmailMetadata.Name,
            "picture": post.EmailMetadata.Picture,
        }
    }

    // if post is forwarded
    if len(post.ForwardFrom.Hex()) > 0 {
        r["forward_from"] = post.ForwardFrom.Hex()
    }

    // if post is replied to another post
    if len(post.ReplyTo.Hex()) > 0 {
        r["reply_to"] = post.ReplyTo.Hex()
    }

    // present post_comments
    postRecentCommentIDs := []bson.ObjectId{}
    postRecentComments := []nested.M{}
    for _, comment := range post.RecentComments {
        postRecentCommentIDs = append(postRecentCommentIDs, comment.ID)
    }
    recentComments := m.worker.Model().Post.GetCommentsByIDs(postRecentCommentIDs)
    for _, comment := range recentComments {
        postRecentComments = append(postRecentComments, m.Comment(comment))
    }
    r["post_comments"] = postRecentComments

    // present post_places
    places := m.worker.Model().Place.GetPlacesByIDs(post.PlaceIDs)
    postPlaces := make([]nested.M, 0, len(places))
    for _, place := range places {
        r := nested.M{
            "_id":         place.ID,
            "name":        place.Name,
            "description": place.Description,
            "picture":     place.Picture,
        }
        r["access"] = place.GetAccessArray(requester.ID)
        postPlaces = append(postPlaces, r)
    }
    r["post_places"] = postPlaces

    // present post_labels
    labels := m.worker.Model().Label.GetByIDs(post.LabelIDs)
    postLabels := make([]nested.M, 0, len(labels))
    for _, label := range labels {
        postLabels = append(postLabels, m.Label(requester, label, false))
    }
    r["post_labels"] = postLabels

    // present post_attachments
    files := m.worker.Model().File.GetFilesByIDs(post.AttachmentIDs)
    postAttachments := make([]nested.M, 0, len(files))
    for _, file := range files {
        postAttachments = append(postAttachments, m.FileInfo(file))
    }
    r["post_attachments"] = postAttachments

    // present post related tasks
    tasks := m.worker.Model().Task.GetTasksByIDs(post.RelatedTasks)
    postTasks := make([]nested.M, 0, len(tasks))
    for _, task := range tasks {
        postTasks = append(postTasks, m.Task(requester, task, false))
    }
    r["related_tasks"] = postTasks
    return r
}
func (m *Mapper) PostActivity(requester *nested.Account, postActivity nested.PostActivity, details bool) nested.M {
    if details {
        var comment *nested.Comment
        actor := m.worker.Model().Account.GetByID(postActivity.ActorID, nil)
        r := nested.M{
            "_id":       postActivity.ID.Hex(),
            "actor_id":  postActivity.ActorID,
            "action":    postActivity.Action,
            "timestamp": postActivity.Timestamp,
        }
        if actor != nil {
            r["actor"] = m.Account(*actor, false)
        }
        switch postActivity.Action {
        case nested.POST_ACTIVITY_ACTION_COMMENT_ADD,
            nested.POST_ACTIVITY_ACTION_COMMENT_REMOVE:
            comment = m.worker.Model().Post.GetCommentByID(postActivity.CommentID)
            if comment != nil {
                r["comment"] = m.worker.Map().Comment(*comment)
            }
        case nested.POST_ACTIVITY_ACTION_LABEL_ADD,
            nested.POST_ACTIVITY_ACTION_LABEL_REMOVE:
            label := m.worker.Model().Label.GetByID(postActivity.LabelID)
            if label != nil {
                r["label"] = m.Label(requester, *label, false)
            }
        case nested.POST_ACTIVITY_ACTION_PLACE_ATTACH:
            newPlace := m.worker.Model().Place.GetByID(postActivity.NewPlaceID, nil)
            if newPlace != nil {
                r["new_place"] = m.worker.Map().Place(requester, *newPlace, nil)
            }
        case nested.POST_ACTIVITY_ACTION_PLACE_MOVE:
            oldPlace := m.worker.Model().Place.GetByID(postActivity.OldPlaceID, nil)
            newPlace := m.worker.Model().Place.GetByID(postActivity.NewPlaceID, nil)
            if oldPlace != nil {
                r["old_place"] = m.worker.Map().Place(requester, *oldPlace, nil)
            }
            if newPlace != nil {
                r["new_place"] = m.worker.Map().Place(requester, *newPlace, nil)
            }
        case nested.POST_ACTIVITY_ACTION_EDITED:
            post := m.worker.Model().Post.GetPostByID(postActivity.PostID)
            if post != nil {
                r["post"] = m.worker.Map().Post(requester, *post, true)
            }

        }
        return r
    }
    r := nested.M{
        "_id":        postActivity.ID.Hex(),
        "actor_id":   postActivity.ActorID,
        "action":     postActivity.Action,
        "post_id":    postActivity.PostID.Hex(),
        "label_id":   postActivity.LabelID,
        "comment_id": postActivity.CommentID.Hex(),
    }
    return r
}
func (m *Mapper) Task(requester *nested.Account, task nested.Task, details bool) nested.M {
    if !details {
        r := nested.M{
            "_id":                task.ID.Hex(),
            "title":              task.Title,
            "description":        task.Description,
            "status":             task.Status,
            "due_date":           task.DueDate,
            "due_data_has_clock": task.DueDateHasClock,
            "completed_on":       task.CompletedOn,
        }
        return r
    }
    r := nested.M{
        "_id":                task.ID.Hex(),
        "title":              task.Title,
        "description":        task.Description,
        "status":             task.Status,
        "counters":           task.Counters,
        "todos":              task.ToDos,
        "due_date":           task.DueDate,
        "due_data_has_clock": task.DueDateHasClock,
        "completed_on":       task.CompletedOn,
        "access":             task.GetAccessArray(requester.ID),
    }

    if task.CompletedOn > 0 {
        r["completed_on"] = task.CompletedOn
    }

    // Task Assignor
    taskAssignor := m.worker.Model().Account.GetByID(task.AssignorID, nil)
    r["assignor"] = m.Account(*taskAssignor, false)

    // Task Assignee
    if len(task.AssigneeID) > 0 {
        taskAssignee := m.worker.Model().Account.GetByID(task.AssigneeID, nil)
        r["assignee"] = m.Account(*taskAssignee, false)
    }

    // Task Candidates
    if len(task.CandidateIDs) > 0 {
        candidates := m.worker.Model().Account.GetAccountsByIDs(task.CandidateIDs)
        taskCandidates := make([]nested.M, 0, len(candidates))
        for _, candidate := range candidates {
            taskCandidates = append(taskCandidates, m.Account(candidate, false))
        }
        r["candidates"] = taskCandidates
    }

    // Task Watchers
    if len(task.WatcherIDs) > 0 {
        watchers := m.worker.Model().Account.GetAccountsByIDs(task.WatcherIDs)
        taskWatchers := make([]nested.M, 0, len(watchers))
        for _, watcher := range watchers {
            taskWatchers = append(taskWatchers, m.Account(watcher, false))
        }
        r["watchers"] = taskWatchers
    }

    // Task Editors
    if len(task.EditorIDs) > 0 {
        editors := m.worker.Model().Account.GetAccountsByIDs(task.EditorIDs)
        taskEditors := make([]nested.M, 0, len(editors))
        for _, editor := range editors {
            taskEditors = append(taskEditors, m.Account(editor, false))
        }
        r["editors"] = taskEditors
    }

    // Task Labels
    if len(task.LabelIDs) > 0 {
        labels := m.worker.Model().Label.GetByIDs(task.LabelIDs)
        taskLabels := make([]nested.M, 0, len(labels))
        for _, label := range labels {
            taskLabels = append(taskLabels, m.Label(requester, label, false))
        }
        r["labels"] = taskLabels
    }

    // Task Attachments
    if len(task.AttachmentIDs) > 0 {
        attachments := m.worker.Model().File.GetFilesByIDs(task.AttachmentIDs)
        taskAttachments := make([]nested.M, 0, len(attachments))
        for _, attachment := range attachments {
            taskAttachments = append(taskAttachments, m.FileInfo(attachment))
        }
        r["attachments"] = taskAttachments
    }

    if len(task.RelatedPost.Hex()) > 0 {
        rPost := m.worker.Model().Post.GetPostByID(task.RelatedPost)
        if rPost != nil && rPost.HasAccess(requester.ID) {
            r["related_post"] = m.Post(requester, *rPost, true)
        }
    }
    // Relate To
    if len(task.RelatedTo.Hex()) > 0 {
        rTask := m.worker.Model().Task.GetByID(task.RelatedTo)
        if rTask != nil && rTask.HasAccess(requester.ID, nested.TASK_ACCESS_READ) {
            r["related_to"] = nested.M{
                "_id":   rTask.ID.Hex(),
                "title": rTask.Title,
            }
        }
    }

    // Related Tasks
    if len(task.RelatedTasks) > 0 {
        rTasks := m.worker.Model().Task.GetTasksByIDs(task.RelatedTasks)
        relatedTasks := []nested.M{}
        for _, t := range rTasks {
            if t.HasAccess(requester.ID, nested.TASK_ACCESS_READ) {
                relatedTasks = append(relatedTasks, nested.M{
                    "_id":   t.ID,
                    "title": t.Title,
                })
            }
        }
        r["related_tasks"] = relatedTasks
    }

    return r

}
func (m *Mapper) TaskActivity(requester *nested.Account, taskActivity nested.TaskActivity, details bool) nested.M {
    r := nested.M{
        "_id":       taskActivity.ID,
        "timestamp": taskActivity.Timestamp,
        "action":    taskActivity.Action,
    }
    if !details {
        return r
    }

    actor := m.worker.Model().Account.GetByID(taskActivity.ActorID, nil)
    r["actor"] = m.Account(*actor, false)
    switch taskActivity.Action {
    case nested.TASK_ACTIVITY_WATCHER_ADDED, nested.TASK_ACTIVITY_WATCHER_REMOVED:
        watchers := m.worker.Model().Account.GetAccountsByIDs(taskActivity.WatcherIDs)
        d := make([]nested.M, 0, len(watchers))
        for _, w := range watchers {
            d = append(d, m.Account(w, false))
        }
        r["watchers"] = d
    case nested.TASK_ACTIVITY_EDITOR_ADDED, nested.TASK_ACTIVITY_EDITOR_REMOVED:
        editors := m.worker.Model().Account.GetAccountsByIDs(taskActivity.EditorIDs)
        d := make([]nested.M, 0, len(editors))
        for _, w := range editors {
            d = append(d, m.Account(w, false))
        }
        r["editors"] = d
    case nested.TASK_ACTIVITY_ATTACHMENT_ADDED, nested.TASK_ACTIVITY_ATTACHMENT_REMOVED:
        attachments := m.worker.Model().File.GetFilesByIDs(taskActivity.AttachmentIDs)
        d := make([]nested.M, 0, len(attachments))
        for _, a := range attachments {
            d = append(d, m.FileInfo(a))
        }
        r["attachments"] = d
    case nested.TASK_ACTIVITY_COMMENT:
        r["comment_text"] = taskActivity.CommentText
    case nested.TASK_ACTIVITY_TITLE_CHANGED:
        r["title"] = taskActivity.Title
    case nested.TASK_ACTIVITY_DESC_CHANGED:
        r["description"] = taskActivity.Desc
    case nested.TASK_ACTIVITY_CANDIDATE_ADDED, nested.TASK_ACTIVITY_CANDIDATE_REMOVED:
        candidates := m.worker.Model().Account.GetAccountsByIDs(taskActivity.CandidateIDs)
        d := []nested.M{}
        for _, w := range candidates {
            d = append(d, m.Account(w, false))
        }
        r["candidates"] = d
    case nested.TASK_ACTIVITY_TODO_ADDED, nested.TASK_ACTIVITY_TODO_REMOVED, nested.TASK_ACTIVITY_TODO_CHANGED,
        nested.TASK_ACTIVITY_TODO_DONE, nested.TASK_ACTIVITY_TODO_UNDONE:
        r["todo_text"] = taskActivity.ToDoText
    case nested.TASK_ACTIVITY_STATUS_CHANGED:
        r["status"] = taskActivity.Status
    case nested.TASK_ACTIVITY_LABEL_ADDED, nested.TASK_ACTIVITY_LABEL_REMOVED:
        labels := m.worker.Model().Label.GetByIDs(taskActivity.LabelIDs)
        mapLabels := []nested.M{}
        for _, label := range labels {
            mapLabels = append(mapLabels, m.Label(requester, label, false))
        }
        r["labels"] = mapLabels
    case nested.TASK_ACTIVITY_DUE_DATE_UPDATED, nested.TASK_ACTIVITY_DUE_DATE_REMOVED:
        r["due_date"] = taskActivity.DueDate
        r["due_date_has_clock"] = taskActivity.DueDateHasClock
    case nested.TASK_ACTIVITY_ASSIGNEE_CHANGED:
        assignee := m.worker.Model().Account.GetByID(taskActivity.AssigneeID, nil)
        r["assignee"] = m.Account(*assignee, false)
    }
    return r
}
func (m *Mapper) App(app nested.App) nested.M {
    r := nested.M{
        "_id":            app.ID,
        "name":           app.Name,
        "developer":      app.Developer,
        "homepage":       app.Homepage,
        "icon_small_url": app.IconSmallURL,
        "icon_large_url": app.IconLargeURL,
    }
    return r
}
func (m *Mapper) AppToken(appToken nested.AppToken) nested.M {
    r := nested.M{
        "_id": appToken.ID,
    }
    if account := m.worker.Model().Account.GetByID(appToken.AccountID, nil); account != nil {
        r["account"] = m.Account(*account, false)
    }
    if app := m.worker.Model().App.GetByID(appToken.AppID); app != nil {
        r["app"] = m.App(*app)
    }
    return r
}
