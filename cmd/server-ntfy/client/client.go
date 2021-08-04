package ntfy

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"git.ronaksoft.com/nested/server/model"
	"github.com/globalsign/mgo/bson"
	"github.com/nats-io/go-nats"
)

var (
	_NotificationTitles map[int]string
)

func init() {
	_NotificationTitles = make(map[int]string)
	_NotificationTitles[nested.NotificationTypeMention] = "Mention in Post"
	_NotificationTitles[nested.NotificationTypeComment] = "Comment on Post"
	_NotificationTitles[nested.NotificationTypeJoinedPlace] = "Join Place"
	_NotificationTitles[nested.NotificationTypePromoted] = "Promoted"
	_NotificationTitles[nested.NotificationTypeDemoted] = "Demoted"
	_NotificationTitles[nested.NotificationTypePlaceSettingsChanged] = "Place Settings Updated"
	_NotificationTitles[nested.NotificationTypeNewSession] = "New Session"
	_NotificationTitles[nested.NotificationTypeLabelRequestApproved] = "Request Approved"
	_NotificationTitles[nested.NotificationTypeLabelRequestRejected] = "Request Rejected"
	_NotificationTitles[nested.NotificationTypeLabelRequestCreated] = "New Request"
	_NotificationTitles[nested.NotificationTypeLabelJoined] = "Access To Label"
	_NotificationTitles[nested.NotificationTypeTaskMention] = "Mention in Task"
	_NotificationTitles[nested.NotificationTypeTaskComment] = "Comment on Task"
	_NotificationTitles[nested.NotificationTypeTaskAssigned] = "Task Assigned"
	_NotificationTitles[nested.NotificationTypeTaskAssigneeChanged] = "Task Assignee Changed"
	_NotificationTitles[nested.NotificationTypeTaskAddToCandidates] = "Added To Task's Candidates"
	_NotificationTitles[nested.NotificationTypeTaskAddToWatchers] = "Added To Task's Watchers"
	_NotificationTitles[nested.NotificationTypeTaskDueTimeUpdated] = "Task Deadline Updated"
	_NotificationTitles[nested.NotificationTypeTaskOverDue] = "Task is Overdue"
	_NotificationTitles[nested.NotificationTypeTaskUpdated] = "Task Updated"
	_NotificationTitles[nested.NotificationTypeTaskRejected] = "Task Rejected"
	_NotificationTitles[nested.NotificationTypeTaskAccepted] = "Task Accepted"
	_NotificationTitles[nested.NotificationTypeTaskCompleted] = "Task Completed"
	_NotificationTitles[nested.NotificationTypeTaskHold] = "Task was Hold"
	_NotificationTitles[nested.NotificationTypeTaskInProgress] = "Task is in Progress"
	_NotificationTitles[nested.NotificationTypeTaskFailed] = "Task Failed"
	_NotificationTitles[nested.NotificationTypeTaskAddToEditors] = "Added to Task's Editors"
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

// ExternalPush send the notification
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
	case nested.NotificationTypeMention:
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
	case nested.NotificationTypeComment:
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
	case nested.NotificationTypeJoinedPlace:
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
	case nested.NotificationTypePromoted:
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
	case nested.NotificationTypeDemoted:
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
	case nested.NotificationTypePlaceSettingsChanged:
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
	case nested.NotificationTypeNewSession:
		txt := fmt.Sprintf("You are logged in from device: %s", n.ClientID)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypeLabelRequestApproved:
		label := c.model.Label.GetByID(n.LabelID)
		if label != nil {
			txt := fmt.Sprintf("Your request for label (%s) was approved.", label.Title)
			pushData["msg"] = txt
			pushData["sound"] = "nc.aiff"
		}
	case nested.NotificationTypeLabelRequestRejected:
		label := c.model.Label.GetByID(n.LabelID)
		if label != nil {
			txt := fmt.Sprintf("Your request for label (%s) was rejected.", label.Title)
			pushData["msg"] = txt
			pushData["sound"] = "nc.aiff"
		}
	case nested.NotificationTypeLabelRequestCreated:
		txt := fmt.Sprintf("New label request from %s %s", actor.FirstName, actor.LastName)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypeTaskAddToWatchers:
		txt := fmt.Sprintf("%s %s added you to the watchers of task: %s", actor.FirstName, actor.LastName, n.Data.TaskTitle)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypeTaskAddToCandidates:
		txt := fmt.Sprintf("%s %s added you to the candidates of task: %s", actor.FirstName, actor.LastName, n.Data.TaskTitle)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypeTaskCompleted:
		txt := fmt.Sprintf("%s %s completed task: %s", actor.FirstName, actor.LastName, n.Data.TaskTitle)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypeTaskAccepted:
		txt := fmt.Sprintf("%s %s accepted your task: %s", actor.FirstName, actor.LastName, n.Data.TaskTitle)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypeTaskRejected:
		txt := fmt.Sprintf("%s %s rejected your task: %s", actor.FirstName, actor.LastName, n.Data.TaskTitle)
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypeTaskMention:
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
		"action": strconv.Itoa(nested.PLACE_ACTIVITY_ACTION_POST_ADD),
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
