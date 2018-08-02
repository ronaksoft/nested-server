package nested

import (
    "bytes"
    "encoding/gob"
    "github.com/globalsign/mgo"
    "github.com/globalsign/mgo/bson"
    "github.com/microcosm-cc/bluemonday"
    "io"
    "fmt"
    "strings"
    "github.com/gomodule/redigo/redis"
)

const (
    POST_TYPE_NORMAL = 0x02
)
const (
    CONTENT_TYPE_TEXT_HTML  = "text/html"
    CONTENT_TYPE_TEXT_PLAIN = "text/plain"
)
const (
    POST_SORT_TIMESTAMP   = "timestamp"
    POST_SORT_LAST_UPDATE = "last_update"
    POST_SORT_PIN_TIME    = "pin_time"
)
const (
    COMMENT_TYPE_TEXT     CommentType = 0x00
    COMMENT_TYPE_VOICE    CommentType = 0x01
    COMMENT_TYPE_ACTIVITY CommentType = 0x02
)

// Post Manager and Methods
type PostManager struct{}

func NewPostManager() *PostManager {
    return new(PostManager)
}

func (pm *PostManager) readFromCache(postID bson.ObjectId) *Post {
    _funcName := "PostManager::readFromCache"
    _Log.FunctionStarted(_funcName, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    post := new(Post)
    c := _Cache.Pool.Get()
    defer c.Close()
    keyID := fmt.Sprintf("post:gob:%s", postID.Hex())
    if gobPost, err := redis.Bytes(c.Do("GET", keyID)); err != nil {
        if err := _MongoDB.C(COLLECTION_POSTS).FindId(postID).One(post); err != nil {
            _Log.Error(_funcName, err.Error(), postID.Hex())
            return nil
        }
        gobPost := new(bytes.Buffer)
        if err := gob.NewEncoder(gobPost).Encode(post); err == nil {
            c.Do("SETEX", keyID, CACHE_LIFETIME, gobPost.Bytes())
        }
        return post
    } else if err := gob.NewDecoder(bytes.NewBuffer(gobPost)).Decode(post); err == nil {
        return post
    }
    return nil
}

func (pm *PostManager) readCommentFromCache(commentID bson.ObjectId) *Comment {
    _funcName := "PostManager::readCommentFromCache"
    _Log.FunctionStarted(_funcName, commentID.Hex())
    defer _Log.FunctionFinished(_funcName)

    comment := new(Comment)
    c := _Cache.Pool.Get()
    defer c.Close()
    keyID := fmt.Sprintf("comment:gob:%s", commentID.Hex())
    if gobComment, err := redis.Bytes(c.Do("GET", keyID)); err != nil {
        if err := _MongoDB.C(COLLECTION_POSTS_COMMENTS).FindId(commentID).One(comment); err != nil {
            _Log.Error(_funcName, err.Error(), commentID.Hex())
            return nil
        }
        gobComment := new(bytes.Buffer)
        if err := gob.NewDecoder(gobComment).Decode(comment); err == nil {
            c.Do("SETEX", keyID, CACHE_LIFETIME, gobComment.Bytes())
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
    _HtmlSanitizer.AllowAttrs("dir", "align", "style","border", "height", "max-height", "hspace", "usemap", "vspace", "width", "max-width").Globally()
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
    _funcName := "PostManager::AddComment"
    _Log.FunctionStarted(_funcName, postID.Hex(), senderID)
    defer _Log.FunctionFinished(_funcName)
    defer _Manager.Post.removeCache(postID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
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
        c.Type = COMMENT_TYPE_VOICE
        c.AttachmentID = attachmentID
    } else {
        c.Type = COMMENT_TYPE_TEXT
    }

    if 0 == len(c.Body) {
        return nil
    }

    // Insert the new comment
    if err := db.C(COLLECTION_POSTS_COMMENTS).Insert(c); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil
    }

    // Update post's last_update and last-comments
    if err := db.C(COLLECTION_POSTS).UpdateId(c.PostID, bson.M{
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
                "$slice": -3,
            },
        },
    }); err != nil {
        _Log.Error(_funcName, err.Error())
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
        _Manager.File.SetStatus(attachmentID, FILE_STATUS_ATTACHED)
    }

    // Add Post Activity
    _Manager.PostActivity.CommentAdd(post.ID, c.SenderID, c.ID)

    return c
}

