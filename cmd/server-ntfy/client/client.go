package ntfy

import (
	"encoding/json"
	"fmt"
	"log"

	"git.ronaksoftware.com/nested/server/model"
	"github.com/globalsign/mgo/bson"
	"github.com/nats-io/go-nats"
)

var (
	_NotificationTitles map[int]string
)

func init() {
	_NotificationTitles = make(map[int]string)
	_NotificationTitles[nested.NOTIFICATION_TYPE_MENTION] = "Mention in Post"
	_NotificationTitles[nested.NOTIFICATION_TYPE_COMMENT] = "Comment on Post"
	_NotificationTitles[nested.NOTIFICATION_TYPE_JOINED_PLACE] = "Join Place"
	_NotificationTitles[nested.NOTIFICATION_TYPE_PROMOTED] = "Promoted"
	_NotificationTitles[nested.NOTIFICATION_TYPE_DEMOTED] = "Demoted"
	_NotificationTitles[nested.NOTIFICATION_TYPE_PLACE_SETTINGS_CHANGED] = "Place Settings Updated"
	_NotificationTitles[nested.NOTIFICATION_TYPE_NEW_SESSION] = "New Session"
	_NotificationTitles[nested.NOTIFICATION_TYPE_LABEL_REQUEST_APPROVED] = "Request Approved"
	_NotificationTitles[nested.NOTIFICATION_TYPE_LABEL_REQUEST_REJECTED] = "Request Rejected"
	_NotificationTitles[nested.NOTIFICATION_TYPE_LABEL_REQUEST_CREATED] = "New Request"
	_NotificationTitles[nested.NOTIFICATION_TYPE_LABEL_JOINED] = "Access To Label"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_MENTION] = "Mention in Task"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_COMMENT] = "Comment on Task"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_ASSIGNED] = "Task Assigned"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_ASSIGNEE_CHANGED] = "Task Assignee Changed"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_ADD_TO_CANDIDATES] = "Added To Task's Candidates"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_ADD_TO_WATCHERS] = "Added To Task's Watchers"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_DUE_TIME_UPDATED] = "Task Deadline Updated"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_OVER_DUE] = "Task is Overdue"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_UPDATED] = "Task Updated"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_REJECTED] = "Task Rejected"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_ACCEPTED] = "Task Accepted"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_COMPLETED] = "Task Completed"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_HOLD] = "Task was Hold"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_IN_PROGRESS] = "Task is in Progress"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_FAILED] = "Task Failed"
	_NotificationTitles[nested.NOTIFICATION_TYPE_TASK_ADD_TO_EDITORS] = "Added to Task's Editors"
}

// WebsocketPush
type WebsocketPush struct {
	BundleID    string `json:"bundle_id" bson:"bundle_id"`
	WebsocketID string `json:"ws_id" bson:"ws_id"`
	Payload     string `json:"payload" bson:"payload"`
}

// Client
type Client struct {
	model   *nested.Manager
	address string
	nat     *nats.Conn
	domain  string
}

// NewClient
func NewClient(address string, model *nested.Manager) *Client {
	c := new(Client)
	c.address = address
	c.model = model
	if nat, err := nats.Connect(address); err != nil {
		log.Println("NTFY::Client::NewClient::Error::", err.Error())
		return nil
	} else {
		c.nat = nat
	}
	return c
}

func (c *Client) Close() {
	if c.nat != nil {
		c.nat.Close()
	}
}

func (c *Client) connect() {
	var err error
	c.nat, err = nats.Connect(c.address)
	if err != nil {
		log.Println("NTFY::Client::connect::Error::", err.Error())
	}
}

func (c *Client) SetDomain(domain string) {
	c.domain = domain
}

func (c *Client) RegisterDevice(id, token, os, userID string) error {
	cmd := CMDRegisterDevice{
		DeviceID:    id,
		DeviceToken: token,
		DeviceOS:    os,
		UserID:      userID,
	}

	if b, err := json.Marshal(cmd); err != nil {
		return err
	} else if err = c.nat.Publish("NTFY.REGISTER.DEVICE", b); err != nil {
		return err

	}
	return nil
}

