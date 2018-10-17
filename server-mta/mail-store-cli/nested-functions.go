package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"git.ronaksoftware.com/nested/server/model"
	"github.com/globalsign/mgo/bson"
	"github.com/microcosm-cc/bluemonday"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"strings"
	"time"
)

func (m *Model) PlaceExist(placeID string) bool {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	n, _ := db.C(nested.COLLECTION_PLACES).FindId(placeID).Count()

	return n > 0
}

func (m *Model) AccountExist(accountID string) bool {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	n, _ := db.C(nested.COLLECTION_ACCOUNTS).FindId(accountID).Count()
	return n > 0
}

func (m *Model) CreateFileToken(uniID nested.UniversalID, issuerID, receiverEmail string) (string, error) {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	ft := nested.FileToken{
		ID:       fmt.Sprintf("FT%s", nested.RandomID(128)),
		Type:     nested.TOKEN_TYPE_FILE,
		FileID:   uniID,
		Issuer:   issuerID,
		Receiver: receiverEmail,
	}
	if err := db.C(nested.COLLECTION_TOKENS_FILES).Insert(ft); err != nil {
		_LOG.Error(err.Error())
		return "", err
	}
	return ft.ID, nil
}

func (m *Model) GetPlaceByID(placeID string) *nested.Place {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	place := new(nested.Place)
	if err := db.C(nested.COLLECTION_PLACES).FindId(placeID).One(place); err != nil {
		_LOG.Error(err.Error())
		return nil
	}
	return place
}

func (m *Model) GetItems(groupID string) []string {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	v := struct {
		ID    string   `json:"_id" bson:"_id"`
		Items []string `json:"items" bson:"items"`
	}{}
	if err := db.C(nested.COLLECTION_PLACES_GROUPS).FindId(groupID).One(&v); err != nil {
		return []string{}
	}
	return v.Items
}

func (m *Model) GetAccountByID(accountID string) *nested.Account {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	account := new(nested.Account)
	if err := db.C(nested.COLLECTION_ACCOUNTS).FindId(accountID).One(account); err != nil {
		_LOG.Error(err.Error(), zap.String("accountID", accountID))
		return nil
	}
	return account
}

// easyjson:json
type CMDPushExternal struct {
	Targets []string          `json:"targets"`
	Data    map[string]string `json:"data"`
}

// External Pushes
func (m *Model) ExternalPush(targets []string, data map[string]string) error {
	cmd := CMDPushExternal{
		Targets: targets,
		Data:    data,
	}
	cmd.Data["domain"] = m.Ntfy.Domain
	if b, err := json.Marshal(cmd); err != nil {
		return err
	} else {
		m.Ntfy.Nat.Publish("NTFY.PUSH.EXTERNAL", b)
	}
	return nil
}

func (m *Model) ExternalPushPlaceActivityPostAdded(post *nested.Post) {
	pushData := nested.MS{
		"type":   "a",
		"action": fmt.Sprintf("%d", nested.PLACE_ACTIVITY_ACTION_POST_ADD),
	}

	if post.Internal {
		_LOG.Debug("Post is internal", zap.Bool("post.INternal", post.Internal))
		actor := m.GetAccountByID(post.SenderID)
		pushData["actor_id"] = actor.ID
		pushData["actor_name"] = fmt.Sprintf("%s %s", actor.FirstName, actor.LastName)
		pushData["actor_picture"] = string(actor.Picture.X128)
		pushData["title"] = "New Post"
	} else {
		pushData["actor_id"] = post.SenderID
		pushData["actor_name"] = post.EmailMetadata.Name
		pushData["actor_picture"] = string(post.EmailMetadata.Picture.Original)
		pushData["title"] = "New Email"
	}
	pushData["sound"] = "np.aiff"
	pushData["post_id"] = post.ID.Hex()

	// Prepare the 'msg' based on which keys are provided: subject | body | attachments
	if len(post.Subject) > 0 {
		pushData["msg"] = fmt.Sprintf("%s shared a post: %s", pushData["actor_name"], post.Subject)
	} else if len(post.Preview) > 0 {
		pushData["msg"] = fmt.Sprintf("%s shared a post: %s", pushData["actor_name"], post.Preview)
	} else {
		if post.Counters.Attachments == 1 {
			pushData["msg"] = fmt.Sprintf("%s shared a post with one Attachment", pushData["actor_name"])
		} else {
			pushData["msg"] = fmt.Sprintf("%s shared a post with %d Attachments", pushData["actor_name"], post.Counters.Attachments)
		}
	}

	for _, placeID := range post.PlaceIDs {
		place := m.GetPlaceByID(placeID)
		if place == nil {
			_LOG.Debug("ExternalPushActivityAddPost::Error::Place_Not_Exists")
			continue
		}
		memberIDs := m.GetItems(place.Groups[nested.NOTIFICATION_GROUP])
		for _, memberID := range memberIDs {
			if memberID != post.SenderID {
				pushData["account_id"] = memberID
				if err := m.ExternalPush([]string{memberID}, pushData); err != nil {
					_LOG.Error(err.Error())
				}
			}
		}
	}
}

