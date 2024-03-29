package nested

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
	"io"
	"strings"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gomodule/redigo/redis"
	"github.com/microcosm-cc/bluemonday"
)

const (
	PostTypeNormal = 0x02
)
const (
	ContentTypeTextHtml  = "text/html"
	ContentTypeTextPlain = "text/plain"
)
const (
	PostSortTimestamp  = "timestamp"
	PostSortLastUpdate = "last_update"
	PostSortPinTime    = "pin_time"
)
const (
	CommentTypeText     CommentType = 0x00
	CommentTypeVoice    CommentType = 0x01
	CommentTypeActivity CommentType = 0x02
)

type PostManager struct{}

func newPostManager() *PostManager {
	return new(PostManager)
}

func (pm *PostManager) readFromCache(postID bson.ObjectId) *Post {
	post := new(Post)
	c := _Cache.Pool.Get()
	defer c.Close()
	keyID := fmt.Sprintf("post:gob:%s", postID.Hex())
	if gobPost, err := redis.Bytes(c.Do("GET", keyID)); err != nil {
		if err := _MongoDB.C(global.CollectionPosts).FindId(postID).One(post); err != nil {
			log.Warn("got error reading post from db", zap.Error(err))
			return nil
		}
		gobPost := new(bytes.Buffer)
		if err := gob.NewEncoder(gobPost).Encode(post); err == nil {
			c.Do("SETEX", keyID, global.CacheLifetime, gobPost.Bytes())
		}
		return post
	} else if err := gob.NewDecoder(bytes.NewBuffer(gobPost)).Decode(post); err == nil {
		return post
	}
	return nil
}

func (pm *PostManager) readCommentFromCache(commentID bson.ObjectId) *Comment {
	comment := new(Comment)
	c := _Cache.Pool.Get()
	defer c.Close()
	keyID := fmt.Sprintf("comment:gob:%s", commentID.Hex())
	if gobComment, err := redis.Bytes(c.Do("GET", keyID)); err != nil {
		if err := _MongoDB.C(global.CollectionPostsComments).FindId(commentID).One(comment); err != nil {
			log.Warn("Got error", zap.Error(err))
			return nil
		}
		gobComment := new(bytes.Buffer)
		if err := gob.NewDecoder(gobComment).Decode(comment); err == nil {
			c.Do("SETEX", keyID, global.CacheLifetime, gobComment.Bytes())
		}
		return comment
	} else if err := gob.NewDecoder(bytes.NewBuffer(gobComment)).Decode(comment); err == nil {
		return comment
	}
	return nil
}

func (pm *PostManager) readMultiFromCache(postIDs []bson.ObjectId) []Post {
	posts := make([]Post, 0, len(postIDs))
	c := _Cache.Pool.Get()
	defer c.Close()
	for _, postID := range postIDs {
		keyID := fmt.Sprintf("post:gob:%s", postID.Hex())
		c.Send("GET", keyID)
	}
	c.Flush()
	for _, postID := range postIDs {
		if gobPost, err := redis.Bytes(c.Receive()); err == nil {
			post := new(Post)
			if err := gob.NewDecoder(bytes.NewBuffer(gobPost)).Decode(post); err == nil {
				posts = append(posts, *post)
			}
		} else {
			if post := _Manager.Post.readFromCache(postID); post != nil {
				posts = append(posts, *post)
			}
		}
	}
	return posts
}

func (pm *PostManager) readMultiCommentsFromCache(commentIDs []bson.ObjectId) []Comment {
	comments := make([]Comment, 0, len(commentIDs))
	c := _Cache.Pool.Get()
	defer c.Close()
	for _, commentID := range commentIDs {
		keyID := fmt.Sprintf("comment:gob:%s", commentID.Hex())
		c.Send("GET", keyID)
	}
	c.Flush()
	for _, commentID := range commentIDs {
		if gobComment, err := redis.Bytes(c.Receive()); err == nil {
			comment := new(Comment)
			if err := gob.NewDecoder(bytes.NewBuffer(gobComment)).Decode(comment); err == nil {
				comments = append(comments, *comment)
			}
		} else {
			if comment := _Manager.Post.readCommentFromCache(commentID); comment != nil {
				comments = append(comments, *comment)
			}
		}
	}
	return comments
}

func (pm *PostManager) removeCache(postID bson.ObjectId) bool {
	c := _Cache.Pool.Get()
	defer c.Close()
	keyID := fmt.Sprintf("post:gob:%s", postID.Hex())
	c.Do("DEL", keyID)
	return true
}

func (pm *PostManager) removeCommentFromCache(commentID bson.ObjectId) bool {
	c := _Cache.Pool.Get()
	defer c.Close()
	keyID := fmt.Sprintf("comment:gob:%s", commentID.Hex())
	c.Do("DEL", keyID)
	return true
}