func (c *Client) UnregisterDevice(id, token, uid string) error {
	cmd := CMDUnRegisterDevice{
		DeviceID:    id,
		DeviceToken: token,
		UserID:      uid,
	}

	if b, err := json.Marshal(cmd); err != nil {
		return err
	} else if err = c.nat.Publish("NTFY.UNREGISTER.DEVICE", b); err != nil {
		return err
	}
	return nil
}

func (c *Client) RegisterWebsocket(userID, deviceID, bundleID, websocketID string) error {
	cmd := CMDRegisterWebsocket{
		DeviceID:    deviceID,
		UserID:      userID,
		BundleID:    bundleID,
		WebsocketID: websocketID,
	}
	if b, err := json.Marshal(cmd); err != nil {
		return err
	} else if err = c.nat.Publish("NTFY.REGISTER.WEBSOCKET", b); err != nil {
		return err
	}
	return nil
}

func (c *Client) UnregisterWebsocket(websocketID, bundleID string) error {
	cmd := CMDUnRegisterWebsocket{
		WebsocketID: websocketID,
		BundleID:    bundleID,
	}
	if b, err := json.Marshal(cmd); err != nil {
		return err
	} else if err = c.nat.Publish("NTFY.UNREGISTER.WEBSOCKET", b); err != nil {
		return err

	}
	return nil
}

// Internal Pushes
func (c *Client) InternalPush(targets []string, msg string, localonly bool) error {
	cmd := CMDPushInternal{
		Targets:   targets,
		Message:   msg,
		LocalOnly: localonly,
	}

	if b, err := json.Marshal(cmd); err != nil {
		log.Println("NotificationClient::InternalPush::Error::", err.Error())
		return err
	} else {
		c.nat.Publish("NTFY.PUSH.INTERNAL", b)
	}
	return nil
}