func (m *Model) InternalPlaceActivitySyncPush(targets []string, placeID string, action int) {
	_LOG.Info("InternalPlaceActivitySyncPush", zap.Strings("targets",targets), zap.String("placeID",placeID), zap.Int("action", action))
	if len(targets) == 0 {
		return
	}
	iStart := 0
	iLength := nested.DEFAULT_MAX_RESULT_LIMIT
	iEnd := iStart + iLength
	if iEnd > len(targets) {
		iEnd = len(targets)
	}
	for {
		msg := nested.M{
			"type": "p",
			"cmd":  "sync-a",
			"data": nested.M{
				"place_id": placeID,
				"action":   action,
			},
		}
		if jmsg, err := json.Marshal(msg); err != nil {
			_LOG.Error("NotificationClient::InternalPlaceActivitySyncPush::Error::", zap.Error(err))
		} else {
			if err := m.InternalPush(targets[iStart:iEnd], string(jmsg), false); err != nil {
				_LOG.Error(err.Error())
			}
		}
		iStart += iLength
		iEnd = iStart + iLength
		if iStart >= len(targets) {
			break
		}
		if iEnd > len(targets) {
			iEnd = len(targets)
		}
	}
}

// easyjson:json
type CMDPushInternal struct {
	Targets   []string `json:"targets"`
	LocalOnly bool     `json:"local_only"`
	Message   string   `json:"msg"`
}

// Internal Pushes
func (m *Model) InternalPush(targets []string, msg string, localonly bool) error {
	cmd := CMDPushInternal{
		Targets:   targets,
		Message:   msg,
		LocalOnly: localonly,
	}

	if b, err := json.Marshal(cmd); err != nil {
		_LOG.Error("NotificationClient::InternalPush::Error::", zap.Error(err))
		return err
	} else {
		if err := m.Ntfy.Nat.Publish("NTFY.PUSH.INTERNAL", b); err != nil {
			_LOG.Error(err.Error())
		}
	}
	return nil
}

func (m *Model) AddPostAsOwner(uniID nested.UniversalID, postID bson.ObjectId) {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	if err := db.C(nested.COLLECTION_FILES).UpdateId(
		uniID,
		bson.M{"$inc": bson.M{"ref_count": 1}},
	); err != nil {
		_LOG.Error(err.Error(), zap.String("Function","AddPostAsOwner::COLLECTION_FILES"))
	}

	if err := db.C(nested.COLLECTION_POSTS_FILES).Insert(
		bson.M{
			"universal_id": uniID,
			"post_id":      postID,
		},
	); err != nil {
		_LOG.Error(err.Error(), zap.String("Function","AddPostAsOwner::COLLECTION_POSTS_FILES"))
	}
}

func (m *Model) GetGrandParentIDs(placeIDs []string) []string {
	var res []string
	for _, v := range placeIDs {
		res = append(res, strings.Split(v, ".")[0])
	}
	return res
}

