package api

import (
	"regexp"
	"strings"

	"git.ronaksoft.com/nested/server/cmd/server-ntfy/client"
	"git.ronaksoft.com/nested/server/model"
	"github.com/globalsign/mgo/bson"
)

type PushManager struct {
	worker       *Worker
	Notification *ntfy.Client
}

func NewPushManager(worker *Worker) *PushManager {
	pm := new(PushManager)
	pm.Notification = ntfy.NewClient(worker.Config().GetString("JOB_ADDRESS"), worker.Model())
	pm.Notification.SetDomain(worker.Config().GetString("SENDER_DOMAIN"))
	pm.worker = worker
	if pm.Notification == nil {
		return nil
	}
	return pm
}

// Post Related Pushes
func (pm *PushManager) CloseConnection() {
	pm.Notification.Close()
}
func (pm *PushManager) NewSession(actorID, clientID string) {
	n := pm.worker.Model().Notification.NewSession(actorID, clientID)
	pm.Notification.ExternalPushNotification(n)
	pm.Notification.InternalNotificationSyncPush([]string{actorID}, nested.NotificationTypeNewSession)
}

// Place Related Pushes
func (pm *PushManager) PlaceJoined(place *nested.Place, actorID, memberID string) {
	// Create notification
	notif := pm.worker.Model().Notification.JoinedPlace(actorID, memberID, place.ID)
	pm.Notification.ExternalPushNotification(notif)

	// Send the activity sync packet over the wire
	memberIDs := place.GetMemberIDs()
	pm.Notification.InternalPlaceActivitySyncPush(memberIDs, place.ID, nested.PlaceActivityActionMemberJoin)

	// Send the notification packet over the wire
	pm.Notification.InternalNotificationSyncPush([]string{memberID}, nested.NotificationTypeJoinedPlace)
}
func (pm *PushManager) PlaceSettingsUpdated(place *nested.Place, actorID string) {
	for _, creatorID := range place.CreatorIDs {
		if creatorID != actorID {
			n := pm.worker.Model().Notification.PlaceSettingsChanged(creatorID, actorID, place.ID)
			if n != nil && n.Timestamp != n.LastUpdate {
				pm.Notification.ExternalPushNotification(n)
				pm.Notification.InternalNotificationSyncPush([]string{creatorID}, nested.NotificationTypePlaceSettingsChanged)
			}
		}
	}
}
func (pm *PushManager) PlaceMemberDemoted(place *nested.Place, actorID, memberID string) {
	notif := pm.worker.Model().Notification.Demoted(memberID, actorID, place.ID)
	pm.Notification.ExternalPushNotification(notif)
	pm.Notification.InternalNotificationSyncPush([]string{memberID}, nested.NotificationTypeDemoted)
}
func (pm *PushManager) PlaceMemberPromoted(place *nested.Place, actorID, memberID string) {
	notif := pm.worker.Model().Notification.Promoted(memberID, actorID, place.ID)
	pm.Notification.ExternalPushNotification(notif)
	pm.Notification.InternalNotificationSyncPush([]string{memberID}, nested.NotificationTypePromoted)
}