// AddPost creates a new post according with data provided by 'pcr'
func (pm *PostManager) AddPost(pcr PostCreateRequest) *Post {
    _funcName := "PostManager::AddPost"
    _Log.FunctionStarted(_funcName)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    post := Post{}
    ts := Timestamp()
    post.Type = POST_TYPE_NORMAL
    post.ReplyTo = pcr.ReplyTo
    post.ForwardFrom = pcr.ForwardFrom
    post.ContentType = pcr.ContentType

    // Returns nil if targets are more than DEFAULT_POST_MAX_TARGETS
    if len(pcr.PlaceIDs)+len(pcr.Recipients) > DEFAULT_POST_MAX_TARGETS {
        return nil
    }

    // Returns nil if number of attachments exceeds DEFAULT_POST_MAX_ATTACHMENTS
    if len(pcr.AttachmentIDs) > DEFAULT_POST_MAX_ATTACHMENTS {
        return nil
    }

    // Returns nil if number of labels exceeds DEFAULT_POST_MAX_LABELS
    if len(pcr.LabelIDs) > DEFAULT_POST_MAX_LABELS {
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
        _Manager.File.AddPostAsOwner(uniID, post.ID)
        _Manager.File.SetStatus(uniID, FILE_STATUS_ATTACHED)
        f := _Manager.File.GetByID(uniID, nil)
        attach_size += f.Size
    }
    post.Counters.Size = attach_size

    // Increment Counters
    _Manager.Report.CountPostAdd()
    _Manager.Report.CountPostAttachCount(len(pcr.AttachmentIDs))
    _Manager.Report.CountPostAttachSize(attach_size)
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
    case CONTENT_TYPE_TEXT_PLAIN:
        if len(post.Body) > 256 {
            post.Ellipsis = true
            post.Preview = string(post.Body[:256])
        } else {
            post.Preview = post.Body
        }
    default:
        post.ContentType = CONTENT_TYPE_TEXT_HTML
        strings.NewReader(pcr.Body)
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


    if err := db.C(COLLECTION_POSTS).Insert(post); err != nil {
        _Log.Error(_funcName, err.Error(), post)
        return nil
    }

    // Update counters of the grand places
    grandParentIDs := _Manager.Place.GetGrandParentIDs(pcr.PlaceIDs)
    _Manager.Place.IncrementCounter(grandParentIDs, PLACE_COUNTER_QUOTA, int(post.Counters.Size))

    // Update counters of the places
    _Manager.Place.IncrementCounter(pcr.PlaceIDs, PLACE_COUNTER_POSTS, 1)

    // Update user contacts list
    if post.Internal {
        _Manager.Account.UpdatePlaceConnection(pcr.SenderID, pcr.PlaceIDs, 1)
        _Manager.Account.UpdateRecipientConnection(pcr.SenderID, pcr.Recipients, 1)
        _Manager.Report.CountPostPerAccount(pcr.SenderID)
    }

    // Create PostRead items per each user of each place
    for _, placeID := range pcr.PlaceIDs {
        // Remove place from cache
        _Manager.Place.removeCache(placeID)

        place := _Manager.Place.GetByID(placeID, nil)
        grandPlace := place.GetGrandParent()
        if place.HasReadAccess(post.SenderID) {
            _Manager.Post.AddAccountToWatcherList(post.ID, post.SenderID)
        }

        // Set Post as UNREAD for all the members of the place except the sender
        var memberIDs []string
        bulk := db.C(COLLECTION_POSTS_READS).Bulk()
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
                Timestamp: ts,
            })
        }
        if r, err := bulk.Run(); err != nil {
            _Log.Error(_funcName, err.Error(), r)
        }

        //// Clear the slice
        //memberIDs = memberIDs[:0]

        // Update unread counters
        if place.Privacy.Locked {
            db.C(COLLECTION_POSTS_READS_COUNTERS).UpdateAll(
                bson.M{
                    "account_id": bson.M{"$ne": post.SenderID},
                    "place_id":   placeID,
                },
                bson.M{"$inc": bson.M{"no_unreads": 1}},
            )
        } else {
            db.C(COLLECTION_POSTS_READS_COUNTERS).UpdateAll(
                bson.M{
                    "account_id": bson.M{"$ne": post.SenderID},
                    "place_id":   placeID,
                },
                bson.M{"$inc": bson.M{"no_unreads": 1}},
            )
        }

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

    return &post
}