func (m *Model) IncrementCounter(placeIDs []string, counterName string, c int) bool {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	switch counterName {
	case nested.PLACE_COUNTER_CHILDREN, nested.PLACE_COUNTER_UNLOCKED_CHILDREN,
		nested.PLACE_COUNTER_CREATORS, nested.PLACE_COUNTER_KEYHOLDERS,
		nested.PLACE_COUNTER_POSTS, nested.PLACE_COUNTER_QUOTA:
		keyName := fmt.Sprintf("counters.%s", counterName)
		if err := db.C(nested.COLLECTION_PLACES).Update(
			bson.M{"_id": bson.M{"$in": placeIDs}},
			bson.M{"$inc": bson.M{keyName: c}},
		); err != nil {
			_LOG.Error(err.Error())
			return false
		}
	}
	return true
}

func (m *Model) UpdatePlaceConnection(accountID string, placeIDs []string, c int) {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	bulk := db.C(nested.COLLECTION_ACCOUNTS_PLACES).Bulk()
	bulk.Unordered()
	for _, pid := range placeIDs {
		if place := m.GetPlaceByID(pid); place != nil {
			bulk.Upsert(
				bson.M{
					"account_id": accountID,
					"place_id":   pid,
				},
				bson.M{
					"$inc": bson.M{"pts": c},
				},
			)
		}
	}
	bulk.Run()
}

func (m *Model) UpdateRecipientConnection(accountID string, recipients []string, c int) {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()
	for _, r := range recipients {
		if _, err := db.C(nested.COLLECTION_ACCOUNTS_RECIPIENTS).Upsert(
			bson.M{
				"account_id": accountID,
				"recipient":  strings.ToLower(r),
			},
			bson.M{"$inc": bson.M{"pts": c}},
		); err != nil {
			_LOG.Error("Model::AccountManager::UpdateRecipientConnection::Error 1::", zap.Error(err))
		}
	}
}

func (m *Model) HasReadAccess(accountID string, p *nested.Place) bool {
	if m.IsMember(accountID, p) {
		return true
	} else if !p.Privacy.Locked && !p.IsGrandPlace() {
		gp := m.GetPlaceByID(p.GrandParentID)
		if gp.IsMember(accountID) {
			return true
		}
	}
	return false
}

func (m *Model) AddAccountToWatcherList(postID bson.ObjectId, accountID string) bool {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	if _, err := db.C(nested.COLLECTION_POSTS_WATCHERS).Upsert(
		bson.M{"_id": postID},
		bson.M{"$addToSet": bson.M{"accounts": accountID}},
	); err != nil {
		_LOG.Error(err.Error())
		return false
	}
	return true
}

func (m *Model) LableIncrementCounter(labelID string, counterName string, value int) bool {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	if err := db.C(nested.COLLECTION_LABELS).UpdateId(
		labelID,
		bson.M{
			"$inc": bson.M{fmt.Sprintf("counters.%s", counterName): value},
		},
	); err != nil {
		_LOG.Error(err.Error())
		return false
	}
	return true
}

func (m *Model) PostAdd(actorID string, placeIDs []string, postID bson.ObjectId) {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	ts := nested.Timestamp()
	bulk := db.C(nested.COLLECTION_PLACES_ACTIVITIES).Bulk()
	bulk.Unordered()
	v := nested.PlaceActivity{
		Timestamp:  ts,
		LastUpdate: ts,
		Action:     nested.PLACE_ACTIVITY_ACTION_POST_ADD,
		Actor:      actorID,
		PostID:     postID,
	}

	for _, placeID := range placeIDs {
		v.ID = bson.NewObjectId()
		v.PlaceID = placeID
		bulk.Insert(v)
	}
	if _, err := bulk.Run(); err != nil {
		_LOG.Error(err.Error())
	}
	return
}

func (m *Model) IsMember(accountID string, p *nested.Place) bool {
	for _, creatorID := range p.CreatorIDs {
		if creatorID == accountID {
			return true
		}
	}
	for _, keyholderID := range p.KeyholderIDs {
		if keyholderID == accountID {
			return true
		}
	}
	return false
}

