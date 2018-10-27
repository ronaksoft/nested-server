package nestedServiceAuth

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"regexp"
	"strings"
	"time"

	"git.ronaksoftware.com/nested/server/model"
	"git.ronaksoftware.com/nested/server/server-gateway/client"
)

// @Command:	auth/get_verification
// @Input:	phone	string	*
// @Input:	uid 	string	*
func (s *AuthService) getPhoneVerificationCode(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var phone string
	if v, ok := request.Data["phone"].(string); ok {
		phone = strings.TrimLeft(v, " +0")
		if len(phone) < 8 {
			response.Error(nested.ERR_INVALID, []string{"phone"})
			return
		}
	} else {
		if v, ok := request.Data["uid"].(string); ok {
			account := s.Worker().Model().Account.GetByID(v, nil)
			if account == nil {
				response.Error(nested.ERR_INVALID, []string{"uid"})
				return
			} else {
				phone = account.Phone
			}
		} else {
			response.Error(nested.ERR_INCOMPLETE, []string{})
			return
		}
	}
	verification := s.Worker().Model().Verification.CreateByPhone(phone)

	adp := NewADP(
		s.Worker().Config().GetString("ADP_USERNAME"),
		s.Worker().Config().GetString("ADP_PASSWORD"),
		s.Worker().Config().GetString("ADP_MESSAGE_URL"),
	)
	adp.SendSms(verification.Phone, "Nested verification code is: "+verification.ShortCode)
	response.OkWithData(nested.M{
		"vid":   verification.ID,
		"phone": fmt.Sprintf("%s******%s", string(phone[:3]), string(phone[len(phone)-2:])),
	})
	return
}

// @Command:	auth/get_email_verification
// @Input:	email	string	*
func (s *AuthService) getEmailVerificationCode(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var email string

	if v, ok := request.Data["email"].(string); ok {
		email = strings.Trim(v, " ")
	} else {
		if v, ok := request.Data["uid"].(string); ok {
			account := s.Worker().Model().Account.GetByID(v, nil)
			if account == nil {
				response.Error(nested.ERR_INVALID, []string{"uid"})
				return
			} else {
				email = account.Email
			}
		}
	}
	verification := s.Worker().Model().Verification.CreateByEmail(email)

	response.OkWithData(nested.M{
		"vid": verification.ID,
		//"email": fmt.Sprintf("%s******%s", string(phone[:3]), string(phone[len(phone) - 2:])),
	})
	return
}

// @Command:	auth/verify_code
// @Input:	code	string	*
// @Input:	vid 	string	*	"Verification ID"
func (s *AuthService) verifyCode(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var verifyID string
	var code string
	if v, ok := request.Data["vid"].(string); ok {
		verifyID = v
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"vid"})
		return
	}
	if v, ok := request.Data["code"].(string); ok {
		code = v
	}
	if s.Worker().Model().Verification.Verify(verifyID, code) {
		response.Ok()
	} else {
		response.Error(nested.ERR_INVALID, []string{"vid", "code"})
	}
	return
}

// @Command:	auth/send_text
// @Input:	vid		string	*
func (s *AuthService) sendCodeByText(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var verification *nested.Verification
	if v, ok := request.Data["vid"].(string); ok {
		verification = s.Worker().Model().Verification.GetByID(v)
		if verification == nil {
			response.Error(nested.ERR_INVALID, []string{"vid"})
			return
		}
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"vid"})
		return
	}
	if verification.Phone == nested.TEST_PHONE_NUMBER {
		return
	}
	if verification.Counters.Sms > 3 {
		response.Error(nested.ERR_LIMIT, []string{"no_more_sms"})
		return
	}
	s.Worker().Model().Verification.IncrementSmsCounter(verification.ID)

	if strings.HasPrefix(verification.Phone, "98") {
		adp := NewADP(
			s.Worker().Config().GetString("ADP_USERNAME"),
			s.Worker().Config().GetString("ADP_PASSWORD"),
			s.Worker().Config().GetString("ADP_MESSAGE_URL"),
		)
		adp.SendSms(verification.Phone, "Nested verification code is: "+verification.ShortCode)
	}
	response.Ok()
	return
}

