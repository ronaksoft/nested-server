package nested

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"golang.org/x/crypto/bcrypt"
	"log"
	"regexp"
	"strings"
	"github.com/gomodule/redigo/redis"
)

const (
	ACCOUNT_TYPE_USER     string = "USER"
	ACCOUNT_TYPE_DEVICE   string = "DEVICE"
	ACCOUNT_TYPE_EXTERNAL string = "EXTERNAL"
	ACCOUNT_TYPE_EMAIL    string = "EMAIL"
)

type AccountUpdateRequest struct {
	FirstName           string `json:"fname" bson:"fname"`
	LastName            string `json:"lname" bson:"lname"`
	Email               string `json:"email" bson:"email"`
	Phone               string `json:"phone" bson:"phone"`
	Gender              string `json:"gender" bson:"gender"`
	DateOfBirth         string `json:"dob" bson:"dob"`
	NumberOfGrandPlaces int    `json:"limits.grand_places" bson:"limits.grand_places"`
}
type Account struct {
	ID                 string           `json:"_id" bson:"_id"`
	Secret             string           `json:"secret" bson:"secret"`
	AuthKey            string           `json:"auth_key" bson:"auth_key"`
	Type               string           `json:"acc_type" bson:"acc_type"`
	Disabled           bool             `json:"disabled" bson:"disabled"`
	FirstName          string           `json:"fname" bson:"fname"`
	LastName           string           `json:"lname" bson:"lname"`
	FullName           string           `json:"full_name" bson:"full_name"`
	Picture            Picture          `json:"picture" bson:"picture"`
	Email              string           `json:"email" bson:"email,omitempty"`
	Phone              string           `json:"phone" bson:"phone,omitempty"`
	Gender             string           `json:"gender" bson:"gender"`
	Country            string           `json:"country" bson:"country"`
	AccessPlaceIDs     []string         `json:"access_places" bson:"access_places"`
	BookmarkedPlaceIDs []string         `json:"bookmarked_places" bson:"bookmarked_places"`
	DateOfBirth        string           `json:"dob" bson:"dob"`
	Authority          AccountAuthority `json:"authority" bson:"authority"`
	Counters           AccountCounters  `json:"counters" bson:"counters"`
	Limits             AccountLimits    `json:"limits" bson:"limits"`
	Privacy            AccountPrivacy   `json:"privacy" bson:"privacy"`
	Flags              AccountFlags     `json:"flags" bson:"flags"`
	Mail               AccountMail      `json:"mail"`
	JoinedOn           uint64           `json:"joined_on" bson:"joined_on"`
}
type AccountCounters struct {
	TotalNotifications  int `json:"total_notifications" bson:"total_notifications"`
	UnreadNotifications int `json:"unread_notifications" bson:"unread_notifications"`
	IncorrectAttempts   int `json:"incorrect_attempts" bson:"incorrect_attempts"`
	Logins              int `json:"logins" bson:"logins"`
	Keys                int `json:"client_keys" bson:"client_keys"`
}
type AccountLimits struct {
	GrandPlaces int `json:"grand_places" bson:"grand_places"`
	Keys        int `json:"client_keys" bson:"client_keys"`
}
type AccountFlags struct {
	ForcePasswordChange bool `json:"force_password_change" bson:"force_password_change"`
}
type AccountPrivacy struct {
	Searchable    bool `json:"searchable" bson:"searchable"`
	ChangePicture bool `json:"change_picture" bson:"change_picture"`
	ChangeProfile bool `json:"change_profile" bson:"change_profile"`
}
type AccountAuthority struct {
	LabelEditor bool `json:"label_editor" bson:"label_editor"`
	Admin       bool `json:"admin" bson:"admin"`
}
type AccountMail struct {
	Active           bool   `json:"active" bson:"active"`
	OutgoingSMTPHost string `json:"outgoing_smtp_host" bson:"outgoing_smtp_host"`
	OutgoingSMTPPort int    `json:"outgoing_smtp_port" bson:"outgoing_smtp_port"`
	OutgoingSMTPUser string `json:"outgoing_smtp_user" bson:"outgoing_smtp_user"`
	OutgoingSMTPPass string `json:"outgoing_smtp_pass" bson:"outgoing_smtp_pass"`
}

// Account Manager and Methods
type AccountManager struct{}

func NewAccountManager() *AccountManager { return new(AccountManager) }

func (am *AccountManager) readFromCache(accountID string) *Account {
	account := new(Account)
	c := _Cache.Pool.Get()
	defer c.Close()
	keyID := fmt.Sprintf("account:gob:%s", accountID)
	if gobAccount, err := redis.Bytes(c.Do("GET", keyID)); err != nil {
		if err := _MongoDB.C(COLLECTION_ACCOUNTS).FindId(accountID).One(account); err != nil {
			log.Println("Model::AccountManager::readFromCache::Error 1::", err.Error(), accountID)
			return nil
		}
		gobAccount := new(bytes.Buffer)
		if err := gob.NewEncoder(gobAccount).Encode(account); err == nil {
			c.Do("SETEX", keyID, CACHE_LIFETIME, gobAccount.Bytes())
		}
		return account
	} else if err := gob.NewDecoder(bytes.NewBuffer(gobAccount)).Decode(account); err == nil {
		return account
	}
	return nil
}