func (m *Model) SetFileStatus(uniID nested.UniversalID, fileStatus string) bool {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	switch fileStatus {
	case nested.FILE_STATUS_PUBLIC, nested.FILE_STATUS_TEMP, nested.FILE_STATUS_THUMBNAIL:
		if err := db.C(nested.COLLECTION_FILES).UpdateId(
			uniID,
			bson.M{"$set": bson.M{
				"upload_time": time.Now().UnixNano(),
				"status":      fileStatus,
			}},
		); err != nil {
			_LOG.Error(err.Error(), zapcore.Field{Interface: uniID}, zap.Any("fileStatus", fileStatus))
			return false
		}
	case nested.FILE_STATUS_ATTACHED:
		if err := db.C(nested.COLLECTION_FILES).Update(
			bson.M{"_id": uniID, "status": bson.M{"$ne": nested.FILE_STATUS_PUBLIC}},
			bson.M{"$set": bson.M{"status": fileStatus}},
		); err != nil {
			_LOG.Error(err.Error(), zap.Any("uniID", uniID), zap.Any("fileStatus", fileStatus))
			return false
		}
	case nested.FILE_STATUS_INTERNAL:
		if err := db.C(nested.COLLECTION_FILES).Update(
			bson.M{"_id": uniID},
			bson.M{"$set": bson.M{"status": fileStatus}},
		); err != nil {
			_LOG.Error(err.Error(), zap.Any("uniID", uniID), zap.Any("fileStatus", fileStatus))
			return false
		}
	default:
		return false
	}
	return true
}

func (m *Model) GetFileByID(uniID nested.UniversalID, pj nested.M) *nested.FileInfo {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	file := new(nested.FileInfo)
	if err := db.C(nested.COLLECTION_FILES).FindId(uniID).One(file); err != nil {
		_LOG.Error(err.Error(), zap.String("Function","GetFileByID"))
		return nil
	}
	return file
}

func (m *Model) IsBlocked(placeID, address string) bool {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	n, err := db.C(nested.COLLECTION_PLACES_BLOCKED_ADDRESSES).FindId(placeID).Select(
		bson.M{"addresses": address},
	).Count()
	if err != nil {
		_LOG.Warn(err.Error())
		return false
	}
	return n > 0
}