// @Command:	auth/recover_pass
// @Input:	vid			string	*
// @Input:	new_pass	string	*
func (s *AuthService) recoverPassword(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var verification *nested.Verification
	var newPass string
	if v, ok := request.Data["vid"].(string); ok {
		verification = s.Worker().Model().Verification.GetByID(v)
		if verification == nil {
			response.Error(nested.ERR_INVALID, []string{"vid"})
			return
		}
		// check verification object is verified
		if !verification.Verified || verification.Expired {
			response.Error(nested.ERR_INVALID, []string{"vid"})
			return
		}
	}
	if v, ok := request.Data["new_pass"].(string); ok {
		//FIXME:: check password meet requirements
		newPass = v
	} else {
		response.Error(nested.ERR_INVALID, []string{"new_pass"})
		return
	}
	if verification.Phone == nested.TEST_PHONE_NUMBER {
		response.OkWithData(nested.M{"text": "this is for test purpose"})
		return
	}
	account := s.Worker().Model().Account.GetByPhone(verification.Phone, nil)
	if account != nil {
		s.Worker().Model().Account.SetPassword(account.ID, newPass)
		response.Ok()
	} else {
		response.Error(nested.ERR_UNKNOWN, []string{})
	}
	s.Worker().Model().Verification.Expire(verification.ID)
}

// @Command:	auth/recover_username
// @Input:	vid		string	*
func (s *AuthService) recoverUsername(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var verification *nested.Verification
	if v, ok := request.Data["vid"].(string); ok {
		verification = s.Worker().Model().Verification.GetByID(v)
		if verification == nil {
			response.Error(nested.ERR_INVALID, []string{"vid"})
			return
		}
		// check verification object is verified
		if !verification.Verified || verification.Expired {
			response.Error(nested.ERR_INVALID, []string{"vid"})
			return
		}
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"vid"})
		return
	}
	if verification.Phone == nested.TEST_PHONE_NUMBER {
		response.OkWithData(nested.M{
			"text": "this is for test purpose",
			"uid":  "_username",
		})
		return
	}
	account := s.Worker().Model().Account.GetByPhone(verification.Phone, nil)
	if account != nil {
		response.OkWithData(nested.M{"uid": account.ID})
	} else {
		response.Error(nested.ERR_UNKNOWN, []string{})
	}
	s.Worker().Model().Verification.Expire(verification.ID)
}