func (c *Client) InternalPlaceActivitySyncPush(targets []string, placeID string, action int) {
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
			log.Println("NotificationClient::InternalPlaceActivitySyncPush::Error::", err.Error())
		} else {
			c.InternalPush(targets[iStart:iEnd], string(jmsg), false)
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

func (c *Client) InternalPostActivitySyncPush(targets []string, postID bson.ObjectId, action nested.PostAction, placeIDs []string) {
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
			"cmd":  "sync-p",
			"data": nested.M{
				"post_id": postID.Hex(),
				"action":  action,
				"places":  placeIDs,
			},
		}
		if jmsg, err := json.Marshal(msg); err != nil {
			log.Println("NotificationClient::InternalPlaceActivitySyncPush::Error::", err.Error())
		} else {
			c.InternalPush(targets[iStart:iEnd], string(jmsg), false)
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

func (c *Client) InternalTaskActivitySyncPush(targets []string, taskID bson.ObjectId, action nested.TaskAction) {
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
			"cmd":  "sync-t",
			"data": nested.M{
				"task_id": taskID,
				"action":  action,
			},
		}
		if jmsg, err := json.Marshal(msg); err != nil {
			log.Println("NotificationClient::InternalTaskActivitySyncPush::Error::", err.Error())
		} else {
			c.InternalPush(targets[iStart:iEnd], string(jmsg), false)
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

func (c *Client) InternalNotificationSyncPush(targets []string, notificationType int) {
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
			"cmd":  "sync-n",
			"data": nested.M{
				"type": notificationType,
			},
		}
		if jmsg, err := json.Marshal(msg); err != nil {
			log.Println("NotificationClient::InternalNotificationSyncPush::Error::", err.Error())
		} else {
			c.InternalPush(targets[iStart:iEnd], string(jmsg), false)
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

// External Pushes
func (c *Client) ExternalPush(targets []string, data map[string]string) error {
	cmd := CMDPushExternal{
		Targets: targets,
		Data:    data,
	}
	cmd.Data["domain"] = c.domain
	if b, err := json.Marshal(cmd); err != nil {
		return err
	} else {
		c.nat.Publish("NTFY.PUSH.EXTERNAL", b)
	}
	return nil
}

func (c *Client) ExternalPushNotification(n *nested.Notification) {
	actor := c.model.Account.GetByID(n.ActorID, nil)

	pushData := nested.MS{
		"actor_id":        actor.ID,
		"actor_name":      fmt.Sprintf("%s %s", actor.FirstName, actor.LastName),
		"actor_picture":   string(actor.Picture.X128),
		"account_id":      n.AccountID,
		"type":            "n",
		"subject":         fmt.Sprintf("%d", n.Type),
		"title":           _NotificationTitles[n.Type],
		"notification_id": n.ID,
	}
	switch n.Type {
	case nested.NOTIFICATION_TYPE_MENTION:
		comment := c.model.Post.GetCommentByID(n.CommentID)
		if comment == nil {
			log.Println("ExternalPushNotification::Error::Comment_Not_Exists")
			log.Println("Arguments:", n.ID)
			return
		}
		txt := fmt.Sprintf("%s %s: %s", actor.FirstName, actor.LastName, comment.Body)
		pushData["post_id"] = n.PostID.Hex()
		pushData["comment_id"] = n.CommentID.Hex()
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_COMMENT:
		comment := c.model.Post.GetCommentByID(n.CommentID)
		if comment == nil {
			log.Println("ExternalPushNotification::Error::Comment_Not_Exists")
			log.Println("Arguments:", n.ID)
			return
		}
		txt := fmt.Sprintf("%s %s commented on your post: %s", actor.FirstName, actor.LastName, comment.Body)
		pushData["post_id"] = n.PostID.Hex()
		pushData["comment_id"] = n.CommentID.Hex()
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_JOINED_PLACE:
		place := c.model.Place.GetByID(n.PlaceID, nested.M{"name": 1})
		if place == nil {
			log.Println("ExternalPushNotification::Error::Place_Not_Exists")
			log.Println("Arguments:", n.ID)
			return
		}
		txt := fmt.Sprintf("%s %s added you to \"%s\"", actor.FirstName, actor.LastName, place.Name)
		pushData["place_id"] = n.PlaceID
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_PROMOTED:
		place := c.model.Place.GetByID(n.PlaceID, nested.M{"name": 1})
		if place == nil {
			log.Println("ExternalPushNotification::Error::Place_Not_Exists")
			log.Println("Arguments:", n.ID)
			return
		}
		txt := fmt.Sprintf("%s %s promoted you to be a Manager in \"%s\"", actor.FirstName, actor.LastName, place.Name)
		pushData["place_id"] = n.PlaceID
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_DEMOTED:
		place := c.model.Place.GetByID(n.PlaceID, nested.M{"name": 1})
		if place == nil {
			log.Println("ExternalPushNotification::Error::Place_Not_Exists")
			log.Println("Arguments:", n.ID)
			return
		}
		txt := fmt.Sprintf("%s %s demoted you in \"%s\"", actor.FirstName, actor.LastName, place.Name)
		pushData["place_id"] = n.PlaceID
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_PLACE_SETTINGS_CHANGED:
		place := c.model.Place.GetByID(n.PlaceID, nested.M{"name": 1})
		if place == nil {
			log.Println("ExternalPushNotification::Error::Place_Not_Exists")
			log.Println("Arguments:", n.ID)
			return
		}
		txt := fmt.Sprintf("%s %s changed the settings of \"%s\"", actor.FirstName, actor.LastName, place.Name)
		pushData["place_id"] = n.PlaceID
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_NEW_SESSION:
		txt := fmt.Sprintf("You are logged in from device: %s", n.ClientID)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_LABEL_REQUEST_APPROVED:
		label := c.model.Label.GetByID(n.LabelID)
		if label != nil {
			txt := fmt.Sprintf("Your request for label (%s) was approved.", label.Title)
			pushData["msg"] = txt
			pushData["sound"] = "nc.aiff"
		}
	case nested.NOTIFICATION_TYPE_LABEL_REQUEST_REJECTED:
		label := c.model.Label.GetByID(n.LabelID)
		if label != nil {
			txt := fmt.Sprintf("Your request for label (%s) was rejected.", label.Title)
			pushData["msg"] = txt
			pushData["sound"] = "nc.aiff"
		}
	case nested.NOTIFICATION_TYPE_LABEL_REQUEST_CREATED:
		txt := fmt.Sprintf("New label request from %s %s", actor.FirstName, actor.LastName)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_TASK_ADD_TO_WATCHERS:
		txt := fmt.Sprintf("%s %s added you to the watchers of task: %s", actor.FirstName, actor.LastName, n.Data.TaskTitle)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_TASK_ADD_TO_CANDIDATES:
		txt := fmt.Sprintf("%s %s added you to the candidates of task: %s", actor.FirstName, actor.LastName, n.Data.TaskTitle)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_TASK_COMPLETED:
		txt := fmt.Sprintf("%s %s completed task: %s", actor.FirstName, actor.LastName, n.Data.TaskTitle)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_TASK_ACCEPTED:
		txt := fmt.Sprintf("%s %s accepted your task: %s", actor.FirstName, actor.LastName, n.Data.TaskTitle)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_TASK_REJECTED:
		txt := fmt.Sprintf("%s %s rejected your task: %s", actor.FirstName, actor.LastName, n.Data.TaskTitle)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NOTIFICATION_TYPE_TASK_MENTION:
		task := c.model.Task.GetByID(n.TaskID)
		if task != nil {
			txt := fmt.Sprintf("%s %s mentioned you in task: %s", actor.FirstName, actor.LastName, task.Title)
			pushData["msg"] = txt
			pushData["sound"] = "nc.aiff"
		}
	default:
		return
	}
	c.ExternalPush([]string{n.AccountID}, pushData)
	return
}

func (c *Client) ExternalPushPlaceActivityPostAdded(post *nested.Post) {
	pushData := nested.MS{
		"type":   "a",
		"action": fmt.Sprintf("%d", nested.PLACE_ACTIVITY_ACTION_POST_ADD),
	}

	if post.Internal {
		actor := c.model.Account.GetByID(post.SenderID, nil)
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
		place := c.model.Place.GetByID(placeID, nested.M{"groups": 1})
		if place == nil {
			log.Println("ExternalPushActivityAddPost::Error::Place_Not_Exists")
			log.Println("Arguments:", post.ID)
			continue
		}
		memberIDs := c.model.Group.GetItems(place.Groups[nested.NOTIFICATION_GROUP])
		for _, memberID := range memberIDs {
			if memberID != post.SenderID {
				pushData["account_id"] = memberID
				c.ExternalPush([]string{memberID}, pushData)
			}
		}
	}

}

func (c *Client) ExternalPushPlaceActivityPostAttached(post *nested.Post, placeIDs []string) {
	pushData := nested.MS{
		"type":   "a",
		"action": string(nested.PLACE_ACTIVITY_ACTION_POST_ADD),
	}

	if post.Internal {
		actor := c.model.Account.GetByID(post.SenderID, nil)
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

	for _, placeID := range placeIDs {
		place := c.model.Place.GetByID(placeID, nested.M{"groups": 1})
		if place == nil {
			log.Println("ExternalPushActivityAddPost::Error::Place_Not_Exists")
			log.Println("Arguments:", post.ID)
			continue
		}
		memberIDs := c.model.Group.GetItems(place.Groups[nested.NOTIFICATION_GROUP])
		for _, memberID := range memberIDs {
			if memberID != post.SenderID {
				pushData["account_id"] = memberID
				c.ExternalPush([]string{memberID}, pushData)
			}
		}
	}

}

func (c *Client) ExternalPushClear(n *nested.Notification) {
	pushData := nested.MS{
		"notification_id": n.ID,
		"subject":         "clear",
	}
	c.ExternalPush([]string{n.AccountID}, pushData)
}

func (c *Client) ExternalPushClearAll(accountID string) {
	pushData := nested.MS{
		"notification_id": "all",
		"subject":         "clear",
	}
	c.ExternalPush([]string{accountID}, pushData)
}

// OnWebsocketPush Registers callback function for receiving messages on subject GATEWAY
// TODO:: fixed subject ?!
func (c *Client) OnWebsocketPush(callback func(push *WebsocketPush)) {
	c.nat.Subscribe("GATEWAY", func(msg *nats.Msg) {
		websocketPush := new(WebsocketPush)
		json.Unmarshal(msg.Data, websocketPush)
		callback(websocketPush)
	})
}
