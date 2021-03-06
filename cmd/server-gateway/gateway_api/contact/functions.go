package nestedServiceContact

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/model"
)

// @Command:	contact/add
// @Input:	contact_id		string	*
func (s *ContactService) addContact(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var contactID string
	if v, ok := request.Data["contact_id"].(string); ok {
		contactID = v
		if !_Model.Account.Exists(contactID) {
			response.Error(nested.ERR_INVALID, []string{"contact_id"})
			return
		}
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"contact_id"})
		return
	}
	if _Model.Contact.AddContact(requester.ID, contactID) {
		response.Ok()
	} else {
		response.Error(nested.ERR_UNKNOWN, []string{})
	}
	return
}

// @Command:	contact/add_favorite
// @Input:	contact_id		string	*
func (s *ContactService) addContactToFavorite(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var contactID string
	if v, ok := request.Data["contact_id"].(string); ok {
		contactID = v
		if _Model.Account.Exists(contactID) {
			if !_Model.Contact.IsContact(requester.ID, contactID) {
				response.Error(nested.ERR_ACCESS, []string{"must_be_contact_first"})
				return
			}
		} else {
			response.Error(nested.ERR_INVALID, []string{"contact_id"})
			return
		}
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"contact_id"})
		return
	}
	if _Model.Contact.AddContactToFavorite(requester.ID, contactID) {
		response.Ok()
	} else {
		response.Error(nested.ERR_UNKNOWN, []string{})
	}
	return
}

// @Command:	contact/get
// @Input:	contact_id		string	*
func (s *ContactService) getContact(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var contactID string
	if v, ok := request.Data["contact_id"].(string); ok {
		contactID = v
		if !_Model.Account.Exists(contactID) {
			response.Error(nested.ERR_INVALID, []string{"contact_id"})
			return
		}
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"contact_id"})
		return
	}
	c := _Model.Account.GetByID(contactID, nil)
	response.OkWithData(s.Worker().Map().Contact(requester, *c))
}

// @Command:	contact/get_all
// @Input:	hash		string	+
func (s *ContactService) getAllContacts(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var hash string
	if v, ok := request.Data["hash"].(string); ok {
		hash = v
	}
	c := _Model.Contact.GetContacts(requester.ID)
	if c.Hash == hash {
		response.Ok()
		return
	}
	r := make([]nested.M, 0, len(c.Contacts))
	iStart := 0
	iLength := nested.DEFAULT_MAX_RESULT_LIMIT
	iEnd := iStart + iLength
	if iEnd > len(c.Contacts) {
		iEnd = len(c.Contacts)
	}
	for {
		for _, account := range _Model.Account.GetAccountsByIDs(c.Contacts[iStart:iEnd]) {
			r = append(r, s.Worker().Map().Contact(requester, account))
		}
		iStart += iLength
		iEnd = iStart + iLength
		if iStart >= len(c.Contacts) {
			break
		}
		if iEnd > len(c.Contacts) {
			iEnd = len(c.Contacts)
		}
	}
	response.OkWithData(nested.M{
		"contacts": r,
		"hash":     c.Hash,
	})
}

// @Command:	contact/remove_contact
// @Input:	contact_id		string	*
func (s *ContactService) removeContact(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var contactID string
	if v, ok := request.Data["contact_id"].(string); ok {
		contactID = v
		if _Model.Account.Exists(contactID) {
			if !_Model.Contact.IsContact(requester.ID, contactID) {
				response.Error(nested.ERR_ACCESS, []string{"must_be_contact_first"})
				return
			}
		} else {
			response.Error(nested.ERR_INVALID, []string{"contact_id"})
			return
		}
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"contact_id"})
		return
	}
	if _Model.Contact.RemoveContact(requester.ID, contactID) {
		response.Ok()
	} else {
		response.Error(nested.ERR_UNKNOWN, []string{})
	}
	return

}

// @Command:	contact/remove_favorite
// @Input:	contact_id		string	*
func (s *ContactService) removeContactFromFavorite(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var contactID string
	if v, ok := request.Data["contact_id"].(string); ok {
		contactID = v
		if _Model.Account.Exists(contactID) {
			if !_Model.Contact.IsContact(requester.ID, contactID) {
				response.Error(nested.ERR_ACCESS, []string{"must_be_contact_first"})
				return
			}
		} else {
			response.Error(nested.ERR_INVALID, []string{"contact_id"})
			return
		}
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"contact_id"})
		return
	}
	if _Model.Contact.RemoveContactFromFavorite(requester.ID, contactID) {
		response.Ok()
	} else {
		response.Error(nested.ERR_UNKNOWN, []string{})
	}
	return

}