func (am *AccountManager) readMultiFromCache(accountIDs []string) []Account {
	accounts := make([]Account, 0, len(accountIDs))
	c := _Cache.Pool.Get()
	defer c.Close()
	for _, accountID := range accountIDs {
		keyID := fmt.Sprintf("account:gob:%s", accountID)
		c.Send("GET", keyID)
	}
	c.Flush()
	for _, accountID := range accountIDs {
		if gobAccount, err := redis.Bytes(c.Receive()); err == nil {
			account := new(Account)
			if err := gob.NewDecoder(bytes.NewBuffer(gobAccount)).Decode(account); err == nil {
				accounts = append(accounts, *account)
			}
		} else {
			if account := _Manager.Account.readFromCache(accountID); account != nil {
				accounts = append(accounts, *account)
			}
		}
	}
	return accounts
}

func (am *AccountManager) readKeyFromCache(keyID string) string {
	doc := M{}
	c := _Cache.Pool.Get()
	defer c.Close()
	if keyValue, err := redis.String(c.Do("GET", fmt.Sprintf("account-key:json:%s", keyID))); err != nil {
		if err := _MongoDB.C(COLLECTION_ACCOUNTS_DATA).FindId(keyID).One(&doc); err != nil {
			log.Println("Model::AccountManager::readKeyFromCache::Error 1::", err.Error())
			return ""
		}
		c.Do("SETEX", keyID, CACHE_LIFETIME, doc["value"].(string))
		return doc["value"].(string)
	} else {
		return keyValue
	}
	return ""
}

func (am *AccountManager) removeCache(accountID string) bool {
	c := _Cache.Pool.Get()
	defer c.Close()
	keyID := fmt.Sprintf("account:gob:%s", accountID)
	c.Do("DEL", keyID)
	return true
}

func (am *AccountManager) removeKeyCache(keyID string) bool {
	c := _Cache.Pool.Get()
	defer c.Close()
	c.Do("DEL", fmt.Sprintf("account-key:json:%s", keyID))
	return true
}

func (am *AccountManager) removeMultiFromCache(accountIDs []string) bool {
	c := _Cache.Pool.Get()
	defer c.Close()
	for _, accountID := range accountIDs {
		keyID := fmt.Sprintf("account:json:%s", accountID)
		c.Send("DEL", keyID)
	}
	c.Flush()
	return true
}

// Description:
// Adds the place to the bookmarked list
func (am *AccountManager) AddPlaceToBookmarks(accountID, placeID string) {
	_Log.FunctionStarted("AccountManager::AddPlaceToBookmarks", accountID, placeID)
	defer _Log.FunctionFinished("AccountManager::AddPlaceToBookmarks")
	defer _Manager.Account.removeCache(accountID)

	if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{
			"$addToSet": bson.M{
				"bookmarked_places": bson.M{
					"$each":  []string{placeID},
					"$slice": -DEFAULT_MAX_BOOKMARKED_PLACES,
				},
			},
		},
	); err != nil {
		log.Println("Model::AccountManager::AddPlaceToBookmarks::Error 1::", err.Error(), accountID, placeID)
	}
	return
}

// Available returns true if account can be created on system otherwise returns false
func (am *AccountManager) Available(accountID string) bool {
	_Log.FunctionStarted("AccountManager::Available", accountID)
	defer _Log.FunctionFinished("AccountManager::Available")
	if matched, err := regexp.MatchString(DEFAULT_REGEX_ACCOUNT_ID, accountID); err != nil {
		return false
	} else if !matched {
		return false
	}
	n, _ := _MongoDB.C(COLLECTION_ACCOUNTS).FindId(accountID).Count()
	if n > 0 {
		return false
	}
	n, _ = _MongoDB.C(COLLECTION_SYS_RESERVED_WORDS).Find(bson.M{"word": accountID}).Count()
	if n > 0 {
		return false
	}

	return true
}

// ClearRecentlyVisited clears the recently visited list of the accountID
func (am *AccountManager) ClearRecentlyVisited(accountID string) {
	_Log.FunctionStarted("AccountManager::ClearRecentlyVisited", accountID)
	defer _Log.FunctionFinished("AccountManager::ClearRecentlyVisited")
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{"$set": bson.M{"recently_visited": []string{}}},
	); err != nil {
		log.Println("Model::AccountManager::ClearRecentlyVisited::Error 1::", err.Error())
	}
	return
}