// Adds accountID to postID's watcher list of placeID
func (pm *PostManager) AddAccountToWatcherList(postID bson.ObjectId, accountID string) bool {
    _funcName := "PostManager::AddAccountToWatcherList"
    _Log.FunctionStarted(_funcName, accountID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if _, err := db.C(COLLECTION_POSTS_WATCHERS).Upsert(
        bson.M{"_id": postID},
        bson.M{"$addToSet": bson.M{"accounts": accountID}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }
    return true
}

// AttachPlace adds a new place to the post, and changes the last_update of the post
func (pm *PostManager) AttachPlace(postID bson.ObjectId, placeID, accountID string) bool {
    _funcName := "PostManager::AttachPlace"
    _Log.FunctionStarted(_funcName, postID.Hex(), placeID, accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    defer _Manager.Post.removeCache(postID)
    defer _Manager.Place.removeCache(placeID)
    post := _Manager.Post.GetPostByID(postID)
    place := _Manager.Place.GetByID(placeID, nil)

    // Update post's document
    // places and last_update fields are updated
    if err := db.C(COLLECTION_POSTS).UpdateId(
        postID,
        bson.M{
            "$addToSet": bson.M{"places": placeID},
            "$set": bson.M{
                "last_update": Timestamp(),
                "_removed":    false,
            },
        },
    ); err != nil {
        _Log.Error(_funcName, err.Error(), "POSTS UPDATE")
        return false
    }

    // Update counters of the attached place
    _Manager.Place.IncrementCounter([]string{placeID}, PLACE_COUNTER_POSTS, 1)

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
    _funcName := "PostManager::AddRelatedTask"
    _Log.FunctionStarted(_funcName, postID.Hex(), postID.Hex(), taskID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    defer _Manager.Post.removeCache(postID)
    if err := db.C(COLLECTION_POSTS).Update(
        bson.M{"_id": postID},
        bson.M{"$addToSet": bson.M{"related_tasks": taskID}},
    ); err != nil {
        _Log.Error(_funcName, err.Error(), "POSTS update")
    }
    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{"_id": taskID},
        bson.M{"$set": bson.M{"related_post": postID}},
    ); err != nil {
        _Log.Error(_funcName, err.Error(), "TASKS update")
    }

}

// RemoveRelatedTask removes the task from the post
func (pm *PostManager) RemoveRelatedTask(postID, taskID bson.ObjectId) {
    _funcName := "PostManager::RemoveRelatedTask"
    _Log.FunctionStarted(_funcName, postID.Hex(), postID.Hex(), taskID.Hex())
    defer _Log.FunctionFinished(_funcName)
    defer _Manager.Post.removeCache(postID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_POSTS).Update(
        bson.M{"_id": postID, "related_tasks": taskID},
        bson.M{"$pull": bson.M{"related_tasks": taskID}},
    ); err != nil {
        _Log.Error(_funcName, err.Error(), "POSTS update")
    }
    if err := db.C(COLLECTION_TASKS).Update(
        bson.M{"_id": taskID},
        bson.M{"$unset": bson.M{"related_post": postID}},
    ); err != nil {
        _Log.Error(_funcName, err.Error(), "TASKS update")
    }
}

// CommentHasAccess if accountID has READ ACCESS to the postID of the commentID it returns TRUE
func (pm *PostManager) CommentHasAccess(commentID bson.ObjectId, accountID string) bool {
    _funcName := "PostManager::CommentHasAccess"
    _Log.FunctionStarted(_funcName, commentID.Hex(), accountID)
    defer _Log.FunctionFinished(_funcName)

    if comment := pm.GetCommentByID(commentID); comment != nil {
        if pm.HasAccess(comment.PostID, accountID) {
            return true
        }
    }
    return false
}

// Exists returns true if post exists and it is not deleted
func (pm *PostManager) Exists(postID bson.ObjectId) bool {
    _funcName := "PostManager::Exists"
    _Log.FunctionStarted(_funcName, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n, _ := db.C(COLLECTION_POSTS).Find(bson.M{
        "_id":      postID,
        "_removed": false,
    }).Count()
    return n > 0
}

// GetPostByID returns Post by postID, if postID does not exists it returns nil
func (pm *PostManager) GetPostByID(postID bson.ObjectId) *Post {
    _funcName := "PostManager::GetPostByID"
    _Log.FunctionStarted(_funcName, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    return _Manager.Post.readFromCache(postID)
}

// GetPostsByIDs returns an array of posts identified by postIDs, it returns an empty slice if nothing was found
func (pm *PostManager) GetPostsByIDs(postIDs []bson.ObjectId) []Post {
    _funcName := "PostManager::GetPostsByIDs"
    _Log.FunctionStarted(_funcName)
    defer _Log.FunctionFinished(_funcName)
    return _Manager.Post.readMultiFromCache(postIDs)
}

// GetPostsByPlace returns an array of Posts that are in placeID
// sortItem could have any of these values:
//		POST_SORT_LAST_UPDATE
//		POST_SORT_TIMESTAMP
// this function is preferred to be called instead of GetPostsOfPlaces
func (pm *PostManager) GetPostsByPlace(placeID, sortItem string, pg Pagination) []Post {
    _funcName := "PostManager::GetPostsByPlace"
    _Log.FunctionStarted(_funcName, placeID, sortItem)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    q := bson.M{"_removed": false}
    posts := make([]Post, 0, DEFAULT_MAX_RESULT_LIMIT)
    switch sortItem {
    case POST_SORT_LAST_UPDATE, POST_SORT_TIMESTAMP:
    default:
        sortItem = POST_SORT_TIMESTAMP
    }
    sortDir := fmt.Sprintf("-%s", sortItem)
    if pg.After > 0 {
        q[sortItem] = bson.M{"$gt": pg.After}
        sortDir = sortItem
    } else if pg.Before > 0 {
        q[sortItem] = bson.M{"$lt": pg.Before}
    }
    q["places"] = placeID
    Q := db.C(COLLECTION_POSTS).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
    _Log.ExplainQuery(_funcName, Q)

    if err := Q.All(&posts); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return posts
}

// GetPostsBySender returns an array of Posts that are sent by accountID
// sortItem could have any of these values:
//		POST_SORT_LAST_UPDATE
//		POST_SORT_TIMESTAMP
func (pm *PostManager) GetPostsBySender(accountID, sortItem string, pg Pagination) []Post {
    _funcName := "PostManager::GetPostsBySender"
    _Log.FunctionStarted(_funcName, accountID, sortItem)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    switch sortItem {
    case POST_SORT_LAST_UPDATE, POST_SORT_TIMESTAMP:
    default:
        sortItem = POST_SORT_TIMESTAMP
    }
    sortDir := fmt.Sprintf("-%s", sortItem)
    q := bson.M{"_removed": false}
    if pg.After > 0 {
        q[sortItem] = bson.M{"$gt": pg.After}
        sortDir = sortItem
    } else if pg.Before > 0 {
        q[sortItem] = bson.M{"$lt": pg.Before}

    }
    if pg.GetLimit() == 0 || pg.GetLimit() > DEFAULT_MAX_RESULT_LIMIT {
        pg.SetLimit(DEFAULT_MAX_RESULT_LIMIT)
    }
    q["sender"] = accountID
    Q := db.C(COLLECTION_POSTS).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
    _Log.ExplainQuery(_funcName, Q)
    posts := make([]Post, 0, pg.GetLimit())
    Q.All(&posts)
    return posts
}

// GetPostsOfPlaces returns an array of Posts that are in any of the places
// sortItem could have any of these values:
//		POST_SORT_LAST_UPDATE
//		POST_SORT_TIMESTAMP
func (pm *PostManager) GetPostsOfPlaces(places []string, sortItem string, pg Pagination) []Post {
    _funcName := "PostManager::GetPostsOfPLaces"
    _Log.FunctionStarted(_funcName, places, sortItem)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    switch sortItem {
    case POST_SORT_LAST_UPDATE, POST_SORT_TIMESTAMP:
    default:
        sortItem = POST_SORT_TIMESTAMP
    }
    sortDir := fmt.Sprintf("-%s", sortItem)

    q := bson.M{"_removed": false}
    posts := make([]Post, 0, pg.GetLimit())
    if pg.After > 0 {
        q[sortItem] = bson.M{"$gt": pg.After}
        sortDir = sortItem
    } else if pg.Before > 0 {
        q[sortItem] = bson.M{"$lt": pg.Before}
    }
    q["places"] = bson.M{"$in": places}
    Q := db.C(COLLECTION_POSTS).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
    _Log.ExplainQuery(_funcName, Q)

    Q.All(&posts)
    return posts
}

// GetPostWatchers returns an array of accountIDs who listen to post notifications
func (pm *PostManager) GetPostWatchers(postID bson.ObjectId) []string {
    _funcName := "PostManager::GetPostWatchers"
    _Log.FunctionStarted(_funcName, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    watchers := struct {
        PostID   bson.ObjectId `bson:"_id"`
        Accounts []string      `bson:"accounts"`
    }{}
    if err := db.C(COLLECTION_POSTS_WATCHERS).Find(bson.M{"_id": postID}).One(&watchers); err != nil {
        _Log.Error(_funcName, err.Error())
        return []string{}
    }
    return watchers.Accounts
}

// GetUnreadPostsByPlace returns an array of Posts that are unseen/unread by accountID and exists in placeID
// if subPlaces set to TRUE then it also returns unseen/unread posts in any sub-places of placeID
// that accountID is member of.
// return an empty slice if there is no unseen/unread post
func (pm *PostManager) GetUnreadPostsByPlace(placeID, accountID string, subPlaces bool, pg Pagination) []Post {
    _funcName := "PostManager::GetUnreadPostsByPlace"
    _Log.FunctionStarted(_funcName, placeID, accountID, subPlaces)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    sortItem := POST_SORT_TIMESTAMP
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
    if pg.After > 0 {
        mq["timestamp"] = bson.M{"$gt": pg.After}
        sortDir = sortItem
    } else if pg.Before > 0 {
        mq["timestamp"] = bson.M{"$lt": pg.Before}
    }

    Q := db.C(COLLECTION_POSTS_READS).Find(mq).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
    _Log.ExplainQuery(_funcName, Q)
    iter := Q.Iter()
    defer iter.Close()
    readItem := M{}
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
    _funcName := "PostManager::GetAccountsWhoReadThis"
    _Log.FunctionStarted(_funcName, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    pr := make([]PostRead, 0, pg.GetLimit())
    Q := db.C(COLLECTION_POSTS_READS_ACCOUNTS).Find(bson.M{"post_id": postID}).Skip(pg.GetSkip()).Limit(pg.GetLimit())
    _Log.ExplainQuery(_funcName, Q)
    if err := Q.All(&pr); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return pr
}

// GetCommentByID returns Comment by the given commentID and return nil if commentID does not exists
func (pm *PostManager) GetCommentByID(commentID bson.ObjectId) *Comment {
    _funcName := "PostManager::GetCommentByID"
    _Log.FunctionStarted(_funcName, commentID.Hex())
    defer _Log.FunctionFinished(_funcName)

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
    _funcName := "PostManager::GetCommentsByIDs"
    _Log.FunctionStarted(_funcName)
    defer _Log.FunctionFinished(_funcName)

    comments := _Manager.Post.readMultiCommentsFromCache(commentIDs)
    return comments

}

// GetCommentsByPostID returns an array of Comments of postID, if postID has no comments then it returns an empty slice.
func (pm *PostManager) GetCommentsByPostID(postID bson.ObjectId, pg Pagination) []Comment {
    _funcName := "PostManager::GetCommentsByPostID"
    _Log.FunctionStarted(_funcName, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    q := bson.M{
        "post_id":  postID,
        "_removed": false,
    }
    sortItem := "-timestamp"
    if pg.After > 0 {
        q["timestamp"] = bson.M{"$gt": pg.After}
        sortItem = "timestamp"
    } else if pg.Before > 0 {
        q["timestamp"] = bson.M{"$lt": pg.Before}
    }

    Q := db.C(COLLECTION_POSTS_COMMENTS).Find(q).Sort(sortItem).Skip(pg.GetSkip()).Limit(pg.GetLimit())
    _Log.ExplainQuery(_funcName, Q)
    res := []Comment{}
    Q.All(&res)
    return res
}

// GetPinnedPosts returns an array of Posts which are pinned by accountID
func (pm *PostManager) GetPinnedPosts(accountID string, pg Pagination) []Post {
    _funcName := "PostManager::GetPinnedPosts"
    _Log.FunctionStarted(_funcName, accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    q := bson.M{
        "account_id": accountID,
    }
    sortItem := POST_SORT_PIN_TIME
    sortDir := fmt.Sprintf("-%s", sortItem)
    if pg.After > 0 {
        q[sortItem] = bson.M{"$gt": pg.After}
        sortDir = sortItem
    } else if pg.Before > 0 {
        q[sortItem] = bson.M{"$lt": pg.Before}
    }
    Q := db.C(COLLECTION_ACCOUNTS_POSTS).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit())
    _Log.ExplainQuery(_funcName, Q)
    iter := Q.Iter()
    item := M{}
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
    _funcName := "PostManager::HasBeenReadBy"
    _Log.FunctionStarted(_funcName, postID.Hex(), accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    q := bson.M{
        "post_id":    postID,
        "account_id": accountID,
        "read":       false,
    }
    Q := db.C(COLLECTION_POSTS_READS).Find(q)
    _Log.ExplainQuery(_funcName, Q)
    if n, _ := Q.Count(); n > 0 {
        return false
    }
    return true
}

// HasBeenWatchedBy returns TRUE if postID has accountID in its watchers list
func (pm *PostManager) HasBeenWatchedBy(postID bson.ObjectId, accountID string) bool {
    _funcName := "PostManager::HasBeenWatchedBy"
    _Log.FunctionStarted(_funcName, postID.Hex(), accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    q := bson.M{
        "_id":      postID,
        "accounts": accountID,
    }
    Q := db.C(COLLECTION_POSTS_WATCHERS).Find(q)
    _Log.ExplainQuery(_funcName, Q)
    if n, _ := Q.Count(); n > 0 {
        return true
    }
    return false
}

// HasAccess checks if accountID has READ ACCESS to any of the postID places then it return TRUE otherwise
// it return FALSE
func (pm *PostManager) HasAccess(postID bson.ObjectId, accountID string) bool {
    _funcName := "PostManager::HasAccess"
    _Log.FunctionStarted(_funcName, postID.Hex(), accountID)
    defer _Log.FunctionFinished(_funcName)
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
    _funcName := "PostManager::HidComment"
    _Log.FunctionStarted(_funcName, commentID.Hex(), accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    c := _Manager.Post.GetCommentByID(commentID)
    if c == nil {
        return false
    }
    if err := db.C(COLLECTION_POSTS_COMMENTS).Update(
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
        _Log.Error(_funcName, err.Error())
        return false
    }

    lcms := make([]bson.M, 0, 3)
    iter := db.C(COLLECTION_POSTS_COMMENTS).Find(bson.M{
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

    db.C(COLLECTION_POSTS).UpdateId(
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
    _funcName := "PostManager::MarkAsRead"
    _Log.FunctionStarted(_funcName, postID.Hex(), accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ts := Timestamp()
    post := pm.GetPostByID(postID)

    if ci, err := db.C(COLLECTION_POSTS_READS).RemoveAll(
        bson.M{
            "account_id": accountID,
            "post_id":    postID,
            "read":       false,
        },
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    } else {
        // Add the accountID to the list of readers if he/she is not already exists
        // This function runs in background
        q := bson.M{
            "post_id":    postID,
            "account_id": accountID,
        }
        Q := db.C(COLLECTION_POSTS_READS_ACCOUNTS).Find(q)
        if n, _ := Q.Count(); n == 0 {
            db.C(COLLECTION_POSTS_READS_ACCOUNTS).Insert(
                bson.M{
                    "post_id":    postID,
                    "account_id": accountID,
                    "timestamp":  ts,
                },
            )
        }

        if ci.Removed > 0 {
            db.C(COLLECTION_POSTS_READS_COUNTERS).UpdateAll(
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

// MarsAsReadByPlace marks all the posts in the placeID as seen/read by accountID
func (pm *PostManager) MarkAsReadByPlace(placeID, accountID string) bool {
    _funcName := "PostManager::MarkAsReadByPlace"
    _Log.FunctionStarted(_funcName, placeID, accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    db.C(COLLECTION_POSTS_READS).RemoveAll(
        bson.M{
            "account_id": accountID,
            "place_id":   placeID,
            "read":       false,
        },
    )

    // Reset unread counter for the placeID
    db.C(COLLECTION_POSTS_READS_COUNTERS).UpdateAll(
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
    _funcName := "PostManager::Move"
    _Log.FunctionStarted(_funcName, postID.Hex(), oldPlaceID, newPlaceID, accountID)
    defer _Log.FunctionFinished(_funcName)
    defer _Manager.Post.removeCache(postID)
    defer _Manager.Place.removeCache(oldPlaceID)
    defer _Manager.Place.removeCache(newPlaceID)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    // Update POSTS collection
    // Add the new place first and then remove the old place
    if err := db.C(COLLECTION_POSTS).UpdateId(
        postID,
        bson.M{
            "$addToSet": bson.M{"places": newPlaceID},
            "$set": bson.M{
                "last_update": Timestamp(),
                "_removed":    false,
            },
        },
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }
    if err := db.C(COLLECTION_POSTS).UpdateId(
        postID,
        bson.M{
            "$pull":     bson.M{"places": oldPlaceID},
            "$addToSet": bson.M{"removed_places": oldPlaceID},
        },
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }

    // Update PLACES collection
    if err := db.C(COLLECTION_PLACES).Update(
        bson.M{"_id": newPlaceID},
        bson.M{"$inc": bson.M{"counters.posts": 1}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }
    if err := db.C(COLLECTION_PLACES).Update(
        bson.M{"_id": oldPlaceID},
        bson.M{"$inc": bson.M{"counters.posts": -1}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }

    pr := new(PostRead)
    iter := db.C(COLLECTION_POSTS_READS).Find(
        bson.M{
            "post_id":  postID,
            "place_id": oldPlaceID,
            "read":     false,
            "_removed": false,
        }).Select(bson.M{"account_id": 1}).Iter()
    defer iter.Close()
    for iter.Next(pr) {
        db.C(COLLECTION_POSTS_READS_COUNTERS).Update(
            bson.M{
                "account_id": pr.AccountID,
                "place_id":   oldPlaceID,
                "no_unreads": bson.M{"$gt": 0},
            },
            bson.M{"$inc": bson.M{"no_unreads": -1}},
        )
    }
    // remove all unreads from oldPlace
    db.C(COLLECTION_POSTS_READS).RemoveAll(
        bson.M{"post_id": postID, "place_id": oldPlaceID},
    )

    // Update timeline
    _Manager.PlaceActivity.PostMove(accountID, oldPlaceID, newPlaceID, postID)
    _Manager.PostActivity.PlaceMove(postID, accountID, oldPlaceID, newPlaceID)
    return true
}

// BookmarkPost adds postID to the pinned posts list of the accountID
func (pm *PostManager) BookmarkPost(accountID string, postID bson.ObjectId) {
    _funcName := "PostManager::BookmarkPost"
    _Log.FunctionStarted(_funcName, accountID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if post := _Manager.Post.GetPostByID(postID); post != nil {
        if _, err := db.C(COLLECTION_ACCOUNTS_POSTS).Upsert(
            bson.M{"account_id": accountID, "post_id": postID},
            bson.M{"$set": bson.M{
                "pin_time": Timestamp(),
            }},
        ); err != nil {
            _Log.Error(_funcName, err.Error())
        }
    }
}

// Removes removes the postID from the placeID.
// if placeID is the last place that postID are in, then removes the comments of the postID too.
func (pm *PostManager) Remove(accountID string, postID bson.ObjectId, placeID string) bool {
    _funcName := "PostManager::Remove"
    _Log.FunctionStarted(_funcName, postID.Hex(), placeID)
    defer _Log.FunctionFinished(_funcName)
    defer _Manager.Post.removeCache(postID)
    defer _Manager.Place.removeCache(placeID)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    post := pm.GetPostByID(postID)
    if post == nil {
        return false
    }
    if len(post.PlaceIDs) == 0 {
        return false
    }
    if !post.IsInPlace(placeID) {
        _Log.Error(_funcName, "Incorrect Call to PostManager::Remove")
        return false
    }

    // Update posts collection
    // if it is the last place of the post
    if len(post.PlaceIDs) == 1 {
        if err := db.C(COLLECTION_POSTS).Update(
            bson.M{"_id": postID},
            bson.M{
                "$pull":     bson.M{"places": placeID},
                "$addToSet": bson.M{"removed_places": placeID},
                "$set":      bson.M{"_removed": true},
            },
        ); err != nil {
            _Log.Error(_funcName, err.Error())
        }
    } else {
        if err := db.C(COLLECTION_POSTS).Update(
            bson.M{"_id": postID},
            bson.M{
                "$pull":     bson.M{"places": placeID},
                "$addToSet": bson.M{"removed_places": placeID},
            },
        ); err != nil {
            _Log.Error(_funcName, err.Error())
        }
    }

    // Update place counter
    _Manager.Place.IncrementCounter([]string{placeID}, PLACE_COUNTER_POSTS, -1)

    pr := new(PostRead)
    iter := db.C(COLLECTION_POSTS_READS).Find(
        bson.M{
            "post_id":  postID,
            "place_id": placeID,
            "read":     false,
            "_removed": false,
        }).Select(bson.M{"account_id": 1}).Iter()
    defer iter.Close()
    for iter.Next(pr) {
        db.C(COLLECTION_POSTS_READS_COUNTERS).Update(
            bson.M{
                "account_id": pr.AccountID,
                "place_id":   placeID,
                "no_unreads": bson.M{"$gt": 0},
            },
            bson.M{"$inc": bson.M{"no_unreads": -1}},
        )
    }
    db.C(COLLECTION_POSTS_READS).RemoveAll(
        bson.M{"post_id": postID, "place_id": placeID},
    )

    // Update timeline items
    _Manager.PlaceActivity.PostRemove(accountID, placeID, postID)

    return true
}

// RemoveComment removes the commentID from its post.
// also removes the time-line activity of the comment
func (pm *PostManager) RemoveComment(accountID string, commentID bson.ObjectId) bool {
    _funcName := "PostManager::RemoveComment"
    _Log.FunctionStarted(_funcName, commentID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ch := mgo.Change{
        Update: bson.M{"$set": bson.M{"_removed": true}},
    }
    c := new(Comment)
    if ci, err := db.C(COLLECTION_POSTS_COMMENTS).FindId(commentID).Apply(ch, c); err != nil {
        _Log.Error(_funcName, err.Error())
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
    iter := db.C(COLLECTION_POSTS_COMMENTS).Find(bson.M{
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

    db.C(COLLECTION_POSTS).UpdateId(
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
    _funcName := "PostManager::RemoveAccountFromWatcherList"
    _Log.FunctionStarted(_funcName, postID.Hex(), accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_POSTS_WATCHERS).Update(
        bson.M{"_id": postID},
        bson.M{"$pull": bson.M{"accounts": accountID}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }
    return true
}

// SetEmailMessageID set MessageID for the post, this function will be used by Gobryas service
func (pm *PostManager) SetEmailMessageID(postID bson.ObjectId, messageID string) bool {
    _funcName := "PostManager::SetEmailMessageID"
    _Log.FunctionStarted(_funcName, postID.Hex(), messageID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_POSTS).UpdateId(
        postID,
        bson.M{
            "$set": bson.M{"email_meta.message_id": messageID},
        },
    ); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return true
}

// UnpinPost removes the postID from accountID's pinned posts list
func (pm *PostManager) UnpinPost(accountID string, postID bson.ObjectId) {
    _funcName := "PostManager::UnpinPost"
    _Log.FunctionStarted(_funcName, accountID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_ACCOUNTS_POSTS).Remove(bson.M{
        "post_id":    postID,
        "account_id": accountID,
    }); err != nil {
        _Log.Error(_funcName, err.Error())
    }
}

// IsPinned returns true if postID has been pinned by accountID
func (pm *PostManager) IsPinned(accountID string, postID bson.ObjectId) bool {
    _funcName := "PostManager::IsPinned"
    _Log.FunctionStarted(_funcName, accountID, postID.Hex())
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if n, err := db.C(COLLECTION_ACCOUNTS_POSTS).Find(bson.M{
        "account_id": accountID, "post_id": postID,
    }).Count(); err != nil {
        _Log.Error(_funcName, err.Error())
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
    Preview         string          `json:"preview" bson:"preview"`
    ReplyTo         bson.ObjectId   `json:"reply_to,omitempty" bson:"reply_to,omitempty"`
    ForwardFrom     bson.ObjectId   `json:"forward_from,omitempty" bson:"forward_from,omitempty"`
    AttachmentIDs   []UniversalID   `json:"attaches" bson:"attaches"`
    AttachmentSizes []int64         `json:"attaches_size" bson:"-"`
    RelatedTasks    []bson.ObjectId `json:"related_tasks,omitempty" bson:"related_tasks,omitempty"`
    IFrameUrl       string          `json:"iframe_url,omitempty" bson:"iframe_url,omitempty"`
    Timestamp       uint64          `json:"timestamp" bson:"timestamp"`
    LastUpdate      uint64          `json:"last_update" bson:"last_update"`
    Spam            float64         `json:"spam" bson:"spam"`
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
    _funcName := "Post::HasAccess"
    _Log.FunctionStarted(_funcName, accountID)
    defer _Log.FunctionFinished(_funcName)

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
    _funcName := "Post::AddLabel"
    _Log.FunctionStarted(_funcName, accountID, labelID)
    defer _Log.FunctionFinished(_funcName)
    defer _Manager.Post.removeCache(p.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_POSTS).UpdateId(
        p.ID,
        bson.M{
            "$addToSet": bson.M{"labels": labelID},
            "$inc":      bson.M{"counters.labels": 1},
        },
    ); err != nil {
        _Log.Error(_funcName, err.Error(), 1)
        return false
    }

    // Increment label counter
    _Manager.Label.IncrementCounter(labelID, "posts", 1)

    // Add Post Activity
    _Manager.PostActivity.LabelAdd(p.ID, accountID, labelID)

    return true
}

// LabelRemoved removes label off the post
func (p *Post) RemoveLabel(accountID, labelID string) bool {
    _funcName := "Post::RemoveLabel"
    _Log.FunctionStarted(_funcName, accountID, labelID)
    defer _Log.FunctionFinished(_funcName)
    defer _Manager.Post.removeCache(p.ID)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_POSTS).UpdateId(
        p.ID,
        bson.M{
            "$pull": bson.M{"labels": labelID},
            "$inc":  bson.M{"counters.labels": -1},
        },
    ); err != nil {
        _Log.Error(_funcName, err.Error(), 1)
        return false
    }

    // Increment label counter
    _Manager.Label.IncrementCounter(labelID, "posts", -1)

    _Manager.PostActivity.LabelRemove(p.ID, accountID, labelID)

    return true
}

// MarkAsRead marks the postID as seen/read by accountID
func (p *Post) MarkAsRead(accountID string) bool {
    _funcName := "PostManager::MarkAsRead"
    _Log.FunctionStarted(_funcName, p.ID.Hex(), accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Copy()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ts := Timestamp()
    if accountID == p.SenderID {
        return false
    }
    if ci, err := db.C(COLLECTION_POSTS_READS).RemoveAll(
        bson.M{
            "account_id": accountID,
            "post_id":    p.ID,
            "read":       false,
        },
    ); err != nil {
        _Log.Error(_funcName, err.Error(), "RemoveAll")
        return false
    } else {
        // Add the accountID to the list of readers if he/she is not already exists
        // This function runs in background
        q := bson.M{
            "post_id":    p.ID,
            "account_id": accountID,
        }
        Q := db.C(COLLECTION_POSTS_READS_ACCOUNTS).Find(q)
        if n, _ := Q.Count(); n == 0 {
            db.C(COLLECTION_POSTS_READS_ACCOUNTS).Insert(
                bson.M{
                    "post_id":    p.ID,
                    "account_id": accountID,
                    "timestamp":  ts,
                },
            )
        }

        if ci.Removed > 0 {
            db.C(COLLECTION_POSTS_READS_COUNTERS).UpdateAll(
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
    _funcName := "PostManager::AddPost"
    _Log.FunctionStarted(_funcName)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    defer _Manager.Post.removeCache(p.ID)

    var postPreview string
    var ellipsis bool
    switch p.ContentType {
    case CONTENT_TYPE_TEXT_PLAIN:
        if len(postBody) > 256 {
            ellipsis = true
            postPreview = string(postBody[:256])
        } else {
            postPreview = postBody
        }
    case CONTENT_TYPE_TEXT_HTML:
        strings.NewReader(postBody)
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

    if err := db.C(COLLECTION_POSTS).UpdateId(
        p.ID,
        bson.M{
            "$set": bson.M{
                "body":    postBody,
                "subject": postSubject,
                "preview": postPreview,
                "ellipsis": ellipsis,
            },
        },
    ); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }

    // Set Post activity
    _Manager.PostActivity.Edit(p.ID, p.SenderID)

    return true

}