// Post Related Pushes
func (pm *PushManager) PostAdded(post *nested.Post) {
	pm.Notification.ExternalPushPlaceActivityPostAdded(post)
	/*
	   Every member of every place of the post will receive an InternalPlaceActivitySync
	*/
	for _, placeID := range post.PlaceIDs {
		// Internal
		place := pm.worker.Model().Place.GetByID(placeID, nil)
		pm.Notification.InternalPlaceActivitySyncPush(
			place.GetMemberIDs(),
			placeID,
			nested.PlaceActivityActionPostAdd,
		)
	}
}
func (pm *PushManager) PostEdited(post *nested.Post) {
	for _, placeID := range post.PlaceIDs {
		place := pm.worker.Model().Place.GetByID(placeID, nil)
		memberIDs := place.GetMemberIDs()
		pm.Notification.InternalPostActivitySyncPush(memberIDs, post.ID, nested.PostActivityActionEdited, post.PlaceIDs)
	}
}
func (pm *PushManager) PostMovedTo(post *nested.Post, oldPlace, newPlace *nested.Place) {
	pm.Notification.InternalPlaceActivitySyncPush(
		newPlace.GetMemberIDs(),
		newPlace.ID,
		nested.PlaceActivityActionPostMoveTo,
	)
	pm.Notification.InternalPlaceActivitySyncPush(
		oldPlace.GetMemberIDs(),
		oldPlace.ID,
		nested.PlaceActivityActionPostMoveFrom,
	)
	for _, placeID := range post.PlaceIDs {
		if placeID == oldPlace.ID || placeID == newPlace.ID {
			continue
		}
		place := pm.worker.Model().Place.GetByID(placeID, nil)
		pm.Notification.InternalPostActivitySyncPush(
			place.GetMemberIDs(),
			post.ID,
			nested.PostActivityActionPlaceMove,
			post.PlaceIDs,
		)
	}

}
func (pm *PushManager) PostAttached(post *nested.Post, attachedPlaceIDs []string) {
	pm.Notification.ExternalPushPlaceActivityPostAttached(post, attachedPlaceIDs)
	for _, placeID := range attachedPlaceIDs {
		// Internal
		place := pm.worker.Model().Place.GetByID(placeID, nil)
		pm.Notification.InternalPlaceActivitySyncPush(
			place.GetMemberIDs(),
			placeID,
			nested.PlaceActivityActionPostAdd,
		)
	}
	for _, placeID := range post.PlaceIDs {
		if place := pm.worker.model.Place.GetByID(placeID, nil); place != nil {
			pm.Notification.InternalPostActivitySyncPush(
				place.GetMemberIDs(),
				post.ID,
				nested.PostActivityActionPlaceAttach,
				post.PlaceIDs,
			)
		}
	}
}
func (pm *PushManager) PostCommentAdded(post *nested.Post, comment *nested.Comment) {
	// find mentioned ids and External Notifications
	regx, _ := regexp.Compile(`@([a-zA-Z0-9-]*)(\s|$)`)
	matches := regx.FindAllString(comment.Body, 100)
	mentionedIDs := nested.MB{}
	for _, m := range matches {
		mentionedID := strings.Trim(string(m[1:]), " ") // remove @ from the mentioned id
		if post.HasAccess(mentionedID) {
			n := pm.worker.Model().Notification.AddMention(comment.SenderID, mentionedID, post.ID, comment.ID)
			pm.Notification.ExternalPushNotification(n)
			pm.Notification.InternalNotificationSyncPush([]string{mentionedID}, nested.NotificationTypeMention)
			mentionedIDs[mentionedID] = true
		}
	}
	// Notification Internal and External Push
	watcherIDs := make([]string, 0)
	for _, accountID := range pm.worker.Model().Post.GetPostWatchers(post.ID) {
		if post.HasAccess(accountID) {
			if comment.SenderID != accountID {
				if _, ok := mentionedIDs[accountID]; !ok {
					n := pm.worker.Model().Notification.Comment(accountID, comment.SenderID, post.ID, comment.ID)
					pm.Notification.ExternalPushNotification(n)
					watcherIDs = append(watcherIDs, accountID)
				}
			}
		} else {
			pm.worker.Model().Post.RemoveAccountFromWatcherList(post.ID, accountID)
		}
	}
	pm.Notification.InternalNotificationSyncPush(watcherIDs, nested.NotificationTypeComment)

	// Activity Internal Push Notifications
	for _, placeID := range post.PlaceIDs {
		place := pm.worker.Model().Place.GetByID(placeID, nil)
		memberIDs := place.GetMemberIDs()
		pm.Notification.InternalPostActivitySyncPush(memberIDs, post.ID, nested.PostActivityActionCommentAdd, post.PlaceIDs)
	}
}
func (pm *PushManager) PostCommentRemoved(post *nested.Post, comment *nested.Comment) {
	// Activity Internal Push Notifications
	for _, placeID := range post.PlaceIDs {
		place := pm.worker.Model().Place.GetByID(placeID, nil)
		memberIDs := place.GetMemberIDs()
		pm.Notification.InternalPostActivitySyncPush(memberIDs, post.ID, nested.PostActivityActionCommentRemove, post.PlaceIDs)
	}
}
func (pm *PushManager) PostLabelAdded(post *nested.Post, label *nested.Label) {
	// Activity Internal Push Notifications
	for _, placeID := range post.PlaceIDs {
		place := pm.worker.Model().Place.GetByID(placeID, nil)
		memberIDs := place.GetMemberIDs()
		pm.Notification.InternalPostActivitySyncPush(memberIDs, post.ID, nested.PostActivityActionLabelAdd, post.PlaceIDs)
	}
}
func (pm *PushManager) PostLabelRemoved(post *nested.Post, label *nested.Label) {
	// Activity Internal Push Notifications
	for _, placeID := range post.PlaceIDs {
		place := pm.worker.Model().Place.GetByID(placeID, nil)
		memberIDs := place.GetMemberIDs()
		pm.Notification.InternalPostActivitySyncPush(memberIDs, post.ID, nested.PostActivityActionLabelRemove, post.PlaceIDs)
	}
}