// CreateUser initially nested-tools-cli user, but the created user is disabled until CompleteUserRegister is called.
// It return TRUE if everything was going through with no problem otherwise return false
func (am *AccountManager) CreateUser(uid, pass, phone, country, fname, lname, email, dob, gender string) bool {
	_Log.FunctionStarted("AccountManager::CreateUser", uid, phone, country, fname, lname, email, dob, gender)
	defer _Log.FunctionFinished("AccountManager::CreateUser")

	y, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.MinCost)
	acc := Account{
		ID:       strings.ToLower(uid),
		Secret:   string(y),
		Disabled: true,
		Type:     ACCOUNT_TYPE_USER,
		Phone:    phone,
		Country:  country,
		JoinedOn: Timestamp(),
		AuthKey:  RandomID(32),
	}

	// Set Default Privacy Settings
	acc.Privacy.Searchable = true
	acc.Privacy.ChangePicture = true
	acc.Privacy.ChangeProfile = true

	acc.Limits.GrandPlaces = DEFAULT_ACCOUNT_GRAND_PLACES
	acc.Limits.Keys = DEFAULT_MAX_CLIENT_OBJ_COUNT
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).Insert(acc); err != nil {
		log.Println("Model::CreateUser::Error 1::", err.Error())
		return false
	}
	// Register account_id to phone
	_Manager.Phone.RegisterPhoneToAccount(acc.ID, acc.Phone)

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

	if err := _MongoDB.C(COLLECTION_ACCOUNTS).Update(
		bson.M{
			"_id":      uid,
			"acc_type": ACCOUNT_TYPE_USER,
		},
		bson.M{
			"$set": bson.M{
				"disabled":  false,
				"fname":     fname,
				"lname":     lname,
				"full_name": fmt.Sprintf("%s %s", fname, lname),
				"email":     email,
				"gender":    gender,
				"dob":       dob,
			},
		},
	); err != nil {
		log.Println("Model::AccountManager::CreateUser::Error 2::", err.Error())
		return false
	}
	// Update System.Internal Counter
	_Manager.System.incrementCounter(MI{SYSTEM_COUNTERS_ENABLED_ACCOUNTS: 1})

	return true
}

// Disable disables the account. Disabled accounts cannot login to the systemm
func (am *AccountManager) Disable(accountID string) bool {
	_Log.FunctionStarted("AccountManager::Disable", accountID)
	defer _Log.FunctionFinished("AccountManager::Disable")
	defer _Manager.Account.removeCache(accountID)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).Update(
		bson.M{"_id": accountID, "disabled": false},
		bson.M{"$set": bson.M{"disabled": true}},
	); err != nil {
		log.Println("Model::AccountManager::Disable::Error::", err.Error())
		return false
	}
	// Update System.Internal Counter
	_Manager.System.incrementCounter(MI{
		SYSTEM_COUNTERS_DISABLED_ACCOUNTS: 1,
		SYSTEM_COUNTERS_ENABLED_ACCOUNTS:  -1,
	})
	return true
}

// EmailExists checks if email already exists
func (am *AccountManager) EmailExists(email string) bool {
	_Log.FunctionStarted("AccountManager::EmailExists", email)
	defer _Log.FunctionFinished("AccountManager::EmailExists")
	n, _ := _MongoDB.C(COLLECTION_ACCOUNTS).Find(bson.M{"email": email}).Count()

	return n > 0
}

// Enables make the accountID enabled
func (am *AccountManager) Enable(accountID string) bool {
	_Log.FunctionStarted("AccountManager::Enable", accountID)
	defer _Log.FunctionFinished("AccountManager::Enable")
	defer _Manager.Account.removeCache(accountID)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).Update(
		bson.M{"_id": accountID, "disabled": true},
		bson.M{"$set": bson.M{
			"disabled":                    false,
			"counters.incorrect_attempts": 0,
		}},
	); err != nil {
		log.Println("Model::AccountManager::Disable::Error::", err.Error())
		return false
	}
	// Update System.Internal Counter
	_Manager.System.incrementCounter(MI{
		SYSTEM_COUNTERS_DISABLED_ACCOUNTS: -1,
		SYSTEM_COUNTERS_ENABLED_ACCOUNTS:  1,
	})
	return true
}

// Exists returns true if account exists otherwise false
// This function just check if the account id has been already created. it returns true
// even if the account is disabled or not completely registered.
func (am *AccountManager) Exists(accountID string) bool {
	_Log.FunctionStarted("AccountManager::Exists", accountID)
	defer _Log.FunctionFinished("AccountManager::Exists")
	n, _ := _MongoDB.C(COLLECTION_ACCOUNTS).FindId(accountID).Count()

	return n > 0
}

func (am *AccountManager) ForcePasswordChange(accountID string, state bool) bool {
	_Log.FunctionStarted("AccountManager::ForcePasswordChange", accountID, state)
	defer _Log.FunctionFinished("AccountManager::ForcePasswordChange")
	defer _Manager.Account.removeCache(accountID)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{"$set": bson.M{"flags.force_password_change": state}},
	); err != nil {
		log.Println("Model::AccountManager::SetPhone::Error 1::", err.Error())
		return false
	}
	return true
}

// GetAccessPlaceIDs returns an array of the place ids which the user has access to
func (am *AccountManager) GetAccessPlaceIDs(accountID string) []string {
	_Log.FunctionStarted("AccountManager::GetAccessPlaceIDs", accountID)
	defer _Log.FunctionFinished("AccountManager::GetAccessPlaceIDs")

	acc := _Manager.Account.GetByID(accountID, nil)
	if acc != nil {
		return acc.AccessPlaceIDs
	}
	return []string{}
}

