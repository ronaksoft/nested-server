package nestedServiceAccount

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"regexp"
	"strings"
	"time"

	"git.ronaksoft.com/nested/server/nested"
)

// @Command: account/available
// @Input:	account_id		string		*
func (s *AccountService) accountIDAvailable(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var accountID string
	if v, ok := request.Data["account_id"].(string); ok {
		accountID = strings.ToLower(v)
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}
	if _Model.Account.Available(accountID) {
		response.Ok()
	} else {
		response.Error(global.ErrUnavailable, []string{"account_id"})
	}
	return
}

// @Command: account/trust_email
// @Input:  email_addr		string		*
// @Input:  domain            bool       *
func (s *AccountService) addToTrustList(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var emailAddr string
	if v, ok := request.Data["email_addr"].(string); ok {
		emailAddr = v
		if !nested.IsValidEmail(emailAddr) {
			response.Error(global.ErrInvalid, []string{"email_addr"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"email_addr"})
		return
	}
	if v, ok := request.Data["domain"].(bool); ok && v {
		emailParts := strings.SplitAfter(emailAddr, "@")
		if len(emailParts) == 2 {
			emailAddr = fmt.Sprintf("@%s", emailParts[1])
		}
	}
	_Model.Account.TrustRecipient(requester.ID, []string{emailAddr})
	response.Ok()
}

// @Command: account/change_phone
// @Input:	vid				string		*
// @Input:	pass			string		*
// @Input:	phone			string		*
func (s *AccountService) changePhone(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var verification *nested.Verification
	var password, phone string
	if !requester.Privacy.ChangeProfile {
		response.Error(global.ErrAccess, []string{"contact_admin"})
		return
	}
	if v, ok := request.Data["vid"].(string); ok {
		verification = _Model.Verification.GetByID(v)
		if verification == nil {
			response.Error(global.ErrInvalid, []string{"vid"})
			return
		}
		// check verification object is verified
		if !verification.Verified || verification.Expired {
			response.Error(global.ErrInvalid, []string{"vid"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"vid"})
		return
	}
	if v, ok := request.Data["pass"].(string); ok {
		password = v
		if !_Model.Account.Verify(requester.ID, password) {
			response.Error(global.ErrInvalid, []string{"pass"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"pass"})
		return
	}
	if v, ok := request.Data["phone"].(string); ok {
		phone = v
		if verification.Phone != phone {
			response.Error(global.ErrInvalid, []string{"phone"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"phone"})
		return
	}
	if _Model.Account.SetPhone(requester.ID, phone) {
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
}

// @Command: account/get
// @Input:	account_id		string		*
func (s *AccountService) getAccountInfo(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var d tools.M
	var details bool
	var account *nested.Account
	if id, ok := request.Data["account_id"].(string); ok && id != requester.ID {
		if acc := _Model.Account.GetByID(id, nil); acc != nil {
			account = acc
		} else {
			if strings.Index(id, "@") != -1 {
				d = tools.M{
					"_id": id,
				}
				response.OkWithData(d)
				return
			} else {
				response.Error(global.ErrUnavailable, []string{"account_id"})
				return
			}
		}
	} else {
		account = requester
		details = true
	}
	response.OkWithData(s.Worker().Map().Account(*account, details))
	return
}

// @Command: account/get_many
// @Input:	account_id		string	*	(comma separated)
func (s *AccountService) getManyAccountsInfo(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var accounts []nested.Account
	if v, ok := request.Data["account_id"].(string); ok {
		inputs := strings.SplitN(v, ",", global.DefaultMaxResultLimit)
		accountIDs := make([]string, 0, len(inputs))
		for _, input := range inputs {
			if strings.Index(input, "@") == -1 {
				accountIDs = append(accountIDs, input)
			}
		}
		accounts = _Model.Account.GetAccountsByIDs(accountIDs)
		if len(accounts) == 0 {
			response.OkWithData(tools.M{"accounts": []tools.M{}})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}
	r := make([]tools.M, 0, len(accounts))
	for _, account := range accounts {
		r = append(r, s.Worker().Map().Account(account, false))
	}
	response.OkWithData(tools.M{"accounts": r})
}

// @Command: account/get_by_token
// @Input:	token		string	*
func (s *AccountService) getAccountInfoByToken(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var account *nested.Account
	if v, ok := request.Data["token"].(string); ok {
		account = _Model.Account.GetAccountByLoginToken(v)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"token"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"token"})
		return
	}
	response.OkWithData(s.Worker().Map().Account(*account, true))
}

// @Command: account/get_posts
// @Input:	by_update		string		+
// @Pagination
func (s *AccountService) getAccountPosts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var sort_item string
	if _, ok := request.Data["by_update"]; ok {
		sort_item = nested.PostSortLastUpdate
	} else {
		sort_item = nested.PostSortTimestamp
	}
	pg := s.Worker().Argument().GetPagination(request)
	posts := _Model.Post.GetPostsOfPlaces(append(requester.AccessPlaceIDs, "*"), sort_item, pg)
	r := make([]tools.M, 0, len(posts))
	for _, post := range posts {
		r = append(r, s.Worker().Map().Post(requester, post, true))
	}
	response.OkWithData(tools.M{
		"skip":  pg.GetSkip(),
		"limit": pg.GetLimit(),
		"posts": r,
	})
	return
}

// @Command: account/get_pinned_posts
// @Pagination
func (s *AccountService) getAccountPinnedPosts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	pg := s.Worker().Argument().GetPagination(request)
	posts := _Model.Post.GetPinnedPosts(requester.ID, pg)
	r := make([]tools.M, 0, len(posts))
	for _, post := range posts {
		r = append(r, s.Worker().Map().Post(requester, post, true))
	}
	response.OkWithData(tools.M{
		"skip":  pg.GetSkip(),
		"limit": pg.GetLimit(),
		"posts": r,
	})
	return
}

// @Command: account/get_favorite_posts
// @Input:	by_update		string		+
// @Pagination
func (s *AccountService) getAccountFavoritePosts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var sort_item string

	if _, ok := request.Data["by_update"]; ok {
		sort_item = nested.PostSortLastUpdate
	} else {
		sort_item = nested.PostSortTimestamp
	}

	pg := s.Worker().Argument().GetPagination(request)
	posts := _Model.Post.GetPostsOfPlaces(append(requester.BookmarkedPlaceIDs, "*"), sort_item, pg)
	r := make([]tools.M, 0, len(posts))
	for _, post := range posts {
		r = append(r, s.Worker().Map().Post(requester, post, true))
	}
	response.OkWithData(tools.M{
		"skip":  pg.GetSkip(),
		"limit": pg.GetLimit(),
		"posts": r,
	})
	return
}

// @Command: account/get_spam_posts
// @Pagination
func (s *AccountService) getAccountSpamPosts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {

	pg := s.Worker().Argument().GetPagination(request)
	posts := _Model.Post.GetSpamPostsOfPlaces(append(requester.BookmarkedPlaceIDs, "*"), pg)
	r := make([]tools.M, 0, len(posts))
	for _, post := range posts {
		r = append(r, s.Worker().Map().Post(requester, post, true))
	}
	response.OkWithData(tools.M{
		"skip":  pg.GetSkip(),
		"limit": pg.GetLimit(),
		"posts": r,
	})
	return
}

// @Command: account/get_sent_posts
// @Pagination
func (s *AccountService) getAccountSentPosts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	pg := s.Worker().Argument().GetPagination(request)
	posts := _Model.Post.GetPostsBySender(requester.ID, nested.PostSortTimestamp, pg)
	r := make([]tools.M, 0, len(posts))
	for _, post := range posts {
		r = append(r, s.Worker().Map().Post(requester, post, true))
	}
	response.OkWithData(tools.M{
		"skip":  pg.GetSkip(),
		"limit": pg.GetLimit(),
		"posts": r,
	})
	return
}