func (m *Model) AddPost(pcr nested.PostCreateRequest) *nested.Post {
	dbSession := m.Session.Clone()
	db := dbSession.DB(m.DB)
	defer dbSession.Close()

	post := nested.Post{}
	ts := nested.Timestamp()
	post.Type = nested.POST_TYPE_NORMAL
	post.ReplyTo = pcr.ReplyTo
	post.ForwardFrom = pcr.ForwardFrom
	post.ContentType = pcr.ContentType

	// Returns nil if targets are more than DEFAULT_POST_MAX_TARGETS
	if len(pcr.PlaceIDs)+len(pcr.Recipients) > nested.DEFAULT_POST_MAX_TARGETS {
		return nil
	}

	// Returns nil if number of attachments exceeds DEFAULT_POST_MAX_ATTACHMENTS
	if len(pcr.AttachmentIDs) > nested.DEFAULT_POST_MAX_ATTACHMENTS {
		return nil
	}

	// Returns nil if number of labels exceeds DEFAULT_POST_MAX_LABELS
	if len(pcr.LabelIDs) > nested.DEFAULT_POST_MAX_LABELS {
		return nil
	}

	post.ID = bson.NewObjectId()
	post.Counters.Attachments = len(pcr.AttachmentIDs)
	post.SenderID = pcr.SenderID
	post.Body = pcr.Body
	post.Subject = pcr.Subject
	post.IFrameUrl = pcr.IFrameUrl
	post.AttachmentIDs = pcr.AttachmentIDs
	post.PlaceIDs = pcr.PlaceIDs
	post.Recipients = pcr.Recipients
	post.LabelIDs = pcr.LabelIDs
	post.Timestamp = ts
	post.LastUpdate = ts

	var attach_size int64
	for _, uniID := range pcr.AttachmentIDs {
		m.AddPostAsOwner(uniID, post.ID)
		success := m.SetFileStatus(uniID, nested.FILE_STATUS_ATTACHED)
		if !success {
			return nil
		}
		f := m.GetFileByID(uniID, nil)
		attach_size += f.Size
	}
	post.Counters.Size = attach_size

	// Increment Counters
	m.Cache.CountPostAdd()
	m.Cache.CountPostAttachCount(len(pcr.AttachmentIDs))
	m.Cache.CountPostAttachSize(attach_size)
	m.Cache.CountPostPerPlace(pcr.PlaceIDs)

	// fill email data
	post.EmailMetadata = pcr.EmailMetadata

	// fill post system data
	post.SystemData = pcr.SystemData

	// if post was an external post
	if m.AccountExist(pcr.SenderID) {
		post.Internal = true
	} else {
		post.Recipients = append(post.Recipients, post.SenderID)
		m.Cache.CountPostExternalAdd()
	}

	switch pcr.ContentType {
	case nested.CONTENT_TYPE_TEXT_PLAIN:
		if len(post.Body) > 256 {
			post.Ellipsis = true
			post.Preview = string(post.Body[:256])
		} else {
			post.Preview = post.Body
		}
	default:
		post.ContentType = nested.CONTENT_TYPE_TEXT_HTML
		strings.NewReader(pcr.Body)
		post.Body = sanitizeBody(strings.NewReader(pcr.Body), post.Internal).String()
		if len(pcr.Body) > 256 {
			post.Preview = sanitizePreview(strings.NewReader(pcr.Body[:256])).String()
		} else {
			post.Preview = sanitizePreview(strings.NewReader(pcr.Body)).String()
		}
		if len(post.Body) != len(post.Preview) || post.Preview != post.Body {
			post.Ellipsis = true
		}
	}

	if err := db.C(nested.COLLECTION_POSTS).Insert(post); err != nil {
		_LOG.Error(err.Error())
		return nil
	}

	// Update counters of the grand places
	grandParentIDs := m.GetGrandParentIDs(pcr.PlaceIDs)
	m.IncrementCounter(grandParentIDs, nested.PLACE_COUNTER_QUOTA, int(post.Counters.Size))

	// Update counters of the places
	m.IncrementCounter(pcr.PlaceIDs, nested.PLACE_COUNTER_POSTS, 1)

	// Update user contacts list
	if post.Internal {
		m.UpdatePlaceConnection(pcr.SenderID, pcr.PlaceIDs, 1)
		m.UpdateRecipientConnection(pcr.SenderID, pcr.Recipients, 1)
		m.Cache.CountPostPerAccount(pcr.SenderID)
	}

	// Create PostRead items per each user of each place
	for _, placeID := range pcr.PlaceIDs {
		// Remove place from cache
		m.Cache.PlaceRemoveCache(placeID)

		place := m.GetPlaceByID(placeID)
		grandPlace := m.GetPlaceByID(place.GrandParentID)
		if m.HasReadAccess(post.SenderID, place) {
			m.AddAccountToWatcherList(post.ID, post.SenderID)
		}

		// Set Post as UNREAD for all the members of the place except the sender
		var memberIDs []string
		bulk := db.C(nested.COLLECTION_POSTS_READS).Bulk()
		bulk.Unordered()
		if place.Privacy.Locked {
			memberIDs = append(place.KeyholderIDs, place.CreatorIDs...)
		} else {
			memberIDs = append(grandPlace.KeyholderIDs, grandPlace.CreatorIDs...)
		}
		for _, cid := range memberIDs {
			if cid == post.SenderID {
				continue
			}
			bulk.Insert(nested.PostRead{
				AccountID: cid,
				PlaceID:   placeID,
				PostID:    post.ID,
				Timestamp: ts,
			})
		}
		if _, err := bulk.Run(); err != nil {
			_LOG.Error(err.Error())
		}

		//// Clear the slice
		//memberIDs = memberIDs[:0]

		// Update unread counters
		if place.Privacy.Locked {
			db.C(nested.COLLECTION_POSTS_READS_COUNTERS).UpdateAll(
				bson.M{
					"account_id": bson.M{"$ne": post.SenderID},
					"place_id":   placeID,
				},
				bson.M{"$inc": bson.M{"no_unreads": 1}},
			)
		} else {
			db.C(nested.COLLECTION_POSTS_READS_COUNTERS).UpdateAll(
				bson.M{
					"account_id": bson.M{"$ne": post.SenderID},
					"place_id":   placeID,
				},
				bson.M{"$inc": bson.M{"no_unreads": 1}},
			)
		}
		// todo implement hook
		// Create the hook event and send it to the hooker
		//_Manager.Hook.chEvents <- NewPostEvent{
		//	PlaceID:          placeID,
		//	PostID:           post.ID,
		//	PostTitle:        post.Subject,
		//	AttachmentsCount: post.Counters.Attachments,
		//	SenderID:         post.SenderID,
		//}
	}

	// Update label counters
	for _, labelID := range post.LabelIDs {
		m.LableIncrementCounter(labelID, "posts", 1)
	}

	// Add the timeline activity to the database
	m.PostAdd(post.SenderID, post.PlaceIDs, post.ID)

	return &post
}

