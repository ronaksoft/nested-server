package nestedServiceLabel

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"strings"

	"git.ronaksoft.com/nested/server/nested"
	"github.com/globalsign/mgo/bson"
)

// @Command:	label/add_member
// @Input:	account_id		string 		*	(comma separated)
// @Input:	label_id			string		*
func (s *LabelService) addMemberToLabel(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var label *nested.Label
	var accountIDs []string
	if label = s.Worker().Argument().GetLabel(request, response); label == nil {
		return
	}
	if accountIDs = s.Worker().Argument().GetAccountIDs(request, response); len(accountIDs) == 0 {
		return
	}

	// If user is not LabelEditor then he/she cannot add member to the label
	if !requester.Authority.LabelEditor {
		response.Error(global.ErrAccess, []string{"not_label_editor"})
		return
	}

	if label.Counters.Members+len(accountIDs) > global.DefaultLabelMaxMembers {
		response.Error(global.ErrLimit, []string{"number_of_members"})
		return
	}

	var availableAccountIDs, notAvailableAccountIDs []string
	for _, accountID := range accountIDs {
		if _Model.Account.IsEnabled(accountID) {
			availableAccountIDs = append(availableAccountIDs, accountID)
		} else {
			notAvailableAccountIDs = append(notAvailableAccountIDs, accountID)
		}
	}
	if !_Model.Label.AddMembers(label.ID, availableAccountIDs) {
		response.Error(global.ErrUnknown, []string{""})
		return
	}
	response.OkWithData(tools.M{"not_available_accounts": notAvailableAccountIDs})
	// TODO:: Notification to users ?!!
}

// @Command:	label/create
// @Input:	title		string		*
// @Input:	code			string		+
// @Input:	is_public	bool			+
func (s *LabelService) createLabel(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var title, labelCode string
	var isPublic bool
	if v, ok := request.Data["title"].(string); ok {
		title = strings.TrimSpace(v)
		if len(title) == 0 {
			response.Error(global.ErrInvalid, []string{"title_length_too_small"})
			return
		} else if len(title) > global.DefaultMaxLabelTitle {
			response.Error(global.ErrInvalid, []string{"title_length_too_large"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"title"})
		return
	}
	if v, ok := request.Data["code"].(string); ok {
		labelCode = v
	}
	if v, ok := request.Data["is_public"].(bool); ok {
		isPublic = v
	}
	labelCode = _Model.Label.SanitizeLabelCode(labelCode)
	labelID := bson.NewObjectId().Hex()

	// If user is not LabelEditor then he/she cannot add member to the label
	if !requester.Authority.LabelEditor {
		response.Error(global.ErrAccess, []string{"not_label_editor"})
		return
	}

	if _Model.Label.TitleExists(title) {
		response.Error(global.ErrDuplicate, []string{"title"})
		return
	}
	if isPublic {
		if !_Model.Label.CreatePublic(labelID, title, labelCode, requester.ID) {
			response.Error(global.ErrUnknown, []string{})
			return
		}
	} else {
		if !_Model.Label.CreatePrivate(labelID, title, labelCode, requester.ID) {
			response.Error(global.ErrUnknown, []string{""})
			return
		}
	}
	response.OkWithData(tools.M{"label_id": labelID})
	return
}

// @Command:	label/request
// @Input:	label_id		string		*
// @Input:	title		string		+
// @Input:	code			string		+
func (s *LabelService) createLabelRequest(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var label *nested.Label
	var labelTitle, labelCode string
	if v, ok := request.Data["title"].(string); ok {
		v = strings.TrimSpace(v)
		//if v == "" {
		//	response.Error(global.ErrInvalid, []string{"title"})
		//	return
		//}
		if len(v) > global.DefaultMaxLabelTitle {
			response.Error(global.ErrInvalid, []string{"title_too_long"})
			return
		}
		labelTitle = v
	}
	if v, ok := request.Data["code"].(string); ok {
		labelCode = _Model.Label.SanitizeLabelCode(v)
	}

	if requester.Authority.LabelEditor {
		response.Error(global.ErrAccess, []string{"label_editor"})
		return
	}
	if labelID, ok := request.Data["label_id"].(string); ok {
		label = s.Worker().Model().Label.GetByID(labelID)
		if label == nil {
			response.Error(global.ErrInvalid, []string{"label_id"})
			return
		}
	}

	if label == nil {
		_Model.Label.CreateRequest(requester.ID, "", labelTitle, labelCode)
	} else {
		_Model.Label.CreateRequest(requester.ID, label.ID, labelTitle, labelCode)
	}
	response.Ok()
}

// @Command:	label/get_members
// @Input:	label_id		string	*
// @Pagination
func (s *LabelService) getLabelMembers(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var label *nested.Label
	if label = s.Worker().Argument().GetLabel(request, response); label == nil {
		return
	}

	// If user is not LabelEditor then he/she cannot add member to the label
	if !requester.Authority.LabelEditor {
		response.Error(global.ErrAccess, []string{"not_label_editor"})
		return
	}
	labelMembers := _Model.Account.GetAccountsByIDs(label.Members)
	var r []tools.M
	for _, member := range labelMembers {
		r = append(r, s.Worker().Map().Account(member, false))
	}
	response.OkWithData(tools.M{"members": r})
}

// @Command:	label/get_many
// @Input:	label_id		string		*	(comma separated)
func (s *LabelService) getManyLabels(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var labels []nested.Label
	if v, ok := request.Data["label_id"].(string); ok {
		labelIDs := strings.Split(v, ",")
		labels = _Model.Label.GetByIDs(labelIDs)
		if len(labels) == 0 {
			response.OkWithData(tools.M{"labels": []tools.M{}})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"label_id"})
		return
	}
	r := make([]tools.M, 0, len(labels))
	for _, label := range labels {
		details := false
		if label.IsMember(requester.ID) {
			details = true
		}
		r = append(r, s.Worker().Map().Label(requester, label, details))
	}
	response.OkWithData(tools.M{"labels": r})
}