// @Command: account/get_all_places
// @Input:	with_children		bool		+
// @Input:	filter				string		+	(creator | key_holder | all)
func (s *AccountService) getAccountAllPlaces(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var d []tools.M
	var filter string
	var withChildren bool
	if v, ok := request.Data["with_children"].(bool); ok {
		withChildren = v
	}
	if v, ok := request.Data["filter"].(string); ok {
		filter = v
	} else {
		filter = nested.MemberTypeAll
	}
	switch filter {
	case nested.MemberTypeCreator:
		places := _Model.Place.GetPlacesByIDs(requester.AccessPlaceIDs)
		for _, place := range places {
			if place.IsCreator(requester.ID) {
				d = append(d, s.Worker().Map().Place(requester, place, place.GetAccess(requester.ID)))
			}
		}
	case nested.MemberTypeKeyHolder:
		places := _Model.Place.GetPlacesByIDs(requester.AccessPlaceIDs)
		for _, place := range places {
			if place.IsKeyholder(requester.ID) {
				d = append(d, s.Worker().Map().Place(requester, place, place.GetAccess(requester.ID)))
			}
		}
	case nested.MemberTypeAll:
		fallthrough
	default:
		places := _Model.Place.GetPlacesByIDs(requester.AccessPlaceIDs)
		for _, place := range places {
			if !place.IsGrandPlace() && !withChildren {
				continue
			}
			if place.IsGrandPlace() {
				if withChildren {
					unlockedPlaces := _Model.Place.GetPlacesByIDs(place.UnlockedChildrenIDs)
					for _, unlockedPlace := range unlockedPlaces {
						if !unlockedPlace.IsMember(requester.ID) {
							d = append(d, s.Worker().Map().Place(requester, unlockedPlace, unlockedPlace.GetAccess(requester.ID)))
						}
					}
				}
			}
			d = append(d, s.Worker().Map().Place(requester, place, place.GetAccess(requester.ID)))
		}
		filter = nested.MemberTypeAll
	}
	response.OkWithData(tools.M{"places": d})
	return
}

