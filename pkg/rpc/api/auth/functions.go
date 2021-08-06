package nestedServiceAuth

import (
	"bytes"
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/config"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"html/template"
	"log"
	"regexp"
	"strings"
	"time"

	"git.ronaksoft.com/nested/server/nested"
)

// @Command:	auth/get_verification
// @Input:	phone	string	*
// @Input:	uid 	string	*
func (s *AuthService) getPhoneVerificationCode(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var phone string
	if v, ok := request.Data["phone"].(string); ok {
		phone = strings.TrimLeft(v, " +0")
		if len(phone) < 8 {
			response.Error(global.ErrInvalid, []string{"phone"})
			return
		}
	} else {
		if v, ok := request.Data["uid"].(string); ok {
			account := s.Worker().Model().Account.GetByID(v, nil)
			if account == nil {
				response.Error(global.ErrInvalid, []string{"uid"})
				return
			} else {
				phone = account.Phone
			}
		} else {
			response.Error(global.ErrIncomplete, []string{})
			return
		}
	}
	verification := s.Worker().Model().Verification.CreateByPhone(phone)

	adp := NewADP(
		config.GetString(config.ADPUsername),
		config.GetString(config.ADPPassword),
		config.GetString(config.ADPMessageUrl),
	)
	adp.SendSms(verification.Phone, "Nested verification code is: "+verification.ShortCode)
	response.OkWithData(tools.M{
		"vid":   verification.ID,
		"phone": fmt.Sprintf("%s******%s", string(phone[:3]), string(phone[len(phone)-2:])),
	})
	return
}

// @Command:	auth/get_email_verification
// @Input:	email	string	*
func (s *AuthService) getEmailVerificationCode(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var email string

	if v, ok := request.Data["email"].(string); ok {
		email = strings.Trim(v, " ")
	} else {
		if v, ok := request.Data["uid"].(string); ok {
			account := s.Worker().Model().Account.GetByID(v, nil)
			if account == nil {
				response.Error(global.ErrInvalid, []string{"uid"})
				return
			} else {
				email = account.Email
			}
		}
	}
	verification := s.Worker().Model().Verification.CreateByEmail(email)

	response.OkWithData(tools.M{
		"vid": verification.ID,
		// "email": fmt.Sprintf("%s******%s", string(phone[:3]), string(phone[len(phone) - 2:])),
	})
	return
}

// @Command:	auth/verify_code
// @Input:	code	string	*
// @Input:	vid 	string	*	"Verification ID"
func (s *AuthService) verifyCode(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var verifyID string
	var code string
	if v, ok := request.Data["vid"].(string); ok {
		verifyID = v
	} else {
		response.Error(global.ErrIncomplete, []string{"vid"})
		return
	}
	if v, ok := request.Data["code"].(string); ok {
		code = v
	}
	if s.Worker().Model().Verification.Verify(verifyID, code) {
		response.Ok()
	} else {
		response.Error(global.ErrInvalid, []string{"vid", "code"})
	}
	return
}