// GetAccountsByIDs returns an array of accounts identified by accountIDs, it returns an empty slice if nothing was found
func (am *AccountManager) GetAccountsByIDs(accountIDs []string) []Account {
	_Log.FunctionStarted("AccountManager::GetAccountsByIDs", accountIDs)
	defer _Log.FunctionFinished("AccountManager::GetAccountsByIDs")
	return _Manager.Account.readMultiFromCache(accountIDs)
}

// GetByID returns the account by giving the ID of the account
func (am *AccountManager) GetByID(accountID string, pj M) *Account {
	_Log.FunctionStarted("AccountManager::GetByID", accountID)
	defer _Log.FunctionFinished("AccountManager::GetByID")
	return _Manager.Account.readFromCache(accountID)
}

// GetAccountByLoginToken returns account by giving a login token
func (am *AccountManager) GetAccountByLoginToken(token string) *Account {
	_funcName := "AccountManager::GetAccountByLoginToken"
	_Log.FunctionStarted(_funcName, token)
	defer _Log.FunctionFinished(_funcName)

	loginToken := _Manager.Token.GetLoginToken(token)
	if loginToken != nil {
		account := _Manager.Account.GetByID(loginToken.AccountID, nil)
		return account
	}
	return nil
}

// Return the account by giving the phone number of the account
func (am *AccountManager) GetByPhone(phone string, pj M) *Account {
	_funcName := "AccountManager::GetByPhone"
	_Log.FunctionStarted(_funcName, phone, pj)
	defer _Log.FunctionFinished(_funcName)

	acc := new(Account)
	if pj == nil {
		pj = M{
			"access_places":     0,
			"bookmarked_places": 0,
		}
	}
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).Find(bson.M{"phone": phone}).Select(pj).One(acc); err != nil {
		log.Println("Model::AccountManager::GetByPhone::Error", err.Error())
		return nil
	}

	return acc
}

// GetByEmail returns the account by giving the email address of the account
func (am *AccountManager) GetByEmail(email string, pj M) *Account {
	_funcName := "AccountManager::GetByEmail"
	_Log.FunctionStarted(_funcName, email)
	defer _Log.FunctionFinished(_funcName)

	acc := new(Account)
	if pj == nil {
		pj = M{
			"access_places":     0,
			"bookmarked_places": 0,
		}
	}
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).Find(bson.M{"email": email}).Select(pj).One(acc); err != nil {
		return nil
	}

	return acc
}

// GetBookmarkedPlaceIDs return an array of the place ids which the user has marked them as favorite
func (am *AccountManager) GetBookmarkedPlaceIDs(accountID string) []string {
	_funcName := "AccountManager::GetBookmarkedPlaceIDs"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)

	acc := _Manager.Account.GetByID(accountID, nil)
	if acc != nil {
		return acc.BookmarkedPlaceIDs
	}
	return []string{}
}

// GetKey get the value of the keyName for accountID
func (am *AccountManager) GetKey(accountID, keyName string) string {
	_funcName := "AccountManager::GetKey"
	_Log.FunctionStarted(_funcName, accountID, keyName)
	defer _Log.FunctionFinished(_funcName)
	keyID := fmt.Sprintf("%s.%s", accountID, keyName)
	keyValue := am.readKeyFromCache(keyID)
	return keyValue
}

// GetAllKeys returns a map of [keyName, keyValue] for accountID
func (am *AccountManager) GetAllKeys(accountID string) []MS {
	_funcName := "AccountManager::GetAllKeys"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)

	docs := []MS{}
	_MongoDB.C(COLLECTION_ACCOUNTS_DATA).Find(bson.M{
		"_id": bson.M{
			"$regex":   fmt.Sprintf("^%s\\..*", accountID),
			"$options": "i",
		},
	}).All(&docs)
	return docs
}

// GetMutualPlaceIDs returns an array of placeIDs which both accounts are member of
func (am *AccountManager) GetMutualPlaceIDs(accountID1, accountID2 string) []string {
	_funcName := "AccountManager::GetMutualPlaceIDs"
	_Log.FunctionStarted(_funcName, accountID1, accountID2)
	defer _Log.FunctionFinished(_funcName)

	placeIDs1 := _Manager.Account.GetAccessPlaceIDs(accountID1)
	placeIDs2 := _Manager.Account.GetAccessPlaceIDs(accountID2)
	if len(placeIDs2) < len(placeIDs1) {
		placeIDs1, placeIDs2 = placeIDs2, placeIDs1
	}
	mutualPlaceIDs := make(map[string]bool, len(placeIDs1))
	for _, placeID := range placeIDs1 {
		mutualPlaceIDs[placeID] = false
	}
	counter := 0
	for _, placeID := range placeIDs2 {
		if _, ok := mutualPlaceIDs[placeID]; ok {
			mutualPlaceIDs[placeID] = true
			counter++
		}
	}
	placeIDs := make([]string, 0, counter)
	for placeID, v := range mutualPlaceIDs {
		if v {
			placeIDs = append(placeIDs, placeID)
		}
	}
	return placeIDs
}

