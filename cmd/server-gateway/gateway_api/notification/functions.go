package nestedServiceNotification

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"strings"

	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/model"
)

// @Command:	notification/get
// @Input:	notification_id		string	*
func (s *NotificationService) getNotificationByID(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var notif *nested.Notification
	if v, ok := request.Data["notification_id"].(string); ok {
		notif = _Model.Notification.GetByID(v)
		if notif == nil {
			response.Error(global.ErrInvalid, []string{"notification_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"notification_id"})
		return
	}
	response.OkWithData(s.Worker().Map().Notification(requester, *notif))
	return
}

// @Command:	notification/get_all
// @Input:	only_unread		bool		+
// @Input:	details			bool		+
// @Input:	subject			string	+	‌("all", "task", "post")
func (s *NotificationService) getNotificationsByAccountID(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var only_unreads, details bool
	var subject string
	if v, ok := request.Data["only_unread"].(bool); ok {
		only_unreads = v
	}
	if v, ok := request.Data["details"].(bool); ok {
		details = v
	}
	if v, ok := request.Data["subject"].(string); ok {
		subject = strings.ToLower(v)
		switch subject {
		case "task":
		case "post":
		default:
			subject = "all"
		}
	}

	pg := s.Worker().Argument().GetPagination(request)
	notifications := _Model.Notification.GetByAccountID(requester.ID, pg, only_unreads, subject)
	if details {
		r := make([]tools.M, 0, pg.GetLimit())
		for _, n := range notifications {
			r = append(r, s.Worker().Map().Notification(requester, n))
		}
		d := tools.M{
			"skip":                 pg.GetSkip(),
			"limit":                pg.GetLimit(),
			"total_notifications":  requester.Counters.TotalNotifications,
			"unread_notifications": requester.Counters.UnreadNotifications,
			"notifications":        r,
		}
		response.OkWithData(d)
	} else {
		d := tools.M{
			"skip":                 pg.GetSkip(),
			"limit":                pg.GetLimit(),
			"total_notifications":  requester.Counters.TotalNotifications,
			"unread_notifications": requester.Counters.UnreadNotifications,
			"notifications":        notifications,
		}
		response.OkWithData(d)
	}
	return
}

// @Command:	notification/mark_as_read
// @Input:	notification_id		string		*	(all | ID)
func (s *NotificationService) markNotificationAsRead(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	if v, ok := request.Data["notification_id"].(string); ok {
		switch v {
		case "all":
			_Model.Notification.MarkAsRead("all", requester.ID)
			go s.Worker().Pusher().ClearNotification(requester, nil)
		default:
			ids := strings.SplitN(v, ",", 100)
			for _, nid := range ids {
				notification := _Model.Notification.GetByID(nid)
				if notification != nil && notification.AccountID == requester.ID {
					_Model.Notification.MarkAsRead(nid, requester.ID)
					go s.Worker().Pusher().ClearNotification(requester, notification)
				}
			}
		}
		response.Ok()
	} else {
		response.Error(global.ErrInvalid, []string{"notification_id"})
	}
	return
}

// @Command: notification/mark_as_read_by_post
// @Input: post_id  string
func (s *NotificationService) markNotificationAsReadByPost(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var post *nested.Post

	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		response.Error(global.ErrInvalid, []string{"post_id"})
		return
	}
	notificationIDs := _Model.Notification.MarkAsReadByPostID(post.ID, requester.ID)
	for _, notificationID := range notificationIDs {
		notification := _Model.Notification.GetByID(notificationID)
		if notification != nil && notification.AccountID == requester.ID {
			_Model.Notification.MarkAsRead(notificationID, requester.ID)
			go s.Worker().Pusher().ClearNotification(requester, notification)
		}
	}
	response.Ok()
}

// @Command:	notification/remove
// @Input:	notification_id		string	*
func (s *NotificationService) removeNotification(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	if v, ok := request.Data["notification_id"].(string); ok {
		ids := strings.SplitN(v, ",", 100)
		for _, nid := range ids {
			notification := _Model.Notification.GetByID(nid)
			if notification != nil && notification.AccountID == requester.ID {
				_Model.Notification.Remove(nid)
			}
		}
		response.Ok()
	} else {
		response.Error(global.ErrInvalid, []string{"notification_id"})
	}

}

// @Command:	notification/reset_counter
func (s *NotificationService) resetNotificationCounter(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	_Model.Account.ResetUnreadNotificationCounter(requester.ID)
	response.Ok()
}

// @Command:	notification/get_counter
func (s *NotificationService) getNotificationCounter(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	account := _Model.Account.GetByID(requester.ID, tools.M{"counters": 1})
	if account != nil {
		response.OkWithData(tools.M{
			"unread_notifications": account.Counters.UnreadNotifications,
			"total_notifications":  account.Counters.TotalNotifications,
		})
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
	return
}