// @Command:	label/get_requests
// @Pagination
func (s *LabelService) listLabelRequests(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	labelRequests := make([]nested.LabelRequest, 0)
	if requester.Authority.LabelEditor {
		labelRequests = _Model.Label.GetRequests(nested.LabelRequestStatusPending, s.Worker().Argument().GetPagination(request))
	} else {
		labelRequests = _Model.Label.GetRequestsByAccountID(requester.ID, s.Worker().Argument().GetPagination(request))
	}
	r := make([]tools.M, 0)
	for _, labelRequest := range labelRequests {
		r = append(r, s.Worker().Map().LabelRequest(labelRequest))
	}
	response.OkWithData(tools.M{"label_requests": r})
}

// @Command:	label/remove_request
// @Input:	request_id		string		*
func (s *LabelService) removeLabelRequest(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var labelRequest *nested.LabelRequest
	if labelRequest = s.Worker().Argument().GetLabelRequest(request, response); labelRequest == nil {
		return
	}
	if labelRequest.RequesterID == requester.ID {
		if _Model.Label.UpdateRequestStatus(requester.ID, labelRequest.ID, nested.LabelRequestStatusCanceled) {
			response.Ok()
		} else {
			response.Error(global.ErrUnknown, []string{})
		}
	}
}

// @Command: label/remove
// @Input:	label_id		string	*
func (s *LabelService) removeLabel(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var label *nested.Label
	if label = s.Worker().Argument().GetLabel(request, response); label == nil {
		return
	}

	// If user is not LabelEditor then he/she cannot add member to the label
	if !requester.Authority.LabelEditor {
		response.Error(global.ErrAccess, []string{"not_label_editor"})
		return
	}

	if _Model.Label.Remove(label.ID) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{""})
	}
}

// @Command: label/remove_member
// @Input:	label_id		string	*
// @Input:	account_id	string	*
func (s *LabelService) removeMemberFromLabel(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var label *nested.Label
	var account *nested.Account
	if label = s.Worker().Argument().GetLabel(request, response); label == nil {
		return
	}
	if account = s.Worker().Argument().GetAccount(request, response); account == nil {
		return
	}

	// If user is not LabelEditor then he/she cannot add member to the label
	if !requester.Authority.LabelEditor {
		response.Error(global.ErrAccess, []string{"not_label_editor"})
		return
	}

	if _Model.Label.RemoveMember(label.ID, account.ID) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{""})
	}

}

