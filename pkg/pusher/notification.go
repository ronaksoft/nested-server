package pusher

import (
	nested "git.ronaksoft.com/nested/server/model"
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