// IncreaseLogins increases the login counter for user "accountID"
func (am *AccountManager) IncreaseLogins(accountID string) {
	_funcName := "AccountManager::IncreaseLogins"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	_MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{"$inc": bson.M{"counters.logins": 1}},
	)
	return
}

// IncrementLimit Increase or Decrease Limit identified by limitKey
// Supported Limit Keys: grand_places
func (am *AccountManager) IncrementLimit(accountID, limitKey string, n int) bool {
	_funcName := "AccountManager::IncrementLimit"
	_Log.FunctionStarted(_funcName, accountID, limitKey, n)
	defer _Log.FunctionFinished(_funcName)

	switch limitKey {
	case "grand_places":
		_MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
			accountID,
			bson.M{"$inc": bson.M{"limits.grand_places": n}},
		)
	default:
		return false
	}
	return true

}

// IsEnabled checks if account is registered and also not disabled
// This function must be used if you want to make sure the account exists and is active
func (am *AccountManager) IsEnabled(accountID string) bool {
	_funcName := "AccountManager::IsEnabled"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)
	n, _ := _MongoDB.C(COLLECTION_ACCOUNTS).Find(bson.M{"disabled": false}).Count()
	return n > 0
}

// PhoneExists checks if phone already exists
func (am *AccountManager) PhoneExists(phone string) bool {
	_funcName := "AccountManager::PhoneExists"
	_Log.FunctionStarted(_funcName, phone)
	defer _Log.FunctionFinished(_funcName)

	n, _ := _MongoDB.C(COLLECTION_ACCOUNTS).Find(bson.M{"phone": phone}).Count()

	return n > 0
}

// RemovePlaceConnection removes 'Account <--> Place' relation points
// Then placeIDs will not be searched in SEARCH::PLACES_FOR_COMPOSE
func (am *AccountManager) RemovePlaceConnection(accountID string, placeIDs []string) bool {
	_funcName := "AccountManager::RemovePlaceConnection"
	_Log.FunctionStarted(_funcName, accountID, placeIDs)
	defer _Log.FunctionFinished(_funcName)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS_PLACES).Remove(bson.M{
		"account_id": accountID,
		"place_id":   bson.M{"$in": placeIDs},
	}); err != nil {
		log.Println("Model::AccountManager::RemovePlaceConnection::Error 1::", err.Error())
		return false
	}
	return true
}

// RemovePlaceFromBookmarks removes the place identified by "placeID" from the bookmarked list of the "accountID"
func (am *AccountManager) RemovePlaceFromBookmarks(accountID, placeID string) {
	_funcName := "AccountManager::RemovePlaceFromBookmarks"
	_Log.FunctionStarted(_funcName, accountID, placeID)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{"$pull": bson.M{"bookmarked_places": placeID}},
	); err != nil {
		log.Println("Model::AccountManager::RemovePlaceFromBookmarks::Error 1::", err.Error())
	}
	return
}

// RemoveRecipientConnection removes 'Account <--> Email Address' relation points
func (am *AccountManager) RemoveRecipientConnection(accountID string, recipients []string) {
	_funcName := "AccountManager::RemoveRecipientConnection"
	_Log.FunctionStarted(_funcName, accountID, recipients)
	defer _Log.FunctionFinished(_funcName)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS_RECIPIENTS).Remove(bson.M{
		"account_id": accountID,
		"recipient":  bson.M{"$in": recipients},
	}); err != nil {
		log.Println("Model::AccountManager::RemoveRecipientConnection::Error 1::", err.Error())
	}

}

// ResetLoginAttempts reset the login attempts
func (am *AccountManager) ResetLoginAttempts(accountID string) {
	_funcName := "AccountManager::ResetLoginAttempts"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	_MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{"$set": bson.M{"counters.incorrect_attempts": 0}},
	)
}

// ResetUnreadNotificationCounter reset the notification counter for user "accountID"
func (am *AccountManager) ResetUnreadNotificationCounter(accountID string) {
	_funcName := "AccountManager::ResetUnreadNotificationCounter"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	_MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{"$set": bson.M{"counters.unread_notifications": 0}},
	)
	return
}

// RemoveKey removes the key from database, if the keyName existed then it return true, otherwise
// returns false.
func (am *AccountManager) RemoveKey(accountID, keyName string) bool {
	_funcName := "AccountManager::RemoveKey"
	_Log.FunctionStarted(_funcName, accountID, keyName)
	defer _Log.FunctionFinished(_funcName)
	change := mgo.Change{
		Remove: true,
	}
	keyID := fmt.Sprintf("%s.%s", accountID, keyName)
	if chInfo, err := _MongoDB.C(COLLECTION_ACCOUNTS_DATA).FindId(keyID).Apply(change, nil); err != nil {
		log.Println("Model::AccountManager::RemoveKey::Error 1::", err.Error())
	} else if chInfo.Removed > 0 {
		if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
			accountID,
			bson.M{"$inc": bson.M{"counters.client_keys": -1}},
		); err != nil {
			log.Println("Model::AccountManager::RemoveKey::Error 2::", err.Error())
		}
		return true
	}
	return false
}