func (pm *PostManager) sanitizeBody(input io.Reader, internal bool) *bytes.Buffer {
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

func (pm *PostManager) sanitizePreview(input io.Reader) *bytes.Buffer {
	_HtmlSanitizer := bluemonday.StrictPolicy()
	_HtmlSanitizer.AllowStandardURLs()
	_HtmlSanitizer.AllowStandardAttributes()
	_HtmlSanitizer.AllowElements("div", "font", "br", "b", "a", "p", "span", "strong")
	_HtmlSanitizer.AllowElements("h1", "h2", "h3", "h4", "h5", "h6", "label")
	_HtmlSanitizer.AddTargetBlankToFullyQualifiedLinks(true)
	_HtmlSanitizer.AddSpaceWhenStrippingTag(true)
	return _HtmlSanitizer.SanitizeReader(input)
}

// AddComment adds new comment to post identified by postID and returns the comment object.
func (pm *PostManager) AddComment(postID bson.ObjectId, senderID string, body string, attachmentID UniversalID) *Comment {
	defer _Manager.Post.removeCache(postID)

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	// Define Default Values
	c := &Comment{
		ID:        bson.NewObjectId(),
		SenderID:  senderID,
		PostID:    postID,
		Body:      body,
		Removed:   false,
		Timestamp: Timestamp(),
	}
	if len(attachmentID) != 0 {
		c.Type = CommentTypeVoice
		c.AttachmentID = attachmentID
	} else {
		c.Type = CommentTypeText
	}

	if 0 == len(c.Body) {
		return nil
	}

	// Insert the new comment
	if err := db.C(global.CollectionPostsComments).Insert(c); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}

	// Update post's last_update and last-comments
	if err := db.C(global.CollectionPosts).UpdateId(c.PostID, bson.M{
		"$set": bson.M{"last_update": c.Timestamp},
		"$inc": bson.M{"counters.comments": 1},
		"$push": bson.M{
			"last-comments": bson.M{
				"$each": []bson.M{
					{
						"_id":           c.ID,
						"post_id":       c.PostID,
						"sender_id":     c.SenderID,
						"type":          c.Type,
						"attachment_id": c.AttachmentID,
						"text":          c.Body,
						"timestamp":     c.Timestamp,
					},
				},
				"$sort":  bson.M{"timestamp": 1},
				"$slice": -3,
			},
		},
	}); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}

	// Add User to the watcher list
	post := _Manager.Post.GetPostByID(postID)
	_Manager.Post.AddAccountToWatcherList(post.ID, c.SenderID)

	// Create the hook event and send it to the hooker
	for _, placeID := range post.PlaceIDs {
		_Manager.Hook.chEvents <- NewPostCommentEvent{
			PlaceID:   placeID,
			PostID:    post.ID,
			CommentID: c.ID,
			SenderID:  c.SenderID,
		}
	}

	// Increment Counter
	_Manager.Report.CountCommentAdd()
	_Manager.Report.CountCommentPerAccount(senderID)
	_Manager.Report.CountCommentPerPlace(post.PlaceIDs)

	if len(attachmentID) > 0 {
		_Manager.File.AddPostAsOwner(attachmentID, post.ID)
		_Manager.File.SetStatus(attachmentID, FileStatusAttached)
	}

	// Add Post Activity
	_Manager.PostActivity.CommentAdd(post.ID, c.SenderID, c.ID)

	return c
}

// AddPost creates a new post according to data provided by 'pcr'
func (pm *PostManager) AddPost(pcr PostCreateRequest) *Post {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	post := Post{}
	ts := Timestamp()
	post.Type = PostTypeNormal
	post.ReplyTo = pcr.ReplyTo
	post.ForwardFrom = pcr.ForwardFrom
	post.ContentType = pcr.ContentType
	post.SpamScore = pcr.SpamScore
	if post.SpamScore > global.DefaultSpamScore {
		post.Spam = true
	}

	// Returns nil if targets are more than DefaultPostMaxTargets
	if len(pcr.PlaceIDs)+len(pcr.Recipients) > global.DefaultPostMaxTargets {
		return nil
	}

	// Returns nil if number of attachments exceeds DefaultPostMaxAttachments
	if len(pcr.AttachmentIDs) > global.DefaultPostMaxAttachments {
		return nil
	}

	// Returns nil if number of labels exceeds DefaultPostMaxLabels
	if len(pcr.LabelIDs) > global.DefaultPostMaxLabels {
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

	var attachSize int64
	for _, uniID := range pcr.AttachmentIDs {
		_Manager.File.AddPostAsOwner(uniID, post.ID)
		_Manager.File.SetStatus(uniID, FileStatusAttached)
		f := _Manager.File.GetByID(uniID, nil)
		attachSize += f.Size
	}
	post.Counters.Size = attachSize

	// Increment Counters
	_Manager.Report.CountPostAdd()
	_Manager.Report.CountPostAttachCount(len(pcr.AttachmentIDs))
	_Manager.Report.CountPostAttachSize(attachSize)
	_Manager.Report.CountPostPerPlace(pcr.PlaceIDs)

	// fill email data
	post.EmailMetadata = pcr.EmailMetadata

	// fill post system data
	post.SystemData = pcr.SystemData

	// if post was an external post
	if _Manager.Account.Exists(pcr.SenderID) {
		post.Internal = true
	} else {
		post.Recipients = append(post.Recipients, post.SenderID)
		_Manager.Report.CountPostExternalAdd()
	}

	switch pcr.ContentType {
	case ContentTypeTextPlain:
		post.Content = post.Body
		if len(post.Body) > 256 {
			post.Ellipsis = true
			post.Preview = string(post.Body[:256])
		} else {
			post.Preview = post.Body
		}
	default:
		post.ContentType = ContentTypeTextHtml
		// clear body text from html elements
		p := strings.NewReader(pcr.Body)
		doc, _ := goquery.NewDocumentFromReader(p)
		doc.Find("").Each(func(i int, el *goquery.Selection) {
			el.Remove()
		})
		post.Content = doc.Text()
		post.Body = pm.sanitizeBody(strings.NewReader(pcr.Body), post.Internal).String()
		if len(pcr.Body) > 256 {
			post.Preview = pm.sanitizePreview(strings.NewReader(pcr.Body[:256])).String()
		} else {
			post.Preview = pm.sanitizePreview(strings.NewReader(pcr.Body)).String()
		}
		if len(post.Body) != len(post.Preview) || post.Preview != post.Body {
			post.Ellipsis = true
		}
	}

	if post.Spam {
		if err := db.C(global.CollectionPostsSpams).Insert(post); err != nil {
			log.Warn("got error inserting into spam collection", zap.Error(err))
			return nil
		}
	} else {
		if err := db.C(global.CollectionPosts).Insert(post); err != nil {
			log.Warn("got error inserting into posts collection", zap.Error(err))
			return nil
		}
		pm.postProcess(&post)
	}

	return &post
}
func (pm *PostManager) postProcess(post *Post) {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	// Update counters of the grand places
	grandParentIDs := _Manager.Place.GetGrandParentIDs(post.PlaceIDs)
	_Manager.Place.IncrementCounter(grandParentIDs, PlaceCounterQuota, int(post.Counters.Size))

	// Update counters of the places
	_Manager.Place.IncrementCounter(post.PlaceIDs, PlaceCounterPosts, 1)

	// Update user contacts list
	if post.Internal {
		_Manager.Account.UpdatePlaceConnection(post.SenderID, post.PlaceIDs, 1)
		_Manager.Account.UpdateRecipientConnection(post.SenderID, post.Recipients, 1)
		_Manager.Report.CountPostPerAccount(post.SenderID)
	}

	// Create PostRead items per each user of each place
	for _, placeID := range post.PlaceIDs {
		// Remove place from cache
		_Manager.Place.removeCache(placeID)

		place := _Manager.Place.GetByID(placeID, nil)
		grandPlace := place.GetGrandParent()
		if place.HasReadAccess(post.SenderID) {
			_Manager.Post.AddAccountToWatcherList(post.ID, post.SenderID)
		}

		// Set Post as UNREAD for all the members of the place except the sender
		var memberIDs []string
		bulk := db.C(global.CollectionPostsReads).Bulk()
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
			bulk.Insert(PostRead{
				AccountID: cid,
				PlaceID:   placeID,
				PostID:    post.ID,
				Timestamp: post.Timestamp,
			})
		}
		if _, err := bulk.Run(); err != nil {
			log.Warn("Got error", zap.Error(err))
		}

		// Update unread counters
		_, _ = db.C(global.CollectionPostsReadsCounters).UpdateAll(
			bson.M{
				"account_id": bson.M{"$ne": post.SenderID},
				"place_id":   placeID,
			},
			bson.M{"$inc": bson.M{"no_unreads": 1}},
		)

		// Create the hook event and send it to the hooker
		_Manager.Hook.chEvents <- NewPostEvent{
			PlaceID:          placeID,
			PostID:           post.ID,
			PostTitle:        post.Subject,
			AttachmentsCount: post.Counters.Attachments,
			SenderID:         post.SenderID,
		}

	}

	// Update label counters
	for _, labelID := range post.LabelIDs {
		_Manager.Label.IncrementCounter(labelID, "posts", 1)
	}

	// Add the timeline activity to the database
	_Manager.PlaceActivity.PostAdd(post.SenderID, post.PlaceIDs, post.ID)

}

