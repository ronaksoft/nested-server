package pusher

import (
	nested "git.ronaksoft.com/nested/server/model"
	"github.com/globalsign/mgo/bson"
)

/*
   Creation Time: 2021 - Aug - 04
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type Internal interface {
	PlaceActivity(targets []string, placeID string, action int)
	PostActivity(targets []string, postID bson.ObjectId, action nested.PostAction, placeIDs []string)
	TaskActivity(targets []string, taskID bson.ObjectId, action nested.TaskAction)
	Notification(targets []string, notificationType int)
}

type External interface {
	Notification(n *Notification)
	PlaceActivityPostAdded(post *nested.Post)
	PlaceActivityPostAttached(post *nested.Post, placeIDs []string)
	Clear(n *Notification)
	ClearAll(accountID string)
}