// SaveKey let clients store their data in servers. The value must be string
// or serialized version of the object. This is clients responsibilities to encode/decode
// their data before saving them on the server.
func (am *AccountManager) SaveKey(accountID, keyName, keyValue string) bool {
	_funcName := "AccountManager::SaveKey"
	_Log.FunctionStarted(_funcName, accountID, keyName, keyValue)
	defer _Log.FunctionFinished(_funcName)

	// Get the account's object
	account := am.GetByID(accountID, nil)
	if account.Counters.Keys >= account.Limits.Keys {
		return false
	}

	// Update/Insert the value into database
	keyID := fmt.Sprintf("%s.%s", accountID, keyName)
	defer am.removeKeyCache(keyID)
	if chInfo, err := _MongoDB.C(COLLECTION_ACCOUNTS_DATA).UpsertId(
		keyID,
		bson.M{"$set": bson.M{
			"value":       keyValue,
			"last-update": Timestamp(),
		}},
	); err != nil {
		log.Println("Model::AccountManager::SaveKey::Error 1::", err.Error())
		return false
	} else if chInfo.Matched == 0 {
		// Increase the counter if it was a new key
		if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
			accountID,
			bson.M{"$inc": bson.M{"counters.client_keys": 1}},
		); err != nil {
			log.Println("Model::AccountManager::SaveKey::Error 2::", err.Error())
		}
	}
	return true
}

// SetAdmin sets or resets the  accountID as admin
func (am *AccountManager) SetAdmin(accountID string, b bool) {
	_funcName := "AccountManager::SetAdmin"
	_Log.FunctionStarted(_funcName, accountID, b)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{"$set": bson.M{"authority.admin": b}},
	); err != nil {
		log.Println("Model::AccountManager::SetAdmin::Error::", err.Error())
	}
	return
}

// SetPhone set the phone number of accountID with new 'phone' number
func (am *AccountManager) SetPhone(accountID, phone string) bool {
	_funcName := "AccountManager::SetPhone"
	_Log.FunctionStarted(_funcName, accountID, phone)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{"$set": bson.M{"phone": phone}},
	); err != nil {
		log.Println("Model::AccountManager::SetPhone::Error 1::", err.Error())
		return false
	}
	return true
}

// SetLimit updates account limits
//	Available keys: grand_places
func (am *AccountManager) SetLimit(accountID, limitKey string, n int) bool {
	_funcName := "AccountManager::SetLimit"
	_Log.FunctionStarted(_funcName, accountID, limitKey)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	switch limitKey {
	case "grand_places":
		_MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
			accountID,
			bson.M{"$set": bson.M{"limits.grand_places": n}},
		)
	default:
		return false
	}
	return true
}

// SetPrivacy updates the account's privacy properties
//	Available privacy keys: searchable | change_picture | change_profile
func (am *AccountManager) SetPrivacy(accountID, privacyKey string, privacyValue interface{}) {
	_funcName := "AccountManager::SetPrivacy"
	_Log.FunctionStarted(_funcName, accountID, privacyKey, privacyValue)
	defer _Log.FunctionFinished(_funcName)

	// Remove the old document from cache
	defer _Manager.Account.removeCache(accountID)
	ok := false
	q := bson.M{}
	switch privacyKey {
	case "searchable":
		q["privacy.searchable"], ok = privacyValue.(bool)
	case "change_picture":
		q["privacy.change_picture"], ok = privacyValue.(bool)
	case "change_profile":
		q["privacy.change_profile"], ok = privacyValue.(bool)
	}
	if ok {
		_MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
			accountID,
			bson.M{"$set": q},
		)
	}
}

// SetPicture set the picture structure as the profile picture of the user and his/her personal place
func (am *AccountManager) SetPicture(accountID string, p Picture) {
	_funcName := "AccountManager::SetPicture"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	defer _Manager.Place.removeCache(accountID)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{"$set": bson.M{"picture": p}},
	); err != nil {
		log.Println("Model::AccountManager::SetPicture::Error 1::", err.Error())
	}
	if err := _MongoDB.C(COLLECTION_PLACES).UpdateId(
		accountID,
		bson.M{"$set": bson.M{"picture": p}},
	); err != nil {
		log.Println("Model::AccountManager::SetPicture::Error 2::", err.Error())
	}
}

// SetPlaceNotification set on/off notification of placeID for accountID
func (am *AccountManager) SetPlaceNotification(accountID, placeID string, on bool) *AccountManager {
	_funcName := "AccountManager::SetPlaceNotification"
	_Log.FunctionStarted(_funcName, accountID, placeID, on)
	defer _Log.FunctionFinished(_funcName)
	if p := _Manager.Place.GetByID(placeID, nil); p == nil {
		return am
	} else {
		if on {
			_Manager.Group.AddItems(p.Groups[NOTIFICATION_GROUP], []string{accountID})
		} else {
			_Manager.Group.RemoveItems(p.Groups[NOTIFICATION_GROUP], []string{accountID})
		}
	}
	return am
}

