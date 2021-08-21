package pusher

import (
	"context"
	"encoding/json"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"fmt"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/config"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"git.ronaksoft.com/nested/server/pkg/session"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"github.com/globalsign/mgo/bson"
	"github.com/kataras/iris/v12/websocket"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"strconv"
	"strings"
	"sync"
)

/*
   Creation Time: 2021 - Aug - 04
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type WebsocketPush struct {
	BundleID    string `json:"bundle_id" bson:"bundle_id"`
	WebsocketID string `json:"ws_id" bson:"ws_id"`
	Payload     string `json:"payload" bson:"payload"`
}

type PushFunc func(push WebsocketPush) bool

type Pusher struct {
	domain    string
	bundleID  string
	ws        *session.WebsocketManager
	dev       *session.DeviceManager
	model     *nested.Manager
	fcm       *messaging.Client
	wsConnMtx sync.RWMutex
	wsConns   map[string]*websocket.Conn
}

func New(model *nested.Manager, bundleID, domain string) *Pusher {
	p := &Pusher{
		domain:   domain,
		bundleID: bundleID,
		ws:       session.NewWebsocketManager(model.Cache()),
		dev:      session.NewDeviceManager(model.DB()),
		model:    model,
		wsConns:  make(map[string]*websocket.Conn, 128),
	}

	// Initialize FCM Client
	fcmCredPath := config.GetString(config.FirebaseCredPath)
	if fcmCredPath != "" {
		if c, err := firebase.NewApp(
			context.Background(),
			nil,
			option.WithCredentialsFile("/ronak/certs/firebase-cred.json"),
		); err != nil {
			log.Fatal("could not create FCM app", zap.String("CredPath", fcmCredPath), zap.Error(err))
		} else {
			p.fcm, err = c.Messaging(context.Background())
			if err != nil {
				log.Fatal("could not create FCM messaging client", zap.Error(err))
			}
		}
	}

	p.ws.RemoveByBundleID(p.bundleID)
	return p
}

func (p *Pusher) GetOnlineAccounts(bundleID string) []string {
	return p.ws.GetAccountsByBundleID(bundleID)
}

func (p *Pusher) RegisterDevice(id, token, os, userID string) error {
	req := cmdRegisterDevice{
		DeviceID:    id,
		DeviceToken: token,
		DeviceOS:    os,
		UserID:      userID,
	}

	log.Debug("Register Device",
		zap.String("DeviceID", req.DeviceID),
		zap.String("UserID", req.UserID),
	)

	if !p.dev.Update(req.DeviceID, req.DeviceToken, req.DeviceOS, req.UserID) {
		if !p.dev.Register(req.DeviceID, req.DeviceToken, req.DeviceOS, req.UserID) {
			log.Warn("We could not register device was not successful")
		}
	}

	return nil
}

func (p *Pusher) UnregisterDevice(id, token, uid string) error {
	req := cmdUnRegisterDevice{
		DeviceID:    id,
		DeviceToken: token,
		UserID:      uid,
	}

	if !p.dev.Remove(req.DeviceID) {
		log.Warn("unregister device was not successful")
	}
	return nil
}

func (p *Pusher) RegisterWebsocket(userID, deviceID, bundleID, websocketID string) error {
	req := cmdRegisterWebsocket{
		DeviceID:    deviceID,
		UserID:      userID,
		BundleID:    bundleID,
		WebsocketID: websocketID,
	}
	// register websocket
	p.ws.Register(req.WebsocketID, req.BundleID, req.DeviceID, req.UserID)

	// Set device as connected and update the badges
	p.dev.SetAsConnected(req.DeviceID, req.UserID)
	return nil
}

func (p *Pusher) UnregisterWebsocket(websocketID, bundleID string) error {
	req := cmdUnRegisterWebsocket{
		WebsocketID: websocketID,
		BundleID:    bundleID,
	}

	// Remove websocket object and set device as disconnected
	ws := p.ws.Remove(req.WebsocketID, req.BundleID)
	if ws != nil {
		p.dev.SetAsDisconnected(ws.DeviceID)
	}
	return nil
}

func (p *Pusher) pushCB(push WebsocketPush) bool {
	p.wsConnMtx.RLock()
	c := p.wsConns[push.WebsocketID]
	p.wsConnMtx.RUnlock()
	if c == nil {
		return false
	}
	c.Write(websocket.Message{
		IsNative: true,
		Body:     []byte(push.Payload),
	})
	return true
}
func (p *Pusher) internalPush(targets []string, msg string, localOnly bool) error {
	req := cmdPushInternal{
		Targets:   targets,
		Message:   msg,
		LocalOnly: localOnly,
	}
	log.Debug("Push Internal",
		zap.Strings("Targets", req.Targets),
		zap.Bool("LocalOnly", localOnly),
		zap.String("MSG", msg),
	)

	if req.LocalOnly {
		for _, uid := range req.Targets {
			websockets := p.ws.GetWebsocketsByAccountID(uid, p.bundleID)
			for _, ws := range websockets {

				if !p.pushCB(
					WebsocketPush{
						WebsocketID: ws.WebsocketID,
						Payload:     req.Message,
						BundleID:    ws.BundleID},
				) {
					p.ws.Remove(ws.WebsocketID, p.bundleID)
				}
			}
		}
	} else {
		for _, uid := range req.Targets {
			websockets := p.ws.GetWebsocketsByAccountID(uid, "")
			for _, ws := range websockets {
				if !p.pushCB(
					WebsocketPush{
						WebsocketID: ws.WebsocketID,
						Payload:     req.Message,
						BundleID:    ws.BundleID},
				) {
					p.ws.Remove(ws.WebsocketID, p.bundleID)
				}
			}
		}
	}
	return nil
}

func (p *Pusher) InternalPlaceActivitySyncPush(targets []string, placeID string, action int) {
	if len(targets) == 0 {
		return
	}
	iStart := 0
	iLength := global.DefaultMaxResultLimit
	iEnd := iStart + iLength
	if iEnd > len(targets) {
		iEnd = len(targets)
	}
	for {
		msg := tools.M{
			"type": "p",
			"cmd":  "sync-a",
			"data": tools.M{
				"place_id": placeID,
				"action":   action,
			},
		}
		if jmsg, err := json.Marshal(msg); err != nil {
			log.Sugar().Warn("NotificationClient::InternalPlaceActivitySyncPush::Error::", err.Error())
		} else {
			p.internalPush(targets[iStart:iEnd], string(jmsg), false)
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

func (p *Pusher) InternalPostActivitySyncPush(targets []string, postID bson.ObjectId, action global.PostAction, placeIDs []string) {
	if len(targets) == 0 {
		return
	}
	iStart := 0
	iLength := global.DefaultMaxResultLimit
	iEnd := iStart + iLength
	if iEnd > len(targets) {
		iEnd = len(targets)
	}
	for {
		msg := tools.M{
			"type": "p",
			"cmd":  "sync-p",
			"data": tools.M{
				"post_id": postID.Hex(),
				"action":  action,
				"places":  placeIDs,
			},
		}
		if jmsg, err := json.Marshal(msg); err != nil {
			log.Sugar().Warn("NotificationClient::InternalPlaceActivitySyncPush::Error::", err.Error())
		} else {
			p.internalPush(targets[iStart:iEnd], string(jmsg), false)
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

func (p *Pusher) InternalTaskActivitySyncPush(targets []string, taskID bson.ObjectId, action global.TaskAction) {
	if len(targets) == 0 {
		return
	}
	iStart := 0
	iLength := global.DefaultMaxResultLimit
	iEnd := iStart + iLength
	if iEnd > len(targets) {
		iEnd = len(targets)
	}
	for {
		msg := tools.M{
			"type": "p",
			"cmd":  "sync-t",
			"data": tools.M{
				"task_id": taskID,
				"action":  action,
			},
		}
		if jmsg, err := json.Marshal(msg); err != nil {
			log.Sugar().Warn("NotificationClient::InternalTaskActivitySyncPush::Error::", err.Error())
		} else {
			p.internalPush(targets[iStart:iEnd], string(jmsg), false)
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

func (p *Pusher) InternalNotificationSyncPush(targets []string, notificationType int) {
	if len(targets) == 0 {
		return
	}
	iStart := 0
	iLength := global.DefaultMaxResultLimit
	iEnd := iStart + iLength
	if iEnd > len(targets) {
		iEnd = len(targets)
	}
	for {
		msg := tools.M{
			"type": "p",
			"cmd":  "sync-n",
			"data": tools.M{
				"type": notificationType,
			},
		}
		if jmsg, err := json.Marshal(msg); err != nil {
			log.Sugar().Warn("NotificationClient::InternalNotificationSyncPush::Error::", err.Error())
		} else {
			p.internalPush(targets[iStart:iEnd], string(jmsg), false)
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

func (p *Pusher) externalPush(targets []string, data map[string]string) {
	req := cmdPushExternal{
		Targets: targets,
		Data:    data,
	}
	req.Data["domain"] = p.domain

	log.Debug("Push External",
		zap.Strings("Targets", req.Targets),
		zap.Any("Data", req.Data),
	)

	for _, uid := range req.Targets {
		go func(uid string) {
			p.dev.IncrementBadge(uid)
			devices := p.dev.GetByAccountID(uid)
			for _, d := range devices {
				p.sendFCM(d, req)
			}
		}(uid)

	}
	return
}

func (p *Pusher) ExternalPushNotification(n *nested.Notification) {
	actor := p.model.Account.GetByID(n.ActorID, nil)

	pushData := tools.MS{
		"actor_id":        actor.ID,
		"actor_name":      fmt.Sprintf("%s %s", actor.FirstName, actor.LastName),
		"actor_picture":   string(actor.Picture.X128),
		"account_id":      n.AccountID,
		"type":            "n",
		"subject":         fmt.Sprintf("%dev", n.Type),
		"title":           _NotificationTitles[n.Type],
		"notification_id": n.ID,
	}
	switch n.Type {
	case nested.NotificationTypeMention:
		comment := p.model.Post.GetCommentByID(n.CommentID)
		if comment == nil {
			log.Sugar().Warn("ExternalPushNotification::Error::Comment_Not_Exists", "Arguments:", n.ID)
			return
		}
		txt := fmt.Sprintf("%s %s: %s", actor.FirstName, actor.LastName, comment.Body)
		pushData["post_id"] = n.PostID.Hex()
		pushData["comment_id"] = n.CommentID.Hex()
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypeComment:
		comment := p.model.Post.GetCommentByID(n.CommentID)
		if comment == nil {
			log.Sugar().Warn("ExternalPushNotification::Error::Comment_Not_Exists", "Arguments", n.ID)
			return
		}
		txt := fmt.Sprintf("%s %s commented on your post: %s", actor.FirstName, actor.LastName, comment.Body)
		pushData["post_id"] = n.PostID.Hex()
		pushData["comment_id"] = n.CommentID.Hex()
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypeJoinedPlace:
		place := p.model.Place.GetByID(n.PlaceID, tools.M{"name": 1})
		if place == nil {
			log.Warn("ExternalPushNotification::Error::Place_Not_Exists", zap.String("NID", n.ID))
			return
		}
		txt := fmt.Sprintf("%s %s added you to \"%s\"", actor.FirstName, actor.LastName, place.Name)
		pushData["place_id"] = n.PlaceID
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypePromoted:
		place := p.model.Place.GetByID(n.PlaceID, tools.M{"name": 1})
		if place == nil {
			log.Warn("ExternalPushNotification::Error::Place_Not_Exists", zap.String("NID", n.ID))
			return
		}
		txt := fmt.Sprintf("%s %s promoted you to be a Manager in \"%s\"", actor.FirstName, actor.LastName, place.Name)
		pushData["place_id"] = n.PlaceID
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypeDemoted:
		place := p.model.Place.GetByID(n.PlaceID, tools.M{"name": 1})
		if place == nil {
			log.Warn("ExternalPushNotification::Error::Place_Not_Exists", zap.String("NID", n.ID))
			return
		}
		txt := fmt.Sprintf("%s %s demoted you in \"%s\"", actor.FirstName, actor.LastName, place.Name)
		pushData["place_id"] = n.PlaceID
		pushData["msg"] = txt
		pushData["sound"] = "nc.aiff"
	case nested.NotificationTypePlaceSettingsChanged:
		place := p.model.Place.GetByID(n.PlaceID, tools.M{"name": 1})
		if place == nil {
			log.Warn("ExternalPushNotification::Error::Place_Not_Exists", zap.String("NID", n.ID))
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
		label := p.model.Label.GetByID(n.LabelID)
		if label != nil {
			txt := fmt.Sprintf("Your request for label (%s) was approved.", label.Title)
			pushData["msg"] = txt
			pushData["sound"] = "nc.aiff"
		}
	case nested.NotificationTypeLabelRequestRejected:
		label := p.model.Label.GetByID(n.LabelID)
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
		task := p.model.Task.GetByID(n.TaskID)
		if task != nil {
			txt := fmt.Sprintf("%s %s mentioned you in task: %s", actor.FirstName, actor.LastName, task.Title)
			pushData["msg"] = txt
			pushData["sound"] = "nc.aiff"
		}
	default:
		return
	}
	p.externalPush([]string{n.AccountID}, pushData)
	return
}

func (p *Pusher) ExternalPushPlaceActivityPostAdded(post *nested.Post) {
	pushData := tools.MS{
		"type":   "a",
		"action": fmt.Sprintf("%dev", nested.PlaceActivityActionPostAdd),
	}

	if post.Internal {
		actor := p.model.Account.GetByID(post.SenderID, nil)
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
			pushData["msg"] = fmt.Sprintf("%s shared a post with %dev Attachments", pushData["actor_name"], post.Counters.Attachments)
		}
	}

	for _, placeID := range post.PlaceIDs {
		place := p.model.Place.GetByID(placeID, tools.M{"groups": 1})
		if place == nil {
			log.Warn("ExternalPushNotification::Error::Place_Not_Exists", zap.String("PostID", post.ID.Hex()))
			continue
		}
		memberIDs := p.model.Group.GetItems(place.Groups[nested.NotificationGroup])
		for _, memberID := range memberIDs {
			if memberID != post.SenderID {
				pushData["account_id"] = memberID
				p.externalPush([]string{memberID}, pushData)
			}
		}
	}

}

func (p *Pusher) ExternalPushPlaceActivityPostAttached(post *nested.Post, placeIDs []string) {
	pushData := tools.MS{
		"type":   "a",
		"action": strconv.Itoa(nested.PlaceActivityActionPostAdd),
	}

	if post.Internal {
		actor := p.model.Account.GetByID(post.SenderID, nil)
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
			pushData["msg"] = fmt.Sprintf("%s shared a post with %dev Attachments", pushData["actor_name"], post.Counters.Attachments)
		}
	}

	for _, placeID := range placeIDs {
		place := p.model.Place.GetByID(placeID, tools.M{"groups": 1})
		if place == nil {
			log.Warn("ExternalPushNotification::Error::Place_Not_Exists", zap.String("PostID", post.ID.Hex()))
			continue
		}
		memberIDs := p.model.Group.GetItems(place.Groups[nested.NotificationGroup])
		for _, memberID := range memberIDs {
			if memberID != post.SenderID {
				pushData["account_id"] = memberID
				p.externalPush([]string{memberID}, pushData)
			}
		}
	}

}

func (p *Pusher) ExternalPushClear(n *nested.Notification) {
	pushData := tools.MS{
		"notification_id": n.ID,
		"subject":         "clear",
	}
	p.externalPush([]string{n.AccountID}, pushData)
}

func (p *Pusher) ExternalPushClearAll(accountID string) {
	pushData := tools.MS{
		"notification_id": "all",
		"subject":         "clear",
	}
	p.externalPush([]string{accountID}, pushData)
}

func (p *Pusher) NewSession(actorID, clientID string) {
	n := p.model.Notification.NewSession(actorID, clientID)
	p.ExternalPushNotification(n)
	p.InternalNotificationSyncPush([]string{actorID}, nested.NotificationTypeNewSession)
}

func (p *Pusher) PlaceJoined(place *nested.Place, actorID, memberID string) {
	// Create notification
	notif := p.model.Notification.JoinedPlace(actorID, memberID, place.ID)
	p.ExternalPushNotification(notif)

	// Send the activity sync packet over the wire
	memberIDs := place.GetMemberIDs()
	p.InternalPlaceActivitySyncPush(memberIDs, place.ID, nested.PlaceActivityActionMemberJoin)

	// Send the notification packet over the wire
	p.InternalNotificationSyncPush([]string{memberID}, nested.NotificationTypeJoinedPlace)
}
func (p *Pusher) PlaceSettingsUpdated(place *nested.Place, actorID string) {
	for _, creatorID := range place.CreatorIDs {
		if creatorID != actorID {
			n := p.model.Notification.PlaceSettingsChanged(creatorID, actorID, place.ID)
			if n != nil && n.Timestamp != n.LastUpdate {
				p.ExternalPushNotification(n)
				p.InternalNotificationSyncPush([]string{creatorID}, nested.NotificationTypePlaceSettingsChanged)
			}
		}
	}
}
func (p *Pusher) PlaceMemberDemoted(place *nested.Place, actorID, memberID string) {
	notif := p.model.Notification.Demoted(memberID, actorID, place.ID)
	p.ExternalPushNotification(notif)
	p.InternalNotificationSyncPush([]string{memberID}, nested.NotificationTypeDemoted)
}
func (p *Pusher) PlaceMemberPromoted(place *nested.Place, actorID, memberID string) {
	notif := p.model.Notification.Promoted(memberID, actorID, place.ID)
	p.ExternalPushNotification(notif)
	p.InternalNotificationSyncPush([]string{memberID}, nested.NotificationTypePromoted)
}

func (p *Pusher) PostAdded(post *nested.Post) {
	p.ExternalPushPlaceActivityPostAdded(post)
	/*
	   Every member of every place of the post will receive an InternalPlaceActivitySync
	*/
	for _, placeID := range post.PlaceIDs {
		// Internal
		place := p.model.Place.GetByID(placeID, nil)
		p.InternalPlaceActivitySyncPush(
			place.GetMemberIDs(),
			placeID,
			nested.PlaceActivityActionPostAdd,
		)
	}
}
func (p *Pusher) PostEdited(post *nested.Post) {
	for _, placeID := range post.PlaceIDs {
		place := p.model.Place.GetByID(placeID, nil)
		memberIDs := place.GetMemberIDs()
		p.InternalPostActivitySyncPush(memberIDs, post.ID, global.PostActivityActionEdited, post.PlaceIDs)
	}
}
func (p *Pusher) PostMovedTo(post *nested.Post, oldPlace, newPlace *nested.Place) {
	p.InternalPlaceActivitySyncPush(
		newPlace.GetMemberIDs(),
		newPlace.ID,
		nested.PlaceActivityActionPostMoveTo,
	)
	p.InternalPlaceActivitySyncPush(
		oldPlace.GetMemberIDs(),
		oldPlace.ID,
		nested.PlaceActivityActionPostMoveFrom,
	)
	for _, placeID := range post.PlaceIDs {
		if placeID == oldPlace.ID || placeID == newPlace.ID {
			continue
		}
		place := p.model.Place.GetByID(placeID, nil)
		p.InternalPostActivitySyncPush(
			place.GetMemberIDs(),
			post.ID,
			global.PostActivityActionPlaceMove,
			post.PlaceIDs,
		)
	}

}
func (p *Pusher) PostAttached(post *nested.Post, attachedPlaceIDs []string) {
	p.ExternalPushPlaceActivityPostAttached(post, attachedPlaceIDs)
	for _, placeID := range attachedPlaceIDs {
		// Internal
		place := p.model.Place.GetByID(placeID, nil)
		p.InternalPlaceActivitySyncPush(
			place.GetMemberIDs(),
			placeID,
			nested.PlaceActivityActionPostAdd,
		)
	}
	for _, placeID := range post.PlaceIDs {
		if place := p.model.Place.GetByID(placeID, nil); place != nil {
			p.InternalPostActivitySyncPush(
				place.GetMemberIDs(),
				post.ID,
				global.PostActivityActionPlaceAttach,
				post.PlaceIDs,
			)
		}
	}
}
func (p *Pusher) PostCommentAdded(post *nested.Post, comment *nested.Comment) {
	matches := global.RegExMention.FindAllString(comment.Body, 100)
	mentionedIDs := tools.MB{}
	for _, m := range matches {
		mentionedID := strings.Trim(string(m[1:]), " ") // remove @ from the mentioned id
		if post.HasAccess(mentionedID) {
			n := p.model.Notification.AddMention(comment.SenderID, mentionedID, post.ID, comment.ID)
			p.ExternalPushNotification(n)
			p.InternalNotificationSyncPush([]string{mentionedID}, nested.NotificationTypeMention)
			mentionedIDs[mentionedID] = true
		}
	}
	// Notification Internal and External Push
	watcherIDs := make([]string, 0)
	for _, accountID := range p.model.Post.GetPostWatchers(post.ID) {
		if post.HasAccess(accountID) {
			if comment.SenderID != accountID {
				if _, ok := mentionedIDs[accountID]; !ok {
					n := p.model.Notification.Comment(accountID, comment.SenderID, post.ID, comment.ID)
					p.ExternalPushNotification(n)
					watcherIDs = append(watcherIDs, accountID)
				}
			}
		} else {
			p.model.Post.RemoveAccountFromWatcherList(post.ID, accountID)
		}
	}
	p.InternalNotificationSyncPush(watcherIDs, nested.NotificationTypeComment)

	// Activity Internal Push Notifications
	for _, placeID := range post.PlaceIDs {
		place := p.model.Place.GetByID(placeID, nil)
		memberIDs := place.GetMemberIDs()
		p.InternalPostActivitySyncPush(memberIDs, post.ID, global.PostActivityActionCommentAdd, post.PlaceIDs)
	}
}
func (p *Pusher) PostCommentRemoved(post *nested.Post, comment *nested.Comment) {
	// Activity Internal Push Notifications
	for _, placeID := range post.PlaceIDs {
		place := p.model.Place.GetByID(placeID, nil)
		memberIDs := place.GetMemberIDs()
		p.InternalPostActivitySyncPush(memberIDs, post.ID, global.PostActivityActionCommentRemove, post.PlaceIDs)
	}
}
func (p *Pusher) PostLabelAdded(post *nested.Post, label *nested.Label) {
	// Activity Internal Push Notifications
	for _, placeID := range post.PlaceIDs {
		place := p.model.Place.GetByID(placeID, nil)
		memberIDs := place.GetMemberIDs()
		p.InternalPostActivitySyncPush(memberIDs, post.ID, global.PostActivityActionLabelAdd, post.PlaceIDs)
	}
}
func (p *Pusher) PostLabelRemoved(post *nested.Post, label *nested.Label) {
	// Activity Internal Push Notifications
	for _, placeID := range post.PlaceIDs {
		place := p.model.Place.GetByID(placeID, nil)
		memberIDs := place.GetMemberIDs()
		p.InternalPostActivitySyncPush(memberIDs, post.ID, global.PostActivityActionLabelRemove, post.PlaceIDs)
	}
}