// Label Related Pushes
func (pm *PushManager) LabelRequestApproved(labelRequest *nested.LabelRequest) {
	notifLabelRequestApproved := pm.worker.Model().Notification.LabelRequestApproved(
		labelRequest.RequesterID,
		labelRequest.LabelID,
		labelRequest.ResponderID,
		labelRequest.ID,
	)
	pm.Notification.ExternalPushNotification(notifLabelRequestApproved)
	pm.Notification.InternalNotificationSyncPush([]string{labelRequest.RequesterID}, nested.NotificationTypeLabelRequestApproved)
}
func (pm *PushManager) LabelRequestRejected(labelRequest *nested.LabelRequest) {
	notifLabelRequestRejected := pm.worker.Model().Notification.LabelRequestRejected(
		labelRequest.RequesterID,
		labelRequest.LabelID,
		labelRequest.ResponderID,
		labelRequest.ID,
	)
	pm.Notification.ExternalPushNotification(notifLabelRequestRejected)
	pm.Notification.InternalNotificationSyncPush([]string{labelRequest.RequesterID}, nested.NotificationTypeLabelRequestRejected)
}

// Task Related Pushes
func (pm *PushManager) TaskAssigned(task *nested.Task) {
	if task.AssignorID != task.AssigneeID {
		n1 := pm.worker.Model().Notification.TaskAssigned(task.AssigneeID, task.AssignorID, task)
		pm.Notification.ExternalPushNotification(n1)
		pm.Notification.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskAssigned)
	}
}
func (pm *PushManager) TaskOverdue(task *nested.Task) {
	n1 := pm.worker.Model().Notification.TaskOverdue(task.AssignorID, task)
	pm.Notification.ExternalPushNotification(n1)
	n2 := pm.worker.Model().Notification.TaskOverdue(task.AssigneeID, task)
	pm.Notification.ExternalPushNotification(n2)
	pm.Notification.InternalNotificationSyncPush([]string{task.AssigneeID, task.AssignorID}, nested.NotificationTypeTaskOverDue)
}
func (pm *PushManager) TaskRejected(task *nested.Task, actorID string) {
	n1 := pm.worker.Model().Notification.TaskRejected(task.AssignorID, actorID, task)
	pm.Notification.ExternalPushNotification(n1)

	// send sync-n to the wire
	pm.Notification.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskRejected)
}
func (pm *PushManager) TaskAccepted(task *nested.Task, actorID string) {
	n1 := pm.worker.Model().Notification.TaskAccepted(task.AssignorID, actorID, task)
	pm.Notification.ExternalPushNotification(n1)
	pm.Notification.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskAccepted)

	// send task activity sync over the wire
	accountIDs := nested.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	pm.Notification.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, nested.TASK_ACTIVITY_STATUS_CHANGED)
}
func (pm *PushManager) TaskFailed(task *nested.Task, actorID string) {
	if actorID != task.AssigneeID {
		n := pm.worker.Model().Notification.TaskCompleted(task.AssigneeID, actorID, task)
		pm.Notification.ExternalPushNotification(n)
		pm.Notification.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskFailed)
	}
	if actorID != task.AssignorID {
		n := pm.worker.Model().Notification.TaskCompleted(task.AssignorID, actorID, task)
		pm.Notification.ExternalPushNotification(n)
		pm.Notification.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskFailed)
	}

	// send task activity sync over the wire
	accountIDs := nested.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	pm.Notification.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, nested.TASK_ACTIVITY_STATUS_CHANGED)
}
func (pm *PushManager) TaskCompleted(task *nested.Task, actorID string) {
	if actorID != task.AssigneeID {
		n := pm.worker.Model().Notification.TaskCompleted(task.AssigneeID, actorID, task)
		pm.Notification.ExternalPushNotification(n)
		pm.Notification.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskCompleted)
	}
	if actorID != task.AssignorID {
		n := pm.worker.Model().Notification.TaskCompleted(task.AssignorID, actorID, task)
		pm.Notification.ExternalPushNotification(n)
		pm.Notification.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskCompleted)
	}

	// send task activity sync over the wire
	accountIDs := nested.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	pm.Notification.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, nested.TASK_ACTIVITY_STATUS_CHANGED)
}
func (pm *PushManager) TaskHold(task *nested.Task, actorID string) {
	if actorID != task.AssignorID {
		n := pm.worker.Model().Notification.TaskHold(task.AssignorID, actorID, task)
		pm.Notification.ExternalPushNotification(n)
		pm.Notification.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskHold)
	}
	if actorID != task.AssigneeID {
		n := pm.worker.Model().Notification.TaskHold(task.AssigneeID, actorID, task)
		pm.Notification.ExternalPushNotification(n)
		pm.Notification.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskHold)
	}

	// send task activity sync over the wire
	accountIDs := nested.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	pm.Notification.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, nested.TASK_ACTIVITY_STATUS_CHANGED)
}
func (pm *PushManager) TaskInProgress(task *nested.Task, actorID string) {
	if actorID != task.AssignorID {
		n := pm.worker.Model().Notification.TaskInProgress(task.AssignorID, actorID, task)
		pm.Notification.ExternalPushNotification(n)
		pm.Notification.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskInProgress)
	}
	if actorID != task.AssigneeID {
		n := pm.worker.Model().Notification.TaskInProgress(task.AssigneeID, actorID, task)
		pm.Notification.ExternalPushNotification(n)
		pm.Notification.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskInProgress)
	}

	// send task activity sync over the wire
	accountIDs := nested.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	pm.Notification.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, nested.TASK_ACTIVITY_STATUS_CHANGED)
}
func (pm *PushManager) TaskCommentAdded(task *nested.Task, actorID string, activityID bson.ObjectId, commentText string) {
	// find mentioned ids and External Notifications
	regx, _ := regexp.Compile(`@([a-zA-Z0-9-]*)(\s|$)`)
	matches := regx.FindAllString(commentText, 100)
	mentionedIDs := nested.MB{}
	for _, m := range matches {
		mentionedID := strings.Trim(string(m[1:]), " ") // remove @ from the mentioned id
		if task.HasAccess(mentionedID, nested.TASK_ACCESS_READ) {
			n := pm.worker.Model().Notification.TaskCommentMentioned(mentionedID, actorID, task, activityID)
			pm.Notification.ExternalPushNotification(n)
			pm.Notification.InternalNotificationSyncPush([]string{mentionedID}, nested.NotificationTypeTaskMention)
			mentionedIDs[mentionedID] = true
		}
	}
	if actorID != task.AssigneeID {
		if _, ok := mentionedIDs[task.AssigneeID]; !ok {
			n := pm.worker.Model().Notification.TaskComment(task.AssigneeID, actorID, task, activityID)
			pm.Notification.ExternalPushNotification(n)
			pm.Notification.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskComment)
		}
	}
	if actorID != task.AssignorID {
		if _, ok := mentionedIDs[task.AssignorID]; !ok {
			n := pm.worker.Model().Notification.TaskComment(task.AssignorID, actorID, task, activityID)
			pm.Notification.ExternalPushNotification(n)
			pm.Notification.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskComment)
		}
	}

	// send task activity sync over the wire
	accountIDs := nested.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	pm.Notification.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, nested.TASK_ACTIVITY_COMMENT)
}
func (pm *PushManager) TaskAddedToCandidates(task *nested.Task, actorID string, memberIDs []string) {
	for _, memberID := range memberIDs {
		if actorID != memberID {
			n1 := pm.worker.Model().Notification.TaskCandidateAdded(memberID, actorID, task)
			pm.Notification.ExternalPushNotification(n1)
		}
	}
	pm.Notification.InternalNotificationSyncPush(memberIDs, nested.NotificationTypeTaskAddToCandidates)

	// send task activity sync over the wire
	accountIDs := nested.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	pm.Notification.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, nested.TASK_ACTIVITY_CANDIDATE_ADDED)

}
func (pm *PushManager) TaskAddedToWatchers(task *nested.Task, actorID string, memberIDs []string) {
	for _, memberID := range memberIDs {
		if actorID != memberID {
			n1 := pm.worker.Model().Notification.TaskWatcherAdded(memberID, actorID, task)
			pm.Notification.ExternalPushNotification(n1)
		}
	}
	pm.Notification.InternalNotificationSyncPush(memberIDs, nested.NotificationTypeTaskAddToWatchers)

	// send task activity sync over the wire
	accountIDs := nested.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	pm.Notification.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, nested.TASK_ACTIVITY_WATCHER_ADDED)

}
func (pm *PushManager) TaskAddedToEditors(task *nested.Task, actorID string, memberIDs []string) {
	for _, memberID := range memberIDs {
		if actorID != memberID {
			n1 := pm.worker.Model().Notification.TaskEditorAdded(memberID, actorID, task)
			pm.Notification.ExternalPushNotification(n1)
		}
	}
	pm.Notification.InternalNotificationSyncPush(memberIDs, nested.NotificationTypeTaskAddToEditors)

	// send task activity sync over the wire
	accountIDs := nested.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	pm.Notification.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, nested.TASK_ACTIVITY_EDITOR_ADDED)

}
func (pm *PushManager) TaskNewActivity(task *nested.Task, action nested.TaskAction) {
	// send task activity sync over the wire
	accountIDs := nested.MB{}
	accountIDs.AddKeys(
		[]string{task.AssignorID, task.AssigneeID},
		task.CandidateIDs,
		task.WatcherIDs,
	)
	pm.Notification.InternalTaskActivitySyncPush(accountIDs.KeysToArray(), task.ID, action)
}

// Notification Related Pushes
func (pm *PushManager) ClearNotification(requester *nested.Account, n *nested.Notification) {
	if n == nil {
		pm.Notification.ExternalPushClearAll(requester.ID)
	} else {
		pm.Notification.ExternalPushClear(n)
	}
}