// @Command:	auth/phone_available
// @Input:	phone		string	*
func (s *AuthService) phoneAvailable(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var phone string
	if v, ok := request.Data["phone"].(string); ok {
		phone = strings.TrimLeft(v, " +0")
		systemConstants := s.Worker().Model().System.GetStringConstants()
		if phone != systemConstants[nested.SYSTEM_CONSTANTS_MAGIC_NUMBER] && s.Worker().Model().Account.PhoneExists(phone) {
			response.Error(nested.ERR_DUPLICATE, []string{"phone"})
			return
		}
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"phone"})
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
func (s *AuthService) registerUserAccount(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var uid, pass, fname, lname, gender, dob, country, email, phone string
	var verification *nested.Verification

	if nested.REGISTER_MODE == nested.REGISTER_MODE_ADMIN_ONLY {
		response.Error(nested.ERR_ACCESS, []string{"only_admin"})
		return
	}

	// Check License Limit
	counters := s.Worker().Model().System.GetCounters()
	maxActiveUsers := s.Worker().Model().License.Get().MaxActiveUsers
	if maxActiveUsers != 0 && counters[nested.SYSTEM_COUNTERS_ENABLED_ACCOUNTS] >= maxActiveUsers {
		response.Error(nested.ERR_LIMIT, []string{"license_users_limit"})
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
			response.Error(nested.ERR_INVALID, []string{"email"})
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
			response.Error(nested.ERR_INVALID, []string{"vid"})
			return
		}

		// check verification object is verified
		if !verification.Verified || verification.Phone != phone {
			response.Error(nested.ERR_INVALID, []string{"vid"})
			return
		}
		s.Worker().Model().Verification.Expire(verification.ID)
	}

	// check if username match the regular expression
	if matched, err := regexp.MatchString(nested.DEFAULT_REGEX_ACCOUNT_ID, uid); err != nil {
		response.Error(nested.ERR_UNKNOWN, []string{err.Error()})
		return
	} else if !matched {
		response.Error(nested.ERR_INVALID, []string{"uid"})
		return
	}
	// check if username is not taken already
	if s.Worker().Model().Account.Exists(uid) || s.Worker().Model().Place.Exists(uid) {
		response.Error(nested.ERR_DUPLICATE, []string{"uid"})
		return
	}
	// check if phone is not taken already
	systemConstants := s.Worker().Model().System.GetStringConstants()
	if phone != systemConstants[nested.SYSTEM_CONSTANTS_MAGIC_NUMBER] && s.Worker().Model().Account.PhoneExists(phone) {
		response.Error(nested.ERR_DUPLICATE, []string{"phone"})
		return
	}
	// check if email is not taken already
	if email != "" && s.Worker().Model().Account.EmailExists(email) {
		response.Error(nested.ERR_DUPLICATE, []string{"email"})
		return
	}
	// check that fname and lname cannot both be empty text
	if fname == "" && lname == "" {
		response.Error(nested.ERR_INVALID, []string{"fname", "lname"})
		return
	}

	if verification.Phone == nested.TEST_PHONE_NUMBER {
		response.OkWithData(nested.M{"info": "This user does not actually created. You are using test phone"})
		return
	}

	if !s.Worker().Model().Account.CreateUser(uid, pass, verification.Phone, country, fname, lname, email, dob, gender) {
		response.Error(nested.ERR_UNKNOWN, []string{""})
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
	pcr.Policy.AddMember = nested.PLACE_POLICY_NOONE
	pcr.Policy.AddPlace = nested.PLACE_POLICY_CREATORS
	pcr.Policy.AddPost = nested.PLACE_POLICY_EVERYONE
	pcr.Privacy.Locked = true
	pcr.Privacy.Receptive = nested.PLACE_RECEPTIVE_EXTERNAL
	pcr.Privacy.Search = true
	s.Worker().Model().Place.CreatePersonalPlace(pcr)

	// add the new user to his/her new personal place
	s.Worker().Model().Place.AddKeyholder(pcr.ID, pcr.AccountID)
	s.Worker().Model().Place.Promote(pcr.ID, pcr.AccountID)

	// add the personal place to his/her favorite place
	s.Worker().Model().Account.AddPlaceToBookmarks(pcr.AccountID, pcr.ID)

	// set notification on for place
	s.Worker().Model().Account.SetPlaceNotification(pcr.AccountID, pcr.ID, true)

	// add user's account & place to search index
	s.Worker().Model().Search.AddPlaceToSearchIndex(uid, fmt.Sprintf("%s %s", fname, lname))

	if placeIDs := s.Worker().Model().Place.GetDefaultPlaces(); len(placeIDs) > 0 {
		log.Println("placeIds", placeIDs, "userID", uid)
		for _, placeID := range placeIDs {
			place :=  s.Worker().Model().Place.GetByID(placeID, nil)
			grandPlace := place.GetGrandParent()
			log.Println("place", place.ID, "grandPlace",grandPlace.ID)
			// if user is already a member of the place then skip
			if place.IsMember(uid) {
				log.Println("place.IsMember")
				continue
			}
			// if user is not a keyHolder or Creator of place grandPlace, then make him to be
			if !grandPlace.IsMember(uid) {
				log.Println("grandPlace.IsMember(uid):: not member of grandPlcae")
				if !grandPlace.HasKeyholderLimit() {
					log.Println("grandplace has capacity")
					s.Worker().Model().Place.AddKeyholder(grandPlace.ID, uid)
					// Enables notification by default
					s.Worker().Model().Account.SetPlaceNotification(uid, grandPlace.ID, true)

					// Add the place to the added user's feed list
					s.Worker().Model().Account.AddPlaceToBookmarks(uid, grandPlace.ID)


					// Handle push notifications and activities
					log.Println("PlaceJoined", grandPlace.ID, requester.ID, uid)
					s.Worker().Pusher().PlaceJoined(grandPlace, requester.ID, uid)
				} else {
					response.Error(nested.ERR_INVALID, []string{"grandplace_keyholder_limit"})
					return
				}
				// if place is a grandPlace then skip going deeper
				if place.IsGrandPlace() {
					log.Println("place.IsGrandPlace()")
					continue
				}
				if !place.HasKeyholderLimit() {
					s.Worker().Model().Place.AddKeyholder(place.ID, uid)

					// Enables notification by default
					s.Worker().Model().Account.SetPlaceNotification(uid, place.ID, true)

					// Add the place to the added user's feed list
					s.Worker().Model().Account.AddPlaceToBookmarks(uid, place.ID)

					// Handle push notifications and activities
					s.Worker().Pusher().PlaceJoined(place, requester.ID, uid)

					place.Counter.Keyholders += 1
				} else {
					response.Error(nested.ERR_INVALID, []string{"place_keyholder_limit"})
					return
				}
			}
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
func (s *AuthService) authorizeApp(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
	var appID, appName, appHomepage, appCallbackUrl string
	if v, ok := request.Data["app_id"].(string); ok {
		appID = v
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"app_id"})
		return
	}
	if v, ok := request.Data["app_name"].(string); ok {
		appName = v
	} else {
		response.Error(nested.ERR_INCOMPLETE, []string{"app_name"})
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
		ContentType: nested.CONTENT_TYPE_TEXT_HTML,
		PlaceIDs:    []string{accountID},
		SystemData: nested.PostSystemData{
			NoComment: true,
		},
	}

	s.Worker().Model().Post.AddPost(pcr)

}