func (p *Pusher) LabelRequestApproved(labelRequest *nested.LabelRequest) {
	notifLabelRequestApproved := p.model.Notification.LabelRequestApproved(
		labelRequest.RequesterID,
		labelRequest.LabelID,
		labelRequest.ResponderID,
		labelRequest.ID,
	)
	p.ExternalPushNotification(notifLabelRequestApproved)
	p.InternalNotificationSyncPush([]string{labelRequest.RequesterID}, nested.NotificationTypeLabelRequestApproved)
}
func (p *Pusher) LabelRequestRejected(labelRequest *nested.LabelRequest) {
	notifLabelRequestRejected := p.model.Notification.LabelRequestRejected(
		labelRequest.RequesterID,
		labelRequest.LabelID,
		labelRequest.ResponderID,
		labelRequest.ID,
	)
	p.ExternalPushNotification(notifLabelRequestRejected)
	p.InternalNotificationSyncPush([]string{labelRequest.RequesterID}, nested.NotificationTypeLabelRequestRejected)
}

func (p *Pusher) TaskAssigned(task *nested.Task) {
	if task.AssignorID != task.AssigneeID {
		n1 := p.model.Notification.TaskAssigned(task.AssigneeID, task.AssignorID, task)
		p.ExternalPushNotification(n1)
		p.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskAssigned)
	}
}
func (p *Pusher) TaskOverdue(task *nested.Task) {
	n1 := p.model.Notification.TaskOverdue(task.AssignorID, task)
	p.ExternalPushNotification(n1)
	n2 := p.model.Notification.TaskOverdue(task.AssigneeID, task)
	p.ExternalPushNotification(n2)
	p.InternalNotificationSyncPush([]string{task.AssigneeID, task.AssignorID}, nested.NotificationTypeTaskOverDue)
}
func (p *Pusher) TaskRejected(task *nested.Task, actorID string) {
	n1 := p.model.Notification.TaskRejected(task.AssignorID, actorID, task)
	p.ExternalPushNotification(n1)

	// send sync-n to the wire
	p.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskRejected)
}
func (p *Pusher) TaskAccepted(task *nested.Task, actorID string) {
	n1 := p.model.Notification.TaskAccepted(task.AssignorID, actorID, task)
	p.ExternalPushNotification(n1)
	p.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskAccepted)

	// send task activity sync over the wire
	accountIDs := tools.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	p.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, global.TaskActivityStatusChanged)
}
func (p *Pusher) TaskFailed(task *nested.Task, actorID string) {
	if actorID != task.AssigneeID {
		n := p.model.Notification.TaskCompleted(task.AssigneeID, actorID, task)
		p.ExternalPushNotification(n)
		p.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskFailed)
	}
	if actorID != task.AssignorID {
		n := p.model.Notification.TaskCompleted(task.AssignorID, actorID, task)
		p.ExternalPushNotification(n)
		p.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskFailed)
	}

	// send task activity sync over the wire
	accountIDs := tools.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	p.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, global.TaskActivityStatusChanged)
}
func (p *Pusher) TaskCompleted(task *nested.Task, actorID string) {
	if actorID != task.AssigneeID {
		n := p.model.Notification.TaskCompleted(task.AssigneeID, actorID, task)
		p.ExternalPushNotification(n)
		p.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskCompleted)
	}
	if actorID != task.AssignorID {
		n := p.model.Notification.TaskCompleted(task.AssignorID, actorID, task)
		p.ExternalPushNotification(n)
		p.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskCompleted)
	}

	// send task activity sync over the wire
	accountIDs := tools.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	p.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, global.TaskActivityStatusChanged)
}
func (p *Pusher) TaskHold(task *nested.Task, actorID string) {
	if actorID != task.AssignorID {
		n := p.model.Notification.TaskHold(task.AssignorID, actorID, task)
		p.ExternalPushNotification(n)
		p.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskHold)
	}
	if actorID != task.AssigneeID {
		n := p.model.Notification.TaskHold(task.AssigneeID, actorID, task)
		p.ExternalPushNotification(n)
		p.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskHold)
	}

	// send task activity sync over the wire
	accountIDs := tools.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	p.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, global.TaskActivityStatusChanged)
}
func (p *Pusher) TaskInProgress(task *nested.Task, actorID string) {
	if actorID != task.AssignorID {
		n := p.model.Notification.TaskInProgress(task.AssignorID, actorID, task)
		p.ExternalPushNotification(n)
		p.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskInProgress)
	}
	if actorID != task.AssigneeID {
		n := p.model.Notification.TaskInProgress(task.AssigneeID, actorID, task)
		p.ExternalPushNotification(n)
		p.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskInProgress)
	}

	// send task activity sync over the wire
	accountIDs := tools.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	p.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, global.TaskActivityStatusChanged)
}
func (p *Pusher) TaskCommentAdded(task *nested.Task, actorID string, activityID bson.ObjectId, commentText string) {
	matches := global.RegExMention.FindAllString(commentText, 100)
	mentionedIDs := tools.MB{}
	for _, m := range matches {
		mentionedID := strings.Trim(string(m[1:]), " ") // remove @ from the mentioned id
		if task.HasAccess(mentionedID, nested.TaskAccessRead) {
			n := p.model.Notification.TaskCommentMentioned(mentionedID, actorID, task, activityID)
			p.ExternalPushNotification(n)
			p.InternalNotificationSyncPush([]string{mentionedID}, nested.NotificationTypeTaskMention)
			mentionedIDs[mentionedID] = true
		}
	}
	if actorID != task.AssigneeID {
		if _, ok := mentionedIDs[task.AssigneeID]; !ok {
			n := p.model.Notification.TaskComment(task.AssigneeID, actorID, task, activityID)
			p.ExternalPushNotification(n)
			p.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskComment)
		}
	}
	if actorID != task.AssignorID {
		if _, ok := mentionedIDs[task.AssignorID]; !ok {
			n := p.model.Notification.TaskComment(task.AssignorID, actorID, task, activityID)
			p.ExternalPushNotification(n)
			p.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskComment)
		}
	}

	// send task activity sync over the wire
	accountIDs := tools.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	p.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, global.TaskActivityComment)
}
func (p *Pusher) TaskAddedToCandidates(task *nested.Task, actorID string, memberIDs []string) {
	for _, memberID := range memberIDs {
		if actorID != memberID {
			n1 := p.model.Notification.TaskCandidateAdded(memberID, actorID, task)
			p.ExternalPushNotification(n1)
		}
	}
	p.InternalNotificationSyncPush(memberIDs, nested.NotificationTypeTaskAddToCandidates)

	// send task activity sync over the wire
	accountIDs := tools.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	p.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, global.TaskActivityCandidateAdded)

}
func (p *Pusher) TaskAddedToWatchers(task *nested.Task, actorID string, memberIDs []string) {
	for _, memberID := range memberIDs {
		if actorID != memberID {
			n1 := p.model.Notification.TaskWatcherAdded(memberID, actorID, task)
			p.ExternalPushNotification(n1)
		}
	}
	p.InternalNotificationSyncPush(memberIDs, nested.NotificationTypeTaskAddToWatchers)

	// send task activity sync over the wire
	accountIDs := tools.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	p.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, global.TaskActivityWatcherAdded)

}
func (p *Pusher) TaskAddedToEditors(task *nested.Task, actorID string, memberIDs []string) {
	for _, memberID := range memberIDs {
		if actorID != memberID {
			n1 := p.model.Notification.TaskEditorAdded(memberID, actorID, task)
			p.ExternalPushNotification(n1)
		}
	}
	p.InternalNotificationSyncPush(memberIDs, nested.NotificationTypeTaskAddToEditors)

	// send task activity sync over the wire
	accountIDs := tools.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	p.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, global.TaskActivityEditorAdded)

}
func (p *Pusher) TaskNewActivity(task *nested.Task, action global.TaskAction) {
	// send task activity sync over the wire
	accountIDs := tools.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	p.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, action)
}

func (p *Pusher) ClearNotification(requester *nested.Account, n *nested.Notification) {
	if n == nil {
		p.ExternalPushClearAll(requester.ID)
	} else {
		p.ExternalPushClear(n)
	}
}