// @Command: account/get_favorite_places
func (s *AccountService) getAccountFavoritePlaces(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	r := make([]tools.M, 0)
	places := _Model.Place.GetPlacesByIDs(requester.BookmarkedPlaceIDs)
	for _, place := range places {
		r = append(r, s.Worker().Map().Place(requester, place, place.GetAccess(requester.ID)))
	}

	response.OkWithData(tools.M{"places": r})

	return
}

// @Command: account/set_password
// @Input:	old_pass	string	*
// @Input:	new_pass	string	*
func (s *AccountService) setAccountPassword(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var oldPass, newPass, accountID string
	var account *nested.Account
	if v, ok := request.Data["old_pass"].(string); ok {
		oldPass = v
	} else {
		response.Error(global.ErrInvalid, []string{"old_pass"})
		return
	}
	if v, ok := request.Data["new_pass"].(string); ok {
		newPass = v
	} else {
		response.Error(global.ErrInvalid, []string{"new_pass"})
		return
	}
	if v, ok := request.Data["account_id"].(string); ok {
		accountID = v
		account = s.Worker().Model().Account.GetByID(v, nil)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"account_id"})
			return
		} else if account.Disabled {
			response.Error(global.ErrAccess, []string{"account_is_disabled"})
			return
		}
	} else {
		if requester != nil {
			accountID = requester.ID
		} else {
			response.Error(global.ErrIncomplete, []string{"account_id"})
			return
		}
	}

	if _Model.Account.Verify(accountID, oldPass) {
		_Model.Account.SetPassword(accountID, newPass)
	} else {
		response.Error(global.ErrInvalid, []string{})
		return
	}
	response.Ok()
}