func sanitizeBody(input io.Reader, internal bool) *bytes.Buffer {
	_HtmlSanitizer := bluemonday.UGCPolicy()
	if internal {
		_HtmlSanitizer.AllowElements("style")
	}
	_HtmlSanitizer.AllowElements("div", "font", "br", "b", "a", "p", "span", "strong")
	_HtmlSanitizer.AllowElements("h1", "h2", "h3", "h4", "h5", "h6", "label")
	_HtmlSanitizer.AllowAttrs("dir", "align", "style", "border", "height", "max-height", "hspace", "usemap", "vspace", "width", "max-width").Globally()
	_HtmlSanitizer.AllowLists()
	_HtmlSanitizer.AllowDataURIImages()
	_HtmlSanitizer.AllowImages()
	_HtmlSanitizer.AllowTables()
	_HtmlSanitizer.AllowStyling()
	_HtmlSanitizer.AllowStandardURLs()
	_HtmlSanitizer.AddSpaceWhenStrippingTag(true)
	_HtmlSanitizer.AllowAttrs("face", "color").Matching(bluemonday.Paragraph).OnElements("font")
	_HtmlSanitizer.AllowAttrs("dir", "lang").Matching(bluemonday.Paragraph).OnElements("head")
	_HtmlSanitizer.AllowAttrs("align", "size", "width").Matching(bluemonday.Paragraph).OnElements("hr")
	_HtmlSanitizer.AllowAttrs("bgcolor", "border", "rules").Matching(bluemonday.Paragraph).OnElements("table")
	_HtmlSanitizer.AllowAttrs("cellspacing", "cellpadding").Matching(bluemonday.Number).OnElements("table")
	_HtmlSanitizer.AllowAttrs("bgcolor").OnElements("td", "th", "tr", "thead", "tbody", "table")
	_HtmlSanitizer.AllowAttrs("abbr").OnElements("th")
	return _HtmlSanitizer.SanitizeReader(input)
}

func sanitizePreview(input io.Reader) *bytes.Buffer {
	_HtmlSanitizer := bluemonday.StrictPolicy()
	_HtmlSanitizer.AllowStandardURLs()
	_HtmlSanitizer.AllowStandardAttributes()
	_HtmlSanitizer.AllowElements("div", "font", "br", "b", "a", "p", "span", "strong")
	_HtmlSanitizer.AllowElements("h1", "h2", "h3", "h4", "h5", "h6", "label")
	_HtmlSanitizer.AddTargetBlankToFullyQualifiedLinks(true)
	_HtmlSanitizer.AddSpaceWhenStrippingTag(true)
	return _HtmlSanitizer.SanitizeReader(input)
}