// NotSpam remove the spam flag of the post and move it to the post collection.
func (pm *PostManager) NotSpam(postID bson.ObjectId) {
	post := pm.GetSpamPostByID(postID)
	if post == nil {
		return
	}

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	post.Spam = false
	_ = db.C(global.CollectionPostsSpams).RemoveId(postID)
	_ = db.C(global.CollectionPosts).Insert(post)
}

// AddAccountToWatcherList adds accountID to postID's watcher list of placeID
func (pm *PostManager) AddAccountToWatcherList(postID bson.ObjectId, accountID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if _, err := db.C(global.CollectionPostsWatchers).Upsert(
		bson.M{"_id": postID},
		bson.M{"$addToSet": bson.M{"accounts": accountID}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

// AttachPlace adds a new place to the post, and changes the last_update of the post
func (pm *PostManager) AttachPlace(postID bson.ObjectId, placeID, accountID string) bool {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	defer _Manager.Post.removeCache(postID)
	defer _Manager.Place.removeCache(placeID)
	post := _Manager.Post.GetPostByID(postID)
	place := _Manager.Place.GetByID(placeID, nil)

	// Update post's document
	// places and last_update fields are updated
	if err := db.C(global.CollectionPosts).UpdateId(
		postID,
		bson.M{
			"$addToSet": bson.M{"places": placeID},
			"$set": bson.M{
				"last_update": Timestamp(),
				"_removed":    false,
			},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}

	// Update counters of the attached place
	_Manager.Place.IncrementCounter([]string{placeID}, PlaceCounterPosts, 1)

	// Add new place to watcher list
	if place.HasReadAccess(post.SenderID) {
		_Manager.Post.AddAccountToWatcherList(postID, post.SenderID)
	}

	// Create Activities
	_Manager.PlaceActivity.PostAttachPlace(accountID, placeID, postID)
	_Manager.PostActivity.PlaceAttached(postID, accountID, placeID)

	return true
}

// AddRelatedTask adds a task to post
func (pm *PostManager) AddRelatedTask(postID, taskID bson.ObjectId) {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	defer _Manager.Post.removeCache(postID)
	if err := db.C(global.CollectionPosts).Update(
		bson.M{"_id": postID},
		bson.M{"$addToSet": bson.M{"related_tasks": taskID}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	if err := db.C(global.CollectionTasks).Update(
		bson.M{"_id": taskID},
		bson.M{"$set": bson.M{"related_post": postID}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}

}

// RemoveRelatedTask removes the task from the post
func (pm *PostManager) RemoveRelatedTask(postID, taskID bson.ObjectId) {

	defer _Manager.Post.removeCache(postID)

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionPosts).Update(
		bson.M{"_id": postID, "related_tasks": taskID},
		bson.M{"$pull": bson.M{"related_tasks": taskID}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	if err := db.C(global.CollectionTasks).Update(
		bson.M{"_id": taskID},
		bson.M{"$unset": bson.M{"related_post": postID}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
}

// CommentHasAccess if accountID has READ ACCESS to the postID of the commentID it returns TRUE
func (pm *PostManager) CommentHasAccess(commentID bson.ObjectId, accountID string) bool {
	if comment := pm.GetCommentByID(commentID); comment != nil {
		if pm.HasAccess(comment.PostID, accountID) {
			return true
		}
	}
	return false
}

// Exists returns true if post exists, and it is not deleted
func (pm *PostManager) Exists(postID bson.ObjectId) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n, _ := db.C(global.CollectionPosts).Find(bson.M{
		"_id":      postID,
		"_removed": false,
	}).Count()
	return n > 0
}

// GetPostByID returns Post by postID, if postID does not exist it returns nil
func (pm *PostManager) GetPostByID(postID bson.ObjectId) *Post {
	return _Manager.Post.readFromCache(postID)
}

// GetSpamPostByID returns Post by postID, if postID does not exist it returns nil
func (pm *PostManager) GetSpamPostByID(postID bson.ObjectId) *Post {
	post := &Post{}
	if err := _MongoDB.C(global.CollectionPostsSpams).FindId(postID).One(post); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}
	return post
}

// GetPostsByIDs returns an array of posts identified by postIDs, it returns an empty slice if nothing was found
func (pm *PostManager) GetPostsByIDs(postIDs []bson.ObjectId) []Post {
	return _Manager.Post.readMultiFromCache(postIDs)
}

// GetPostsByPlace returns an array of Posts that are in placeID
// sortItem could have any of these values:
//		PostSortLastUpdate
//		PostSortTimestamp
// this function is preferred to be called instead of GetPostsOfPlaces
func (pm *PostManager) GetPostsByPlace(placeID, sortItem string, pg Pagination) []Post {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	q := bson.M{"_removed": false}
	posts := make([]Post, 0, global.DefaultMaxResultLimit)
	switch sortItem {
	case PostSortLastUpdate, PostSortTimestamp:
	default:
		sortItem = PostSortTimestamp
	}
	sortDir := fmt.Sprintf("-%s", sortItem)
	q, sortDir = pg.FillQuery(q, sortItem, sortDir)

	q["places"] = placeID
	Q := db.C(global.CollectionPosts).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
	// Log Explain Query

	if err := Q.All(&posts); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return posts
}

// GetPostsBySender returns an array of Posts that are sent by accountID
// sortItem could have any of these values:
//		PostSortLastUpdate
//		PostSortTimestamp
func (pm *PostManager) GetPostsBySender(accountID, sortItem string, pg Pagination) []Post {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	switch sortItem {
	case PostSortLastUpdate, PostSortTimestamp:
	default:
		sortItem = PostSortTimestamp
	}
	sortDir := fmt.Sprintf("-%s", sortItem)
	q := bson.M{"_removed": false}
	q, sortDir = pg.FillQuery(q, sortItem, sortDir)

	if pg.GetLimit() == 0 || pg.GetLimit() > global.DefaultMaxResultLimit {
		pg.SetLimit(global.DefaultMaxResultLimit)
	}
	q["sender"] = accountID
	Q := db.C(global.CollectionPosts).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
	// Log Explain Query
	posts := make([]Post, 0, pg.GetLimit())
	Q.All(&posts)
	return posts
}

// GetPostsOfPlaces returns an array of Posts that are in any of the places
// sortItem could have any of these values:
//		PostSortLastUpdate
//		PostSortTimestamp
func (pm *PostManager) GetPostsOfPlaces(places []string, sortItem string, pg Pagination) []Post {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	switch sortItem {
	case PostSortLastUpdate, PostSortTimestamp:
	default:
		sortItem = PostSortTimestamp
	}
	sortDir := fmt.Sprintf("-%s", sortItem)

	q := bson.M{"_removed": false}
	posts := make([]Post, 0, pg.GetLimit())
	q, sortDir = pg.FillQuery(q, sortItem, sortDir)
	q["places"] = bson.M{"$in": places}
	Q := db.C(global.CollectionPosts).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
	// Log Explain Query

	Q.All(&posts)
	return posts
}

// GetSpamPostsOfPlaces returns an array of Posts that are in any of the places
// sortItem could have any of these values:
//		PostSortLastUpdate
//		PostSortTimestamp
func (pm *PostManager) GetSpamPostsOfPlaces(places []string, pg Pagination) []Post {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	posts := make([]Post, 0, pg.GetLimit())
	q := bson.M{
		"places": bson.M{"$in": places},
	}

	Q := db.C(global.CollectionPosts).Find(q).Skip(pg.GetSkip()).Limit(pg.GetLimit())
	_ = Q.All(&posts)
	return posts
}

// GetPostWatchers returns an array of accountIDs who listen to post notifications
func (pm *PostManager) GetPostWatchers(postID bson.ObjectId) []string {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	watchers := struct {
		PostID   bson.ObjectId `bson:"_id"`
		Accounts []string      `bson:"accounts"`
	}{}
	if err := db.C(global.CollectionPostsWatchers).Find(bson.M{"_id": postID}).One(&watchers); err != nil {
		log.Warn("Got error", zap.Error(err))
		return []string{}
	}
	return watchers.Accounts
}

// GetUnreadPostsByPlace returns an array of Posts that are unseen/unread by accountID and exists in placeID
// if subPlaces set to TRUE then it also returns unseen/unread posts in any sub-places of placeID
// that accountID is member of.
// return an empty slice if there is no unseen/unread post
func (pm *PostManager) GetUnreadPostsByPlace(placeID, accountID string, subPlaces bool, pg Pagination) []Post {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	sortItem := PostSortTimestamp
	sortDir := fmt.Sprintf("-%s", sortItem)
	// Match query
	mq := bson.M{
		"account_id": accountID,
		"read":       false,
	}
	if subPlaces {
		mq["place_id"] = bson.M{"$regex": fmt.Sprintf("^%s\\b", placeID)}
	} else {
		mq["place_id"] = placeID
	}
	mq, sortDir = pg.FillQuery(mq, sortItem, sortDir)

	Q := db.C(global.CollectionPostsReads).Find(mq).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
	// Log Explain Query
	iter := Q.Iter()
	defer iter.Close()
	readItem := tools.M{}
	posts := make([]Post, 0, pg.GetLimit())
	for iter.Next(&readItem) {
		if post := _Manager.Post.GetPostByID(readItem["post_id"].(bson.ObjectId)); post != nil {
			posts = append(posts, *post)
		}
	}
	return posts
}

// GetAccountsWhoReadThis returns a list of members who have read this post
func (pm *PostManager) GetAccountsWhoReadThis(postID bson.ObjectId, pg Pagination) []PostRead {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	pr := make([]PostRead, 0, pg.GetLimit())
	Q := db.C(global.CollectionPostsReadsAccounts).Find(bson.M{"post_id": postID}).Skip(pg.GetSkip()).Limit(pg.GetLimit())
	// Log Explain Query
	if err := Q.All(&pr); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return pr
}

// GetCommentByID returns Comment by the given commentID and return nil if commentID does not exists
func (pm *PostManager) GetCommentByID(commentID bson.ObjectId) *Comment {
	comment := _Manager.Post.readCommentFromCache(commentID)
	if comment != nil {
		return comment
	}
	return nil
}

//	GetCommentsByIDs returns an array of comments identified by commentIDs. If some comments were not
//	found then they will be ignored silently. Caller may compare the length of result with length of
//	input to detect if any comment was missing
func (pm *PostManager) GetCommentsByIDs(commentIDs []bson.ObjectId) []Comment {
	comments := _Manager.Post.readMultiCommentsFromCache(commentIDs)
	return comments

}

// GetCommentsByPostID returns an array of Comments of postID, if postID has no comments then it returns an empty slice.
func (pm *PostManager) GetCommentsByPostID(postID bson.ObjectId, pg Pagination) []Comment {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	q := bson.M{
		"post_id":  postID,
		"_removed": false,
	}
	sortItem := PostSortTimestamp
	sortDir := fmt.Sprintf("-%s", sortItem)
	q, sortDir = pg.FillQuery(q, sortItem, sortDir)
	Q := db.C(global.CollectionPostsComments).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
	// Log Explain Query
	res := make([]Comment, 0)
	Q.All(&res)
	return res
}

// GetPinnedPosts returns an array of Posts which are pinned by accountID
func (pm *PostManager) GetPinnedPosts(accountID string, pg Pagination) []Post {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	q := bson.M{
		"account_id": accountID,
	}
	sortItem := PostSortPinTime
	sortDir := fmt.Sprintf("-%s", sortItem)
	q, sortDir = pg.FillQuery(q, sortItem, sortDir)

	Q := db.C(global.CollectionAccountsPosts).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
	// Log Explain Query
	iter := Q.Iter()
	item := tools.M{}
	posts := make([]Post, 0, pg.GetLimit())
	for iter.Next(item) {
		postID := item["post_id"].(bson.ObjectId)
		if post := _Manager.Post.GetPostByID(postID); post != nil {
			if post.HasAccess(accountID) {
				posts = append(posts, *post)
			} else {
				_Manager.Post.UnpinPost(accountID, postID)
			}
		}
	}
	return posts

}

// HasBeenReadBy returns TRUE if postID has been seen/read by accountID
func (pm *PostManager) HasBeenReadBy(postID bson.ObjectId, accountID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	q := bson.M{
		"post_id":    postID,
		"account_id": accountID,
		"read":       false,
	}
	Q := db.C(global.CollectionPostsReads).Find(q)
	// Log Explain Query
	if n, _ := Q.Count(); n > 0 {
		return false
	}
	return true
}

// HasBeenWatchedBy returns TRUE if postID has accountID in its watchers list
func (pm *PostManager) HasBeenWatchedBy(postID bson.ObjectId, accountID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	q := bson.M{
		"_id":      postID,
		"accounts": accountID,
	}
	Q := db.C(global.CollectionPostsWatchers).Find(q)
	// Log Explain Query
	if n, _ := Q.Count(); n > 0 {
		return true
	}
	return false
}

// HasAccess checks if accountID has READ ACCESS to any of the postID places then it returns TRUE otherwise
// it returns FALSE
func (pm *PostManager) HasAccess(postID bson.ObjectId, accountID string) bool {
	if post := pm.GetPostByID(postID); post != nil {
		for _, placeID := range post.PlaceIDs {
			if placeID == "*" {
				return true
			}
			place := _Manager.Place.GetByID(placeID, nil)
			if place.HasReadAccess(accountID) {
				return true
			}
		}
	}
	return false
}

// HideComment acts like RemoveComment but it does not actually removes the comment and is undoable.
// it does not remove the time-line activity of the post, only hides the comments and
// identifies accountID as the remover of the commentID
func (pm *PostManager) HideComment(commentID bson.ObjectId, accountID string) bool {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	c := _Manager.Post.GetCommentByID(commentID)
	if c == nil {
		return false
	}
	if err := db.C(global.CollectionPostsComments).Update(
		bson.M{
			"_id":        commentID,
			"removed_by": bson.M{"$exists": false},
		},
		bson.M{
			"$set": bson.M{
				"old_text":   c.Body,
				"removed_by": accountID,
				"text":       "",
			},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}

	lcms := make([]bson.M, 0, 3)
	iter := db.C(global.CollectionPostsComments).Find(bson.M{
		"post_id":  c.PostID,
		"txt":      bson.M{"$ne": ""},
		"_removed": false,
	}).Sort("-_id").Limit(3).Iter()

	for iter.Next(&c) {
		lcms = append(lcms, bson.M{
			"_id":       c.ID,
			"sender_id": c.SenderID,
			"post_id":   c.PostID,
			"text":      c.Body,
			"timestamp": c.Timestamp,
		})
	}

	db.C(global.CollectionPosts).UpdateId(
		c.PostID,
		bson.M{
			"$inc": bson.M{"counters.comments": -1},
			"$set": bson.M{"last-comments": lcms},
		},
	)

	return true
}

// MarkAsRead marks the postID as seen/read by accountID
func (pm *PostManager) MarkAsRead(postID bson.ObjectId, accountID string) bool {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	post := pm.GetPostByID(postID)

	if ci, err := db.C(global.CollectionPostsReads).RemoveAll(
		bson.M{
			"account_id": accountID,
			"post_id":    postID,
			"read":       false,
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	} else {
		// Add the accountID to the list of readers if he/she is not already exists
		// This function runs in background
		q := bson.M{
			"post_id":    postID,
			"account_id": accountID,
		}
		Q := db.C(global.CollectionPostsReadsAccounts).Find(q)
		if n, _ := Q.Count(); n == 0 {
			db.C(global.CollectionPostsReadsAccounts).Insert(
				bson.M{
					"post_id":    postID,
					"account_id": accountID,
					"timestamp":  ts,
				},
			)
		}

		if ci.Removed > 0 {
			db.C(global.CollectionPostsReadsCounters).UpdateAll(
				bson.M{
					"account_id": accountID,
					"place_id":   bson.M{"$in": post.PlaceIDs},
					"no_unreads": bson.M{"$gt": 0},
				},
				bson.M{"$inc": bson.M{"no_unreads": -1}},
			)
			return true
		}
	}
	return false
}

// MarkAsReadByPlace marks all the posts in the placeID as seen/read by accountID
func (pm *PostManager) MarkAsReadByPlace(placeID, accountID string) bool {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	db.C(global.CollectionPostsReads).RemoveAll(
		bson.M{
			"account_id": accountID,
			"place_id":   placeID,
			"read":       false,
		},
	)

	// Reset unread counter for the placeID
	db.C(global.CollectionPostsReadsCounters).UpdateAll(
		bson.M{
			"account_id": accountID,
			"place_id":   placeID,
			"no_unreads": bson.M{"$gt": 0},
		},
		bson.M{"$set": bson.M{"no_unreads": 0}},
	)
	return true
}

// Move moves post from one place to another place
func (pm *PostManager) Move(postID bson.ObjectId, oldPlaceID, newPlaceID, accountID string) bool {
	defer _Manager.Post.removeCache(postID)
	defer _Manager.Place.removeCache(oldPlaceID)
	defer _Manager.Place.removeCache(newPlaceID)

	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	// Update POSTS collection
	// Add the new place first and then remove the old place
	if err := db.C(global.CollectionPosts).UpdateId(
		postID,
		bson.M{
			"$addToSet": bson.M{"places": newPlaceID},
			"$set": bson.M{
				"last_update": Timestamp(),
				"_removed":    false,
			},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	if err := db.C(global.CollectionPosts).UpdateId(
		postID,
		bson.M{
			"$pull":     bson.M{"places": oldPlaceID},
			"$addToSet": bson.M{"removed_places": oldPlaceID},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}

	// Update PLACES collection
	if err := db.C(global.CollectionPlaces).Update(
		bson.M{"_id": newPlaceID},
		bson.M{"$inc": bson.M{"counters.posts": 1}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	if err := db.C(global.CollectionPlaces).Update(
		bson.M{"_id": oldPlaceID},
		bson.M{"$inc": bson.M{"counters.posts": -1}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}

	pr := new(PostRead)
	iter := db.C(global.CollectionPostsReads).Find(
		bson.M{
			"post_id":  postID,
			"place_id": oldPlaceID,
			"read":     false,
			"_removed": false,
		}).Select(bson.M{"account_id": 1}).Iter()
	defer iter.Close()
	for iter.Next(pr) {
		db.C(global.CollectionPostsReadsCounters).Update(
			bson.M{
				"account_id": pr.AccountID,
				"place_id":   oldPlaceID,
				"no_unreads": bson.M{"$gt": 0},
			},
			bson.M{"$inc": bson.M{"no_unreads": -1}},
		)
	}
	// remove all unreads from oldPlace
	db.C(global.CollectionPostsReads).RemoveAll(
		bson.M{"post_id": postID, "place_id": oldPlaceID},
	)

	// Update timeline
	_Manager.PlaceActivity.PostMove(accountID, oldPlaceID, newPlaceID, postID)
	_Manager.PostActivity.PlaceMove(postID, accountID, oldPlaceID, newPlaceID)
	return true
}

// BookmarkPost adds postID to the pinned posts list of the accountID
func (pm *PostManager) BookmarkPost(accountID string, postID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if post := _Manager.Post.GetPostByID(postID); post != nil {
		if _, err := db.C(global.CollectionAccountsPosts).Upsert(
			bson.M{"account_id": accountID, "post_id": postID},
			bson.M{"$set": bson.M{
				"pin_time": Timestamp(),
			}},
		); err != nil {
			log.Warn("Got error", zap.Error(err))
		}
	}
}

// Remove removes the postID from the placeID.
// if placeID is the last place that postID are in, then removes the comments of the postID too.
func (pm *PostManager) Remove(accountID string, postID bson.ObjectId, placeID string) bool {
	defer _Manager.Post.removeCache(postID)
	defer _Manager.Place.removeCache(placeID)

	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	post := pm.GetPostByID(postID)
	if post == nil {
		return false
	}
	if len(post.PlaceIDs) == 0 {
		return false
	}
	if !post.IsInPlace(placeID) {
		log.Warn("Incorrect Call to PostManager::Remove")
		return false
	}

	// Update posts collection
	// if it is the last place of the post
	if len(post.PlaceIDs) == 1 {
		if err := db.C(global.CollectionPosts).Update(
			bson.M{"_id": postID},
			bson.M{
				"$pull":     bson.M{"places": placeID},
				"$addToSet": bson.M{"removed_places": placeID},
				"$set":      bson.M{"_removed": true},
			},
		); err != nil {
			log.Warn("Got error", zap.Error(err))
		}
	} else {
		if err := db.C(global.CollectionPosts).Update(
			bson.M{"_id": postID},
			bson.M{
				"$pull":     bson.M{"places": placeID},
				"$addToSet": bson.M{"removed_places": placeID},
			},
		); err != nil {
			log.Warn("Got error", zap.Error(err))
		}
	}

	// Update place counter
	_Manager.Place.IncrementCounter([]string{placeID}, PlaceCounterPosts, -1)

	pr := new(PostRead)
	iter := db.C(global.CollectionPostsReads).Find(
		bson.M{
			"post_id":  postID,
			"place_id": placeID,
			"read":     false,
			"_removed": false,
		}).Select(bson.M{"account_id": 1}).Iter()
	defer iter.Close()
	for iter.Next(pr) {
		db.C(global.CollectionPostsReadsCounters).Update(
			bson.M{
				"account_id": pr.AccountID,
				"place_id":   placeID,
				"no_unreads": bson.M{"$gt": 0},
			},
			bson.M{"$inc": bson.M{"no_unreads": -1}},
		)
	}
	db.C(global.CollectionPostsReads).RemoveAll(
		bson.M{"post_id": postID, "place_id": placeID},
	)

	// Update timeline items
	_Manager.PlaceActivity.PostRemove(accountID, placeID, postID)

	return true
}

// RemoveByPlaceID removes all the posts from the placeID.
// if placeID is the last place that postID are in, then removes the comments of the postID too.
func (pm *PostManager) RemoveByPlaceID(accountID string, placeID string) bool {
	defer _Manager.Place.removeCache(placeID)

	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	_, err := db.C(global.CollectionPosts).UpdateAll(
		bson.M{"places": placeID},
		bson.M{
			"$pull":     bson.M{"places": placeID},
			"$addToSet": bson.M{"removed_places": placeID},
		},
	)
	if err != nil {
		return false
	}

	// Update place counter
	_Manager.Place.SetCounter([]string{placeID}, PlaceCounterPosts, 0)

	_, _ = db.C(global.CollectionPostsReadsCounters).UpdateAll(
		bson.M{
			"place_id":   placeID,
			"no_unreads": bson.M{"$gt": 0},
		},
		bson.M{"$set": bson.M{"no_unreads": -1}},
	)

	// Update timeline items
	_Manager.PlaceActivity.PostRemoveAll(accountID, placeID)

	return true
}

// RemoveComment removes the commentID from its post.
// also removes the time-line activity of the comment
func (pm *PostManager) RemoveComment(accountID string, commentID bson.ObjectId) bool {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ch := mgo.Change{
		Update: bson.M{"$set": bson.M{"_removed": true}},
	}
	c := new(Comment)
	if ci, err := db.C(global.CollectionPostsComments).FindId(commentID).Apply(ch, c); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	} else {
		if ci.Updated == 0 {
			return false
		}
	}
	post := _Manager.Post.GetPostByID(c.PostID)
	if len(c.AttachmentID) > 0 {
		_Manager.File.RemovePostAsOwner(c.AttachmentID, c.PostID)
	}

	defer _Manager.Post.removeCommentFromCache(commentID)
	defer _Manager.Post.removeCache(c.PostID)
	lcms := make([]bson.M, 0, 3)
	iter := db.C(global.CollectionPostsComments).Find(bson.M{
		"post_id":  c.PostID,
		"_removed": false,
	}).Sort("-_id").Limit(3).Iter()
	defer iter.Close()

	for iter.Next(&c) {
		lcms = append(lcms, bson.M{
			"_id":       c.ID,
			"sender_id": c.SenderID,
			"post_id":   c.PostID,
			"text":      c.Body,
			"timestamp": c.Timestamp,
		})
	}

	db.C(global.CollectionPosts).UpdateId(
		c.PostID,
		bson.M{
			"$inc": bson.M{"counters.comments": -1},
			"$set": bson.M{"last-comments": lcms},
		},
	)

	// Add Post Activity
	_Manager.PostActivity.CommentRemove(post.ID, accountID, commentID)

	return true
}

// RemoveAccountFromWatcherList removes accountID from postID's watchers list of the placeID
func (pm *PostManager) RemoveAccountFromWatcherList(postID bson.ObjectId, accountID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionPostsWatchers).Update(
		bson.M{"_id": postID},
		bson.M{"$pull": bson.M{"accounts": accountID}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

// SetEmailMessageID set MessageID for the post, this function will be used by Gobryas service
func (pm *PostManager) SetEmailMessageID(postID bson.ObjectId, messageID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionPosts).UpdateId(
		postID,
		bson.M{
			"$set": bson.M{"email_meta.message_id": messageID},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return true
}

// UnpinPost removes the postID from accountID's pinned posts list
func (pm *PostManager) UnpinPost(accountID string, postID bson.ObjectId) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionAccountsPosts).Remove(bson.M{
		"post_id":    postID,
		"account_id": accountID,
	}); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
}

// IsPinned returns true if postID has been pinned by accountID
func (pm *PostManager) IsPinned(accountID string, postID bson.ObjectId) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if n, err := db.C(global.CollectionAccountsPosts).Find(bson.M{
		"account_id": accountID, "post_id": postID,
	}).Count(); err != nil {
		log.Warn("Got error", zap.Error(err))
	} else if n > 0 {
		return true
	}
	return false
}

type CommentType int
type Comment struct {
	ID           bson.ObjectId `json:"_id" bson:"_id"`
	Type         CommentType   `json:"type" bson:"type"`
	SenderID     string        `json:"sender_id" bson:"sender_id"`
	PostID       bson.ObjectId `json:"post_id,omitempty" bson:"post_id"`
	Body         string        `json:"text" bson:"text"`
	Timestamp    uint64        `json:"timestamp" bson:"timestamp"`
	AttachmentID UniversalID   `json:"attachment_id,omitempty" bson:"attachment_id"`
	Removed      bool          `json:"_removed,omitempty" bson:"_removed"`
	RemovedBy    string        `json:"removed_by,omitempty" bson:"removed_by,omitempty"`
}
type PostCreateRequest struct {
	ID              bson.ObjectId  `json:"_id"`
	SenderID        string         `json:"sender"`
	PlaceIDs        []string       `json:"places"`
	Recipients      []string       `json:"recipients"`
	LabelIDs        []string       `json:"labels"`
	Subject         string         `json:"subject"`
	ContentType     string         `json:"content_type"`
	Body            string         `json:"body"`
	IFrameUrl       string         `json:"iframe_url,omitempty"`
	ReplyTo         bson.ObjectId  `json:"reply_to,omitempty"`
	ForwardFrom     bson.ObjectId  `json:"forward_from,omitempty"`
	AttachmentIDs   []UniversalID  `json:"attaches"`
	AttachmentSizes []int64        `json:"attaches_size"`
	EmailMetadata   EmailMetadata  `json:"email_meta"`
	SystemData      PostSystemData `json:"system_data"`
	SpamScore       float64        `json:"spam_score"`
}
type Post struct {
	ID              bson.ObjectId   `json:"_id" bson:"_id"`
	Type            int             `json:"type" bson:"type"`
	SenderID        string          `json:"sender" bson:"sender"`
	PlaceIDs        []string        `json:"places" bson:"places"`
	Recipients      []string        `json:"recipients" bson:"recipients"`
	LabelIDs        []string        `json:"labels" bson:"labels"`
	Subject         string          `json:"subject" bson:"subject"`
	ContentType     string          `json:"content_type" bson:"content_type"`
	Body            string          `json:"body" bson:"body"`
	Content         string          `json:"content" bson:"content"`
	Preview         string          `json:"preview" bson:"preview"`
	ReplyTo         bson.ObjectId   `json:"reply_to,omitempty" bson:"reply_to,omitempty"`
	ForwardFrom     bson.ObjectId   `json:"forward_from,omitempty" bson:"forward_from,omitempty"`
	AttachmentIDs   []UniversalID   `json:"attaches" bson:"attaches"`
	AttachmentSizes []int64         `json:"attaches_size" bson:"-"`
	RelatedTasks    []bson.ObjectId `json:"related_tasks,omitempty" bson:"related_tasks,omitempty"`
	IFrameUrl       string          `json:"iframe_url,omitempty" bson:"iframe_url,omitempty"`
	Timestamp       uint64          `json:"timestamp" bson:"timestamp"`
	LastUpdate      uint64          `json:"last_update" bson:"last_update"`
	SpamScore       float64         `json:"spam_score" bson:"spam_score"`
	Spam            bool            `json:"spam" bson:"spam"`
	Internal        bool            `json:"internal" bson:"internal"`
	Ellipsis        bool            `json:"ellipsis" bson:"ellipsis"`
	Counters        PostCounters    `json:"counters" bson:"counters"`
	RecentComments  []Comment       `json:"last-comments" bson:"last-comments"`
	EmailMetadata   EmailMetadata   `json:"email_meta" bson:"email_meta"`
	SystemData      PostSystemData  `json:"system_data" bson:"system_data"`
	Archived        bool            `json:"archived" bson:"archived"`
	Removed         bool            `json:"_removed" bson:"_removed"`
}
type PostCounters struct {
	Attachments int   `json:"attaches" bson:"attaches"`
	Comments    int   `json:"comments" bson:"comments"`
	Replied     int   `json:"replied" bson:"replied"`
	Forwarded   int   `json:"forwarded" bson:"forwarded"`
	Size        int64 `json:"size" bson:"size"`
	Labels      int   `json:"labels" bson:"labels"`
}
type PostRead struct {
	ID        bson.ObjectId `bson:"_id,omitempty"`
	AccountID string        `json:"account_id" bson:"account_id"`
	PlaceID   string        `json:"place_id" bson:"place_id"`
	PostID    bson.ObjectId `json:"post_id" bson:"post_id"`
	Read      bool          `json:"read" bson:"read"`
	Timestamp uint64        `json:"timestamp" bson:"timestamp"`
	ReadOn    uint64        `json:"read_on,omitempty" bson:"read_on,omitempty"`
	Removed   bool          `json:"_removed" bson:"_removed"`
}
type EmailMetadata struct {
	Name           string      `json:"name" bson:"name"`
	MessageID      string      `json:"message_id" bson:"message_id"`
	InReplyTo      string      `json:"in_reply_to" bson:"in_reply_to"`
	ReplyTo        string      `json:"reply_to" bson:"reply_to"`
	Picture        Picture     `json:"picture" bson:"picture"`
	RawMessageFile UniversalID `json:"raw_msg_id" bson:"raw_msg_id"`
}
type PostSystemData struct {
	CopyFrom  bson.ObjectId `json:"copy_from" bson:"copy_from,omitempty"`
	Copier    string        `json:"copier" bson:"copier"`
	NoComment bool          `json:"no_comment" bson:"no_comment"`
}

func (p *Post) IsInPlace(placeID string) bool {
	for _, v := range p.PlaceIDs {
		if v == placeID {
			return true
		}
	}
	return false
}

// HasAccess checks if accountID has access to the post, if he/she has access it returns TRUE otherwise FALSE
func (p *Post) HasAccess(accountID string) bool {
	for _, placeID := range p.PlaceIDs {
		if placeID == "*" {
			return true
		}
		place := _Manager.Place.GetByID(placeID, nil)
		if place.HasReadAccess(accountID) {
			return true
		}
	}

	return false
}

// AddLabel add new label to the post
func (p *Post) AddLabel(accountID, labelID string) bool {
	defer _Manager.Post.removeCache(p.ID)

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionPosts).UpdateId(
		p.ID,
		bson.M{
			"$addToSet": bson.M{"labels": labelID},
			"$inc":      bson.M{"counters.labels": 1},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}

	// Increment label counter
	_Manager.Label.IncrementCounter(labelID, "posts", 1)

	// Add Post Activity
	_Manager.PostActivity.LabelAdd(p.ID, accountID, labelID)

	return true
}

// RemoveLabel removes label off the post
func (p *Post) RemoveLabel(accountID, labelID string) bool {
	defer _Manager.Post.removeCache(p.ID)

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionPosts).UpdateId(
		p.ID,
		bson.M{
			"$pull": bson.M{"labels": labelID},
			"$inc":  bson.M{"counters.labels": -1},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}

	// Increment label counter
	_Manager.Label.IncrementCounter(labelID, "posts", -1)

	_Manager.PostActivity.LabelRemove(p.ID, accountID, labelID)

	return true
}

// MarkAsRead marks the postID as seen/read by accountID
func (p *Post) MarkAsRead(accountID string) bool {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	ts := Timestamp()
	if accountID == p.SenderID {
		return false
	}
	if ci, err := db.C(global.CollectionPostsReads).RemoveAll(
		bson.M{
			"account_id": accountID,
			"post_id":    p.ID,
			"read":       false,
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	} else {
		// Add the accountID to the list of readers if he/she is not already exists
		// This function runs in background
		q := bson.M{
			"post_id":    p.ID,
			"account_id": accountID,
		}
		Q := db.C(global.CollectionPostsReadsAccounts).Find(q)
		if n, _ := Q.Count(); n == 0 {
			db.C(global.CollectionPostsReadsAccounts).Insert(
				bson.M{
					"post_id":    p.ID,
					"account_id": accountID,
					"timestamp":  ts,
				},
			)
		}

		if ci.Removed > 0 {
			db.C(global.CollectionPostsReadsCounters).UpdateAll(
				bson.M{
					"account_id": accountID,
					"place_id":   bson.M{"$in": p.PlaceIDs},
					"no_unreads": bson.M{"$gt": 0},
				},
				bson.M{"$inc": bson.M{"no_unreads": -1}},
			)
			return true
		}
	}
	return false
}

// Update updates the post
func (p *Post) Update(postSubject, postBody string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	defer _Manager.Post.removeCache(p.ID)

	var postPreview, postContent string
	var ellipsis bool
	switch p.ContentType {
	case ContentTypeTextPlain:
		postContent = postBody
		if len(postBody) > 256 {
			ellipsis = true
			postPreview = string(postBody[:256])
		} else {
			postPreview = postBody
		}
	case ContentTypeTextHtml:
		reader := strings.NewReader(postBody)
		doc, _ := goquery.NewDocumentFromReader(reader)
		doc.Find("").Each(func(i int, el *goquery.Selection) {
			el.Remove()
		})
		postContent = doc.Text()
		postBody = _Manager.Post.sanitizeBody(strings.NewReader(postBody), p.Internal).String()
		if len(postBody) > 256 {
			postPreview = _Manager.Post.sanitizePreview(strings.NewReader(postBody[:256])).String()
		} else {
			postPreview = _Manager.Post.sanitizePreview(strings.NewReader(postBody)).String()
		}
		if len(postBody) != len(postPreview) || postPreview != postBody {
			ellipsis = true
		}
	default:
		return false
	}

	if err := db.C(global.CollectionPosts).UpdateId(
		p.ID,
		bson.M{
			"$set": bson.M{
				"body":     postBody,
				"subject":  postSubject,
				"preview":  postPreview,
				"ellipsis": ellipsis,
				"content":  postContent,
			},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}

	// Set Post activity
	_Manager.PostActivity.Edit(p.ID, p.SenderID)

	return true

}