// @Command: account/set_password_by_token
// @Input:	token			string		*
// @Input:	new_pass		string		*
func (s *AccountService) setAccountPasswordByLoginToken(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var account *nested.Account
	var token, newPass string
	if v, ok := request.Data["token"].(string); ok {
		token = v
		account = _Model.Account.GetAccountByLoginToken(token)
		if account == nil {
			response.Error(global.ErrInvalid, []string{"token"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"token"})
		return
	}
	if v, ok := request.Data["new_pass"].(string); ok {
		newPass = v
	} else {
		response.Error(global.ErrIncomplete, []string{"new_pass"})
		return
	}
	if _Model.Account.SetPassword(account.ID, newPass) {
		// remove the login token from db, prevent from using it in future
		_Model.Token.RevokeLoginToken(token)
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
	return
}

// @Command: account/set_picture
// @Input:	universal_id		string			*
func (s *AccountService) setAccountPicture(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var uni_id nested.UniversalID
	if !requester.Privacy.ChangePicture {
		response.Error(global.ErrAccess, []string{"contact_admin"})
		return
	}
	if v, ok := request.Data["universal_id"].(string); ok {
		uni_id = nested.UniversalID(v)
		if !_Model.File.Exists(uni_id) {
			response.Error(global.ErrUnavailable, []string{"universal_id"})
			return
		}
	}
	f := _Model.File.GetByID(uni_id, nil)
	_Model.Account.SetPicture(requester.ID, f.Thumbnails)
	if requester.Privacy.Searchable {
		_Model.Search.AddPlaceToSearchIndex(requester.ID, fmt.Sprintf("%s %s", requester.FirstName, requester.LastName), f.Thumbnails)
	}
	response.Ok()
	return
}

// @Command: account/remove_picture
func (s *AccountService) removeAccountPicture(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	pic := nested.Picture{}
	_Model.Account.SetPicture(requester.ID, pic)
	response.Ok()
	return
}

// @Command: account/untrust_email
// @Input:	email_addr		string			*
func (s *AccountService) removeFromTrustList(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var emailAddr string
	if v, ok := request.Data["email_addr"].(string); ok {
		emailAddr = v
		if !nested.IsValidEmail(emailAddr) {
			response.Error(global.ErrInvalid, []string{"email_addr"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"email_addr"})
		return
	}
	_Model.Account.UnTrustRecipient(requester.ID, []string{emailAddr})
	response.Ok()
}

// @Command: account/register_device
// @Input:	_dt		string 		*	(device token)
// @Input:	_did	    string 		*	(device id)
// @Input:	_os		string 		*	(android | ios | chrome | firefox | safari | opera | edge)
func (s *AccountService) registerDevice(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var deviceID, deviceToken, deviceOS string
	if v, ok := request.Data["_dt"].(string); ok {
		deviceToken = v
	}
	if v, ok := request.Data["_did"].(string); ok {
		deviceID = v
	}
	if v, ok := request.Data["_os"].(string); ok {
		deviceOS = v
	}

	s.Worker().Pusher().RegisterDevice(deviceID, deviceToken, deviceOS, requester.ID)
	response.Ok()
}

// @Command: account/unregister_device
// @Input:	_dt		string 		*	(device token)
// @Input:	_did	string 		*	(device id)
func (s *AccountService) unregisterDevice(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var deviceID, deviceToken string
	if v, ok := request.Data["_dt"].(string); ok {
		deviceToken = v
	}
	if v, ok := request.Data["_did"].(string); ok {
		deviceID = v
	}
	s.Worker().Pusher().UnregisterDevice(deviceID, deviceToken, requester.ID)
	response.Ok()
}

// @Command: account/update
// @Input:	fname		string			+
// @Input:	lname		string			+
// @Input:	gender		string			+	(m | f | o | x)
// @Input:	dob			string			+	(YYYY-MM-DD)
// @Input:	email		string			+
// @Input:	searchable	bool			+
func (s *AccountService) updateAccount(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	aur := nested.AccountUpdateRequest{}
	placeUpdateRequest := tools.M{}
	if !requester.Privacy.ChangeProfile {
		response.Error(global.ErrAccess, []string{"contact_admin"})
		return
	}
	if fname, ok := request.Data["fname"].(string); ok {
		fname = strings.TrimSpace(fname)
		if len(fname) > 0 {
			aur.FirstName = fname
		}
		if len(fname) > global.DefaultMaxAccountName {
			aur.FirstName = fname[:global.DefaultMaxAccountName]
		}
	}
	if lname, ok := request.Data["lname"].(string); ok {
		lname = strings.TrimSpace(lname)
		if len(lname) > 0 {
			aur.LastName = lname
		}
		if len(lname) > global.DefaultMaxAccountName {
			aur.LastName = lname[:global.DefaultMaxAccountName]
		}
	}
	if gender, ok := request.Data["gender"].(string); ok && gender != "" {
		switch gender {
		case "m", "male", "man", "boy":
			gender = "m"
		case "f", "female", "woman", "girl":
			gender = "f"
		case "o", "other":
			gender = "o"
		default:
			gender = "x"
		}
		aur.Gender = gender
	}
	if dob, ok := request.Data["dob"].(string); ok {
		if _, err := time.Parse("2006-01-02", dob); err == nil {
			aur.DateOfBirth = dob
		}
	}
	if email, ok := request.Data["email"].(string); ok {
		email = strings.Trim(email, " ")
		if b, err := regexp.MatchString(global.DefaultRegexEmail, email); err == nil && b {
			aur.Email = email
		}
	}
	if searchable, ok := request.Data["searchable"].(bool); ok {
		if searchable {
			_Model.Search.AddPlaceToSearchIndex(requester.ID, fmt.Sprintf("%s %s", requester.FirstName, requester.LastName), requester.Picture)
			placeUpdateRequest["privacy.search"] = true
		} else {
			_Model.Search.RemovePlaceFromSearchIndex(requester.ID)
			placeUpdateRequest["privacy.search"] = false
		}
		_Model.Account.SetPrivacy(requester.ID, "searchable", searchable)
	}
	_Model.Account.Update(requester.ID, aur)
	_Model.Place.Update(requester.ID, placeUpdateRequest)

	if requester.Privacy.Searchable && (aur.FirstName != "" || aur.LastName != "") {
		_Model.Search.AddPlaceToSearchIndex(requester.ID, fmt.Sprintf("%s %s", requester.FirstName, requester.LastName), requester.Picture)
	}

	response.Ok()
	return
}

// @Command: account/update_email
// @Input:	host			string			+
// @Input:	port			int				+
// @Input:	username		string			*
// @Input:	password		string			*
// @Input:	status			bool			*
func (s *AccountService) updateEmail(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var host, username, password string
	var port int
	var status bool
	if u, ok := request.Data["username"].(string); ok {
		if u == "" {
			username = requester.Mail.OutgoingSMTPUser
		} else {
			u = strings.TrimSpace(u)
			u = strings.ToLower(u)
			index := strings.Index(u, "@")
			if len(u) == 0 || index == -1 {
				response.Error(global.ErrInvalid, []string{"user-name"})
				return
			} else {
				username = u
				if !nested.IsValidEmail(username) {
					response.Error(global.ErrInvalid, []string{"user-name"})
					return
				}
				switch u[index+1:] {
				case "gmail.com":
					host = "smtp.gmail.com"
					port = 465
				case "yahoo.com":
					host = "smtp.yahoo.com"
					port = 465
				default:
					if h, ok := request.Data["host"].(string); ok {
						host = h
					} else {
						response.Error(global.ErrInvalid, []string{"host"})
						return
					}
					if p, ok := request.Data["port"].(int); ok {
						port = p
					} else {
						response.Error(global.ErrInvalid, []string{"port"})
						return
					}
				}
			}
		}
	} else {
		response.Error(global.ErrInvalid, []string{"user-name"})
		return
	}
	if p, ok := request.Data["password"].(string); ok {
		if p == "" {
			password = nested.Decrypt(nested.EMAIL_ENCRYPT_KEY, requester.Mail.OutgoingSMTPPass)
		} else {
			password = p
		}
	} else {
		response.Error(global.ErrInvalid, []string{"password"})
		return
	}
	if p, ok := request.Data["status"].(bool); ok {
		status = p
	} else {
		response.Error(global.ErrInvalid, []string{"status"})
		return
	}
	accountMail := nested.AccountMail{
		Active:           status,
		OutgoingSMTPHost: host,
		OutgoingSMTPPort: port,
		OutgoingSMTPUser: username,
		OutgoingSMTPPass: password,
	}
	if _Model.Account.UpdateEmail(requester.ID, accountMail) {
		response.Ok()
		return
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
}

// @Command: account/remove_email
func (s *AccountService) removeEmail(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	accountMail := nested.AccountMail{
		Active:           false,
		OutgoingSMTPHost: "",
		OutgoingSMTPPort: 0,
		OutgoingSMTPUser: "",
		OutgoingSMTPPass: "",
	}
	if _Model.Account.UpdateEmail(requester.ID, accountMail) {
		response.Ok()
		return
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
}
