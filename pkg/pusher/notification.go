package pusher

import (
	"context"
	"firebase.google.com/go/v4/messaging"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/session"
)

/*
   Creation Time: 2021 - Aug - 04
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

var (
	_NotificationTitles map[int]string
)

func init() {
	_NotificationTitles = map[int]string{
		nested.NotificationTypeMention:              "Mention in Post",
		nested.NotificationTypeComment:              "Comment on Post",
		nested.NotificationTypeJoinedPlace:          "Join Place",
		nested.NotificationTypePromoted:             "Promoted",
		nested.NotificationTypeDemoted:              "Demoted",
		nested.NotificationTypePlaceSettingsChanged: "Place Settings Updated",
		nested.NotificationTypeNewSession:           "New Session",
		nested.NotificationTypeLabelRequestApproved: "Request Approved",
		nested.NotificationTypeLabelRequestRejected: "Request Rejected",
		nested.NotificationTypeLabelRequestCreated:  "New Request",
		nested.NotificationTypeLabelJoined:          "Access To Label",
		nested.NotificationTypeTaskMention:          "Mention in Task",
		nested.NotificationTypeTaskComment:          "Comment on Task",
		nested.NotificationTypeTaskAssigned:         "Task Assigned",
		nested.NotificationTypeTaskAssigneeChanged:  "Task Assignee Changed",
		nested.NotificationTypeTaskAddToCandidates:  "Added To Task's Candidates",
		nested.NotificationTypeTaskAddToWatchers:    "Added To Task's Watchers",
		nested.NotificationTypeTaskDueTimeUpdated:   "Task Deadline Updated",
		nested.NotificationTypeTaskOverDue:          "Task is Overdue",
		nested.NotificationTypeTaskUpdated:          "Task Updated",
		nested.NotificationTypeTaskRejected:         "Task Rejected",
		nested.NotificationTypeTaskAccepted:         "Task Accepted",
		nested.NotificationTypeTaskCompleted:        "Task Completed",
		nested.NotificationTypeTaskHold:             "Task was Hold",
		nested.NotificationTypeTaskInProgress:       "Task is in Progress",
		nested.NotificationTypeTaskFailed:           "Task Failed",
		nested.NotificationTypeTaskAddToEditors:     "Added to Task's Editors",
	}

}

func (p *Pusher) sendFCM(d session.Device, req cmdPushExternal) {
	if p.fcm == nil {
		return
	}

	message := messaging.Message{
		Data:  req.Data,
		Token: d.Token,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Data:     req.Data,
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: req.Data["title"],
						Body:  req.Data["msg"],
					},
					Badge:      &d.Badge,
					CustomData: make(map[string]interface{}),
				},
			},
		},
	}
	for k, v := range req.Data {
		message.APNS.Payload.Aps.CustomData[k] = v
	}

	ctx := context.Background()
	if _, err := p.fcm.Send(ctx, &message); err != nil {
		p.dev.Remove(d.ID)
	}
}
