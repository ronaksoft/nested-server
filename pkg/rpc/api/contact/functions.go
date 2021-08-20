package nestedServiceContact

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
)

// @Command:	contact/add
// @Input:	contact_id		string	*
func (s *ContactService) addContact(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var contactID string
	if v, ok := request.Data["contact_id"].(string); ok {
		contactID = v
		if !s.Worker().Model().Account.Exists(contactID) {
			response.Error(global.ErrInvalid, []string{"contact_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"contact_id"})
		return
	}
	if s.Worker().Model().Contact.AddContact(requester.ID, contactID) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
	return
}

// @Command:	contact/add_favorite
// @Input:	contact_id		string	*
func (s *ContactService) addContactToFavorite(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var contactID string
	if v, ok := request.Data["contact_id"].(string); ok {
		contactID = v
		if s.Worker().Model().Account.Exists(contactID) {
			if !s.Worker().Model().Contact.IsContact(requester.ID, contactID) {
				response.Error(global.ErrAccess, []string{"must_be_contact_first"})
				return
			}
		} else {
			response.Error(global.ErrInvalid, []string{"contact_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"contact_id"})
		return
	}
	if s.Worker().Model().Contact.AddContactToFavorite(requester.ID, contactID) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
	return
}

// @Command:	contact/get
// @Input:	contact_id		string	*
func (s *ContactService) getContact(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var contactID string
	if v, ok := request.Data["contact_id"].(string); ok {
		contactID = v
		if !s.Worker().Model().Account.Exists(contactID) {
			response.Error(global.ErrInvalid, []string{"contact_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"contact_id"})
		return
	}
	c := s.Worker().Model().Account.GetByID(contactID, nil)
	response.OkWithData(s.Worker().Map().Contact(requester, *c))
}

// @Command:	contact/get_all
// @Input:	hash		string	+
func (s *ContactService) getAllContacts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var hash string
	if v, ok := request.Data["hash"].(string); ok {
		hash = v
	}
	c := s.Worker().Model().Contact.GetContacts(requester.ID)
	if c.Hash == hash {
		response.Ok()
		return
	}
	r := make([]tools.M, 0, len(c.Contacts))
	iStart := 0
	iLength := global.DefaultMaxResultLimit
	iEnd := iStart + iLength
	if iEnd > len(c.Contacts) {
		iEnd = len(c.Contacts)
	}
	for {
		for _, account := range s.Worker().Model().Account.GetAccountsByIDs(c.Contacts[iStart:iEnd]) {
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
	response.OkWithData(tools.M{
		"contacts": r,
		"hash":     c.Hash,
	})
}

// @Command:	contact/remove_contact
// @Input:	contact_id		string	*
func (s *ContactService) removeContact(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var contactID string
	if v, ok := request.Data["contact_id"].(string); ok {
		contactID = v
		if s.Worker().Model().Account.Exists(contactID) {
			if !s.Worker().Model().Contact.IsContact(requester.ID, contactID) {
				response.Error(global.ErrAccess, []string{"must_be_contact_first"})
				return
			}
		} else {
			response.Error(global.ErrInvalid, []string{"contact_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"contact_id"})
		return
	}
	if s.Worker().Model().Contact.RemoveContact(requester.ID, contactID) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
	return

}

// @Command:	contact/remove_favorite
// @Input:	contact_id		string	*
func (s *ContactService) removeContactFromFavorite(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var contactID string
	if v, ok := request.Data["contact_id"].(string); ok {
		contactID = v
		if s.Worker().Model().Account.Exists(contactID) {
			if !s.Worker().Model().Contact.IsContact(requester.ID, contactID) {
				response.Error(global.ErrAccess, []string{"must_be_contact_first"})
				return
			}
		} else {
			response.Error(global.ErrInvalid, []string{"contact_id"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"contact_id"})
		return
	}
	if s.Worker().Model().Contact.RemoveContactFromFavorite(requester.ID, contactID) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
	return

}