// @Command:	auth/send_text
// @Input:	vid		string	*
func (s *AuthService) sendCodeByText(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var verification *nested.Verification
	if v, ok := request.Data["vid"].(string); ok {
		verification = s.Worker().Model().Verification.GetByID(v)
		if verification == nil {
			response.Error(global.ErrInvalid, []string{"vid"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"vid"})
		return
	}
	if verification.Phone == nested.TestPhoneNumber {
		return
	}
	if verification.Counters.Sms > 3 {
		response.Error(global.ErrLimit, []string{"no_more_sms"})
		return
	}
	s.Worker().Model().Verification.IncrementSmsCounter(verification.ID)

	if strings.HasPrefix(verification.Phone, "98") {
		adp := NewADP(
			config.GetString(config.ADPUsername),
			config.GetString(config.ADPPassword),
			config.GetString(config.ADPMessageUrl),
		)
		adp.SendSms(verification.Phone, "Nested verification code is: "+verification.ShortCode)
	}
	response.Ok()
	return
}

// @Command:	auth/recover_pass
// @Input:	vid			string	*
// @Input:	new_pass	string	*
func (s *AuthService) recoverPassword(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var verification *nested.Verification
	var newPass string
	if v, ok := request.Data["vid"].(string); ok {
		verification = s.Worker().Model().Verification.GetByID(v)
		if verification == nil {
			response.Error(global.ErrInvalid, []string{"vid"})
			return
		}
		// check verification object is verified
		if !verification.Verified || verification.Expired {
			response.Error(global.ErrInvalid, []string{"vid"})
			return
		}
	}
	if v, ok := request.Data["new_pass"].(string); ok {
		// FIXME:: check password meet requirements
		newPass = v
	} else {
		response.Error(global.ErrInvalid, []string{"new_pass"})
		return
	}
	if verification.Phone == nested.TestPhoneNumber {
		response.OkWithData(tools.M{"text": "this is for test purpose"})
		return
	}
	account := s.Worker().Model().Account.GetByPhone(verification.Phone, nil)
	if account != nil {
		s.Worker().Model().Account.SetPassword(account.ID, newPass)
		response.Ok()
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
	s.Worker().Model().Verification.Expire(verification.ID)
}

// @Command:	auth/recover_username
// @Input:	vid		string	*
func (s *AuthService) recoverUsername(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var verification *nested.Verification
	if v, ok := request.Data["vid"].(string); ok {
		verification = s.Worker().Model().Verification.GetByID(v)
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
	if verification.Phone == nested.TestPhoneNumber {
		response.OkWithData(tools.M{
			"text": "this is for test purpose",
			"uid":  "_username",
		})
		return
	}
	account := s.Worker().Model().Account.GetByPhone(verification.Phone, nil)
	if account != nil {
		response.OkWithData(tools.M{"uid": account.ID})
	} else {
		response.Error(global.ErrUnknown, []string{})
	}
	s.Worker().Model().Verification.Expire(verification.ID)
}

// @Command:	auth/phone_available
// @Input:	phone		string	*
func (s *AuthService) phoneAvailable(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var phone string
	if v, ok := request.Data["phone"].(string); ok {
		phone = strings.TrimLeft(v, " +0")
		systemConstants := s.Worker().Model().System.GetStringConstants()
		if phone != systemConstants[global.SystemConstantsMagicNumber] && s.Worker().Model().Account.PhoneExists(phone) {
			response.Error(global.ErrDuplicate, []string{"phone"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"phone"})
		return
	}
	response.Ok()
	return

}

// @Command:	auth/register_user
// @Input:	uid			string	*
// @Input:	pass		string	*
// @Input:	fname		string	*
// @Input:	lname		string	*
// @Input:	gender		string	+
// @Input:	dob			string	+
// @Input:	country		string	+
// @Input:	email		string	+
// @Input:	phone		string	*
// @Input:	vid			string	*
func (s *AuthService) registerUserAccount(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var uid, pass, fname, lname, gender, dob, country, email, phone string
	var verification *nested.Verification

	if global.RegisterMode == global.RegisterModeAdminOnly {
		response.Error(global.ErrAccess, []string{"only_admin"})
		return
	}

	// Check License Limit
	counters := s.Worker().Model().System.GetCounters()
	maxActiveUsers := s.Worker().Model().License.Get().MaxActiveUsers
	if maxActiveUsers != 0 && counters[global.SystemCountersEnabledAccounts] >= maxActiveUsers {
		response.Error(global.ErrLimit, []string{"license_users_limit"})
		return
	}

	if v, ok := request.Data["uid"].(string); ok {
		uid = strings.ToLower(strings.Trim(v, " "))
	}
	if v, ok := request.Data["pass"].(string); ok {
		pass = v
	}
	if v, ok := request.Data["fname"].(string); ok {
		fname = strings.TrimSpace(v)
	}
	if v, ok := request.Data["lname"].(string); ok {
		lname = strings.TrimSpace(v)
	}
	if v, ok := request.Data["gender"].(string); ok {
		gender = v
	}
	if v, ok := request.Data["dob"].(string); ok {
		if _, err := time.Parse("2006-01-02", v); err == nil {
			dob = v
		}
	}
	if v, ok := request.Data["country"].(string); ok {
		country = v
	}
	if v, ok := request.Data["email"].(string); ok && len(v) > 0 {
		email = strings.ToLower(strings.Trim(v, " "))
		if !nested.IsValidEmail(email) {
			response.Error(global.ErrInvalid, []string{"email"})
			return
		}
	}
	if v, ok := request.Data["phone"].(string); ok {
		phone = v
		phone = strings.TrimLeft(phone, "+0")
	}
	if v, ok := request.Data["vid"].(string); ok {
		verification = s.Worker().Model().Verification.GetByID(v)
		if verification == nil {
			log.Println("RegisterUserAccount::Error::Invalid_VerificationID")
			log.Println("Arguments:", request.Data)
			response.Error(global.ErrInvalid, []string{"vid"})
			return
		}

		// check verification object is verified
		if !verification.Verified || verification.Phone != phone {
			response.Error(global.ErrInvalid, []string{"vid"})
			return
		}
		s.Worker().Model().Verification.Expire(verification.ID)
	}

	// check if username match the regular expression
	if matched, err := regexp.MatchString(global.DefaultRegexAccountID, uid); err != nil {
		response.Error(global.ErrUnknown, []string{err.Error()})
		return
	} else if !matched {
		response.Error(global.ErrInvalid, []string{"uid"})
		return
	}
	// check if username is not taken already
	if s.Worker().Model().Account.Exists(uid) || s.Worker().Model().Place.Exists(uid) {
		response.Error(global.ErrDuplicate, []string{"uid"})
		return
	}
	// check if phone is not taken already
	systemConstants := s.Worker().Model().System.GetStringConstants()
	if phone != systemConstants[global.SystemConstantsMagicNumber] && s.Worker().Model().Account.PhoneExists(phone) {
		response.Error(global.ErrDuplicate, []string{"phone"})
		return
	}
	// check if email is not taken already
	if email != "" && s.Worker().Model().Account.EmailExists(email) {
		response.Error(global.ErrDuplicate, []string{"email"})
		return
	}
	// check that fname and lname cannot both be empty text
	if fname == "" && lname == "" {
		response.Error(global.ErrInvalid, []string{"fname", "lname"})
		return
	}

	if verification.Phone == nested.TestPhoneNumber {
		response.OkWithData(tools.M{"info": "This user does not actually created. You are using test phone"})
		return
	}

	if !s.Worker().Model().Account.CreateUser(uid, pass, verification.Phone, country, fname, lname, email, dob, gender) {
		response.Error(global.ErrUnknown, []string{""})
		return
	}

	// create personal place for the new account
	pcr := nested.PlaceCreateRequest{
		ID:            uid,
		GrandParentID: uid,
		AccountID:     uid,
		Name:          fmt.Sprintf("%s %s", fname, lname),
		Description:   fmt.Sprintf("Personal place for %s", uid),
	}
	pcr.Policy.AddMember = nested.PlacePolicyNoOne
	pcr.Policy.AddPlace = nested.PlacePolicyCreators
	pcr.Policy.AddPost = nested.PlacePolicyEveryone
	pcr.Privacy.Locked = true
	pcr.Privacy.Receptive = nested.PlaceReceptiveExternal
	pcr.Privacy.Search = true
	s.Worker().Model().Place.CreatePersonalPlace(pcr)

	// add the new user to his/her new personal place
	s.Worker().Model().Place.AddKeyHolder(pcr.ID, pcr.AccountID)
	s.Worker().Model().Place.Promote(pcr.ID, pcr.AccountID)

	// add the personal place to his/her favorite place
	s.Worker().Model().Account.AddPlaceToBookmarks(pcr.AccountID, pcr.ID)

	// set notification on for place
	s.Worker().Model().Account.SetPlaceNotification(pcr.AccountID, pcr.ID, true)

	// add user's account & place to search index
	s.Worker().Model().Search.AddPlaceToSearchIndex(uid, fmt.Sprintf("%s %s", fname, lname), pcr.Picture)

	if placeIDs := s.Worker().Model().Place.GetDefaultPlaces(); len(placeIDs) > 0 {
		for _, placeID := range placeIDs {
			place := s.Worker().Model().Place.GetByID(placeID, nil)
			if place == nil {
				continue
			}
			grandPlace := place.GetGrandParent()
			// if user is already a member of the place then skip
			if place.IsMember(uid) {
				continue
			}
			// if user is not a keyHolder or Creator of place grandPlace, then make him to be
			if !grandPlace.IsMember(uid) {
				log.Println("grandPlace.IsMember(uid):: not member of grandPlcae")
				s.Worker().Model().Place.AddKeyHolder(grandPlace.ID, uid)
				// Enables notification by default
				s.Worker().Model().Account.SetPlaceNotification(uid, grandPlace.ID, true)

				// Add the place to the added user's feed list
				s.Worker().Model().Account.AddPlaceToBookmarks(uid, grandPlace.ID)

				// Handle push notifications and activities
				s.Worker().Pusher().PlaceJoined(grandPlace, uid, uid)
			}
			// if place is a grandPlace then skip going deeper
			if place.IsGrandPlace() {
				log.Println("place.IsGrandPlace()")
				continue
			}
			// if !place.HasKeyholderLimit() {
			s.Worker().Model().Place.AddKeyHolder(place.ID, uid)

			// Enables notification by default
			s.Worker().Model().Account.SetPlaceNotification(uid, place.ID, true)

			// Add the place to the added user's feed list
			s.Worker().Model().Account.AddPlaceToBookmarks(uid, place.ID)

			// Handle push notifications and activities
			s.Worker().Pusher().PlaceJoined(place, uid, uid)

			place.Counter.Keyholders += 1
		}
	}
	// prepare welcome message and invitations
	go s.prepareWelcome(uid)

	response.Ok()
	return
}

// @Command:	auth/authorize_app
// @Input:	app_id				string	*
// @Input:	app_name			string	*
// @Input:	app_homepage 		string	+
// @Input:	app_callback_url	string	+
func (s *AuthService) authorizeApp(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var appID, appName, appHomepage, appCallbackUrl string
	if v, ok := request.Data["app_id"].(string); ok {
		appID = v
	} else {
		response.Error(global.ErrIncomplete, []string{"app_id"})
		return
	}
	if v, ok := request.Data["app_name"].(string); ok {
		appName = v
	} else {
		response.Error(global.ErrIncomplete, []string{"app_name"})
		return
	}
	_, _, _, _ = appID, appName, appHomepage, appCallbackUrl
	// TODO:: Not Implemented

}

func (s *AuthService) prepareWelcome(accountID string) {
	account := s.Worker().Model().Account.GetByID(accountID, nil)
	var fillData struct {
		AccountFirstName string
		AccountLastName  string
	}
	fillData.AccountFirstName = account.FirstName
	fillData.AccountLastName = account.LastName

	msgTemplates := s.Worker().Model().System.GetMessageTemplates()
	var body bytes.Buffer
	t, _ := template.New("Welcome").Parse(msgTemplates["WELCOME_MSG"].Body)
	t.Execute(&body, fillData)

	pcr := nested.PostCreateRequest{
		SenderID:    "nested",
		Subject:     msgTemplates["WELCOME_MSG"].Subject,
		Body:        body.String(),
		ContentType: nested.ContentTypeTextHtml,
		PlaceIDs:    []string{accountID},
		SystemData: nested.PostSystemData{
			NoComment: true,
		},
	}

	s.Worker().Model().Post.AddPost(pcr)

}
