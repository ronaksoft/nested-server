package api

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"github.com/globalsign/mgo/bson"
	"strconv"
	"strings"
)

/*
   Creation Time: 2021 - Aug - 17
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

// ArgumentHandler provides functions for easy argument extraction
type ArgumentHandler struct {
	worker *Worker
}

func NewArgumentHandler(worker *Worker) *ArgumentHandler {
	ah := new(ArgumentHandler)
	ah.worker = worker
	return ah
}

func (ae *ArgumentHandler) GetAccount(request *rpc.Request, response *rpc.Response) *nested.Account {
	var account *nested.Account
	if accountID, ok := request.Data["account_id"].(string); ok {
		account = ae.worker.Model().Account.GetByID(accountID, nil)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return nil
	}
	return account
}

func (ae *ArgumentHandler) GetAccounts(request *rpc.Request, response *rpc.Response) []nested.Account {
	var uniqueAccountIDs []string
	var accounts []nested.Account
	if csAccountIDs, ok := request.Data["account_id"].(string); ok {
		accountIDs := strings.SplitN(csAccountIDs, ",", global.DefaultMaxResultLimit)
		mapAccountIDs := tools.MB{}
		for _, accountID := range accountIDs {
			mapAccountIDs[accountID] = true
		}
		for accountID := range mapAccountIDs {
			uniqueAccountIDs = append(uniqueAccountIDs, accountID)
		}
		accounts = ae.worker.Model().Account.GetAccountsByIDs(uniqueAccountIDs)
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
	}
	return accounts
}

func (ae *ArgumentHandler) GetAccountIDs(request *rpc.Request, response *rpc.Response) []string {
	var uniqueAccountIDs []string
	if csAccountIDs, ok := request.Data["account_id"].(string); ok {
		accountIDs := strings.SplitN(csAccountIDs, ",", global.DefaultMaxResultLimit)
		mapAccountIDs := tools.MB{}
		for _, accountID := range accountIDs {
			mapAccountIDs[accountID] = true
		}
		for accountID := range mapAccountIDs {
			uniqueAccountIDs = append(uniqueAccountIDs, accountID)
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
	}
	return uniqueAccountIDs
}

func (ae *ArgumentHandler) GetComment(request *rpc.Request, response *rpc.Response) *nested.Comment {
	var comment *nested.Comment
	if commentID, ok := request.Data["comment_id"].(string); ok {
		if bson.IsObjectIdHex(commentID) {
			comment = ae.worker.Model().Post.GetCommentByID(bson.ObjectIdHex(commentID))
			if comment == nil {
				response.Error(global.ErrUnavailable, []string{"comment_id"})
				return nil
			}
		} else {
			response.Error(global.ErrInvalid, []string{"comment_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"comment_id"})
		return nil
	}
	return comment
}

func (ae *ArgumentHandler) GetLabel(request *rpc.Request, response *rpc.Response) *nested.Label {
	var label *nested.Label
	if labelID, ok := request.Data["label_id"].(string); ok {
		label = ae.worker.Model().Label.GetByID(labelID)
		if label == nil {
			response.Error(global.ErrInvalid, []string{"label_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"label_id"})
		return nil
	}
	return label
}

func (ae *ArgumentHandler) GetLabelRequest(request *rpc.Request, response *rpc.Response) *nested.LabelRequest {
	var labelRequest *nested.LabelRequest

	if labelRequestID, ok := request.Data["request_id"].(string); ok {
		if !bson.IsObjectIdHex(labelRequestID) {
			response.Error(global.ErrInvalid, []string{"request_id"})
			return nil
		}
		labelRequest = ae.worker.Model().Label.GetRequestByID(bson.ObjectIdHex(labelRequestID))
		if labelRequest == nil {
			response.Error(global.ErrInvalid, []string{"request_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"request_id"})
		return nil
	}
	return labelRequest
}

func (ae *ArgumentHandler) GetPlace(request *rpc.Request, response *rpc.Response) *nested.Place {
	var place *nested.Place
	if placeID, ok := request.Data["place_id"].(string); ok {
		place = ae.worker.Model().Place.GetByID(placeID, nil)
		if place == nil {
			response.Error(global.ErrInvalid, []string{"place_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"place_id"})
		return nil
	}
	return place
}

func (ae *ArgumentHandler) GetPost(request *rpc.Request, response *rpc.Response) *nested.Post {
	var post *nested.Post
	if postID, ok := request.Data["post_id"].(string); ok {
		if bson.IsObjectIdHex(postID) {
			post = ae.worker.Model().Post.GetPostByID(bson.ObjectIdHex(postID))
			if post == nil {
				response.Error(global.ErrUnavailable, []string{"post_id"})
				return nil
			}
		} else {
			response.Error(global.ErrInvalid, []string{"post_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"post_id"})
		return nil
	}
	return post
}

func (ae *ArgumentHandler) GetPagination(request *rpc.Request) nested.Pagination {
	pg := nested.NewPagination(0, 0, 0, 0)
	if v, ok := request.Data["skip"].(float64); ok {
		pg.SetSkip(int(v))
	} else if v, ok := request.Data["skip"].(string); ok {
		skip, _ := strconv.Atoi(v)
		pg.SetSkip(skip)
	}
	if v, ok := request.Data["limit"].(float64); ok {
		pg.SetLimit(int(v))
	} else if v, ok := request.Data["limit"].(string); ok {
		limit, _ := strconv.Atoi(v)
		pg.SetLimit(limit)
	}
	if v, ok := request.Data["after"].(float64); ok {
		pg.After = int64(v)
	} else if v, ok := request.Data["after"].(string); ok {
		after, _ := strconv.Atoi(v)
		pg.After = int64(after)
	}
	if v, ok := request.Data["before"].(float64); ok {
		pg.Before = int64(v)
	} else if v, ok := request.Data["before"].(string); ok {
		before, _ := strconv.Atoi(v)
		pg.Before = int64(before)
	}
	return pg
}

func (ae *ArgumentHandler) GetTask(request *rpc.Request, response *rpc.Response) *nested.Task {
	var task *nested.Task
	if taskID, ok := request.Data["task_id"].(string); ok {
		if bson.IsObjectIdHex(taskID) {
			task = ae.worker.Model().Task.GetByID(bson.ObjectIdHex(taskID))
			if task == nil {
				response.Error(global.ErrUnavailable, []string{"task_id"})
				return nil
			}
		} else {
			response.Error(global.ErrInvalid, []string{"task_id"})
			return nil
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"task_id"})
		return nil
	}
	return task
}