// @Command: label/update
// @Input:	label_id		string	*
// @Input:	code			string	+
// @Input:	title		string	*
func (s *LabelService) updateLabel(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var label *nested.Label
	var labelTitle, labelCode string
	if label = s.Worker().Argument().GetLabel(request, response); label == nil {
		return
	}
	if v, ok := request.Data["title"].(string); ok {
		if len(v) > global.DefaultMaxLabelTitle {
			response.Error(global.ErrInvalid, []string{"title_too_long"})
			return
		}
		labelTitle = v
	} else {
		response.Error(global.ErrIncomplete, []string{"title"})
		return
	}
	if v, ok := request.Data["code"].(string); ok {
		labelCode = v
	}
	labelCode = _Model.Label.SanitizeLabelCode(labelCode)

	// If user is not LabelEditor then he/she cannot add member to the label
	if !requester.Authority.LabelEditor {
		response.Error(global.ErrAccess, []string{"not_label_editor"})
		return
	}

	if _Model.Label.Update(label.ID, labelCode, labelTitle) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
}

// @Command: label/update_request
// @Input:	request_id		string	*
// @Input:	status			string	*		(approve | reject)
func (s *LabelService) updateLabelRequest(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var status string
	var labelRequest *nested.LabelRequest
	if v, ok := request.Data["status"].(string); ok {
		status = v
	} else {
		response.Error(global.ErrIncomplete, []string{"status"})
		return
	}
	if labelRequest = s.Worker().Argument().GetLabelRequest(request, response); labelRequest == nil {
		return
	}
	labelRequest.ResponderID = requester.ID
	// If user is not LabelEditor then he/she cannot add member to the label
	if !requester.Authority.LabelEditor {
		response.Error(global.ErrAccess, []string{"not_label_editor"})
		return
	}

	switch status {
	case "approve", "accept":
		// If LabelID is set:
		//	1. title is set then update label with new title and code
		//	2. title is not set then add requester to the label members
		// else
		//	Create new label
		if len(labelRequest.LabelID) > 0 {
			if label := _Model.Label.GetByID(labelRequest.LabelID); label == nil {
				_Model.Label.UpdateRequestStatus(requester.ID, labelRequest.ID, nested.LabelRequestStatusFailed)
				response.Error(global.ErrUnavailable, []string{"label_not_exists"})
				return
			}
			if len(labelRequest.Title) > 0 {
				_Model.Label.Update(labelRequest.LabelID, labelRequest.ColourCode, labelRequest.Title)
				_Model.Label.UpdateRequestStatus(requester.ID, labelRequest.ID, nested.LabelRequestStatusApproved)

				// handle push messages (notification)
				go s.Worker().Pusher().LabelRequestApproved(labelRequest)

			} else {
				_Model.Label.AddMembers(labelRequest.LabelID, []string{labelRequest.RequesterID})
				_Model.Label.UpdateRequestStatus(requester.ID, labelRequest.ID, nested.LabelRequestStatusApproved)

				// handle push messages (notification)
				go s.Worker().Pusher().LabelRequestApproved(labelRequest)
			}
			response.Ok()
			return
		} else {
			labelID := bson.NewObjectId().Hex()
			if _Model.Label.CreatePrivate(labelID, labelRequest.Title, labelRequest.ColourCode, requester.ID) {
				_Model.Label.UpdateRequestStatus(requester.ID, labelRequest.ID, nested.LabelRequestStatusApproved)
				_Model.Label.AddMembers(labelID, []string{labelRequest.RequesterID})
				labelRequest.LabelID = labelID
				// handle push messages (notification)
				go s.Worker().Pusher().LabelRequestApproved(labelRequest)

				response.OkWithData(tools.M{"label_id": labelID})
				return
			} else {
				_Model.Label.UpdateRequestStatus(requester.ID, labelRequest.ID, nested.LabelRequestStatusFailed)
			}
		}
	case "reject", "deny":
		_Model.Label.UpdateRequestStatus(requester.ID, labelRequest.ID, nested.LabelRequestStatusRejected)

		// handle push messages (notification)
		go s.Worker().Pusher().LabelRequestRejected(labelRequest)

		response.Ok()
	}
}