// SetPassword set the password for "accountID" if everything was going through with no problem it returns true
// otherwise returns false
func (am *AccountManager) SetPassword(accountID, newPass string) bool {
	_funcName := "AccountManager::SetPassword"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	if hashed_pass, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost); err != nil {
		log.Println("Model::AccountManager::SetPassword::Error 1::", err.Error())
		return false
	} else {
		if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
			accountID,
			bson.M{"$set": bson.M{
				"secret":                      string(hashed_pass),
				"flags.force_password_change": false,
			}},
		); err != nil {
			log.Println("Model::AccountManager::SetPassword::Error 2::", err.Error())
			return false
		}
	}
	return true
}

// Verify verifies if the username and password match and returns true if they were matched
func (am *AccountManager) Verify(accountID, pass string) bool {
	_funcName := "AccountManager::Verify"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)
	acc := new(Account)
	ch := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"counters.incorrect_attempts": 1}},
		ReturnNew: true,
	}
	if _, err := _MongoDB.C(COLLECTION_ACCOUNTS).Find(
		bson.M{"_id": accountID},
	).Apply(ch, acc); err != nil {
		log.Println("Model::AccountManager::Verify::Error 1::", err.Error())
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(acc.Secret), []byte(pass)); err != nil {
		if acc.Counters.IncorrectAttempts > 10 {
			_Manager.Account.Disable(accountID)
		}
		log.Println("Model::AccountManager::Verify::Error 2::", err.Error())
		return false
	} else {
		_Manager.Account.ResetLoginAttempts(accountID)
	}
	return true
}

// Update updates some fields of the account document, it cannot update all account's properties.
func (am *AccountManager) Update(accountID string, aur AccountUpdateRequest) bool {
	_funcName := "AccountManager::Update"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	defer _Manager.Place.removeCache(accountID)
	q := bson.M{}
	if aur.FirstName != "" {
		q["fname"] = aur.FirstName
	}
	if aur.LastName != "" {
		q["lname"] = aur.LastName
	}
	if aur.DateOfBirth != "" {
		q["dob"] = aur.DateOfBirth
	}
	if aur.Gender != "" {
		q["gender"] = aur.Gender
	}
	if aur.Email != "" {
		q["email"] = aur.Email
	}

	account := new(Account)
	change := mgo.Change{
		Update:    bson.M{"$set": q},
		ReturnNew: true,
	}
	if chInfo, err := _MongoDB.C(COLLECTION_ACCOUNTS).FindId(accountID).Apply(change, account); err != nil {
		return false
	} else if chInfo.Updated > 0 {
		if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
			accountID,
			bson.M{"$set": bson.M{"full_name": fmt.Sprintf("%s %s", account.FirstName, account.LastName)}},
		); err != nil {
			_Log.Error(_funcName, err.Error(), "ACCOUNTS update")
		}
		if err := _MongoDB.C(COLLECTION_PLACES).UpdateId(
			accountID,
			bson.M{"$set": bson.M{"name": fmt.Sprintf("%s %s", account.FirstName, account.LastName)}},
		); err != nil {
			_Log.Error(_funcName, err.Error(), "PLACES update")
		}
	}

	return true
}

// UpdateAuthority updates the authority sub-document of the account document
func (am *AccountManager) UpdateAuthority(accountID string, authority AccountAuthority) bool {
	_funcName := "AccountManager::UpdateAuthority"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{"$set": bson.M{"authority": authority}},
	); err != nil {
		log.Println("Model::AccountManager::UpdateAuthority::Error 1::", err.Error())
		return false
	}
	return true
}

// UpdateLimits updates limit parameters of the accountID,
func (am *AccountManager) UpdateLimits(accountID string, limits MI) bool {
	_funcName := "AccountManager::UpdateLimits"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	m := MI{}
	for limitKey, limitValue := range limits {
		switch limitKey {
		case "limits.grand_places":
			m[limitKey] = ClampInteger(limitValue, SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_LL, SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_UL)
		}
	}
	if len(m) == 0 {
		return false
	}
	if _, err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateAll(
		bson.M{"_id": accountID},
		bson.M{"$set": m},
	); err != nil {
		log.Println("Model::AccountManager::UpdateLimits::Error 1::", err.Error())
		return false
	}
	return true
}

// UpdatePlaceConnection updates 'Account <---> Place' relations points by 'c'
func (am *AccountManager) UpdatePlaceConnection(accountID string, placeIDs []string, c int) {
	_funcName := "AccountManager::UpdatePlaceConnection"
	_Log.FunctionStarted(_funcName, accountID, placeIDs, c)
	defer _Log.FunctionFinished(_funcName)

	bulk := _MongoDB.C(COLLECTION_ACCOUNTS_PLACES).Bulk()
	bulk.Unordered()
	for _, pid := range placeIDs {
		if place := _Manager.Place.GetByID(pid, M{"name": 1}); place != nil {
			bulk.Upsert(
				bson.M{
					"account_id": accountID,
					"place_id":   pid,
				},
				bson.M{
					"$inc": bson.M{"pts": c},
				},
			)
		}
	}
	bulk.Run()
}

// UpdateAccountConnection updates 'Account <---> Account' relations points by 'c'
func (am *AccountManager) UpdateAccountConnection(accountID string, otherAccountIDs []string, c int) {
	_funcName := "AccountManager::UpdateAccountConnection"
	_Log.FunctionStarted(_funcName, accountID, otherAccountIDs, c)
	defer _Log.FunctionFinished(_funcName)

	bulk := _MongoDB.C(COLLECTION_ACCOUNTS_ACCOUNTS).Bulk()
	bulk.Unordered()
	for _, aid := range otherAccountIDs {
		bulk.Upsert(
			bson.M{
				"account_id":       accountID,
				"other_account_id": aid,
			},
			bson.M{
				"$inc": bson.M{"pts": c},
			},
		)
		bulk.Upsert(
			bson.M{
				"account_id":       aid,
				"other_account_id": accountID,
			},
			bson.M{
				"$inc": bson.M{"pts": c},
			},
		)
	}
	bulk.Run()
}

// UpdateRecipientConnection updates 'Account <---> Recipients(Emails) relation points by 'c'
func (am *AccountManager) UpdateRecipientConnection(accountID string, recipients []string, c int) {
	_funcName := "AccountManager::UpdateRecipientConnection"
	_Log.FunctionStarted(_funcName, accountID, recipients)
	defer _Log.FunctionFinished(_funcName)
	for _, r := range recipients {
		if _, err := _MongoDB.C(COLLECTION_ACCOUNTS_RECIPIENTS).Upsert(
			bson.M{
				"account_id": accountID,
				"recipient":  strings.ToLower(r),
			},
			bson.M{"$inc": bson.M{"pts": c}},
		); err != nil {
			log.Println("Model::AccountManager::UpdateRecipientConnection::Error 1::", err.Error())
		}
	}
}

// UnTrustRecipient removes the email address from the trusted lists for the accountID
func (am AccountManager) UnTrustRecipient(accountID string, recipients []string) bool {
	_funcName := "AccountManager::UnTrustRecipient"
	_Log.FunctionStarted(_funcName, accountID, recipients)
	defer _Log.FunctionFinished(_funcName)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS_TRUSTED).UpdateId(
		accountID,
		bson.M{"$pull": bson.M{"recipients": recipients}},
	); err != nil {
		_Log.Error(_funcName, err.Error())
		return false
	}
	return true
}

// TrustRecipient adds the email address to the trusted lists for accountID
func (am AccountManager) TrustRecipient(accountID string, recipients []string) bool {
	_funcName := "AccountManager::TrustRecipient"
	_Log.FunctionStarted(_funcName, accountID, recipients)
	defer _Log.FunctionFinished(_funcName)
	if _, err := _MongoDB.C(COLLECTION_ACCOUNTS_TRUSTED).UpsertId(
		accountID,
		bson.M{
			"$addToSet": bson.M{"recipients": bson.M{"$each": recipients}},
		},
	); err != nil {
		_Log.Error(_funcName, err.Error())
		return false
	}
	return true
}

// IsRecipientTrusted returns TRUE is recipient or its domain is trusted, otherwise returns FALSE
func (am AccountManager) IsRecipientTrusted(accountID string, recipient string) bool {
	_funcName := "AccountManager::IsRecipientTrusted"
	_Log.FunctionStarted(_funcName, accountID, recipient)
	defer _Log.FunctionFinished(_funcName)
	in := []string{recipient}
	emailParts := strings.SplitAfter(recipient, "@")
	if len(emailParts) == 2 {
		in = append(in, fmt.Sprintf("@%s", emailParts[1]))
	}
	if n, err := _MongoDB.C(COLLECTION_ACCOUNTS_TRUSTED).Find(
		bson.M{
			"_id":        accountID,
			"recipients": bson.M{"$in": in},
		},
	).Count(); err != nil {
		return false
	} else if n > 0 {
		return true
	}
	return false
}

// UpdateEmail sets the user's email SMTP settings for out going emails
func (am *AccountManager) UpdateEmail(accountID string, email AccountMail) bool {
	_funcName := "AccountManager::UpdateEmail"
	_Log.FunctionStarted(_funcName, accountID)
	defer _Log.FunctionFinished(_funcName)
	defer _Manager.Account.removeCache(accountID)
	if err := _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(
		accountID,
		bson.M{"$set": bson.M{"mail": email}},
	); err != nil {
		log.Println("Model::AccountManager::UpdateEmail::Error 1::", err.Error())
		return false
	}
	return true
}


func (a *Account) IsBookmarked(placeID string) bool {
	for _, bookmarkedPlaceID := range a.BookmarkedPlaceIDs {
		if placeID == bookmarkedPlaceID {
			return true
		}
	}
	return false
}

