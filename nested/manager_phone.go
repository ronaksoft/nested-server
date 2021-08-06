package nested

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"github.com/globalsign/mgo/bson"
	"go.uber.org/zap"
)

type PhoneManager struct {
	m *Manager
}

type PhoneContacts struct {
	PhoneNumber string   `json:"_id" bson:"_id"`
	OwnerID     string   `json:"owner_id" bson:"owner_id"`
	Contacts    []string `json:"contacts" bson:"contacts"`
}
type AccountContact struct {
	AccountID string   `json:"account_id" bson:"account_id"`
	ContactID string   `json:"contact_id" bson:"contact_id"`
	Phones    []string `json:"phones" bson:"phones"`
	Emails    []string `json:"emails" bson:"emails"`
}

func newPhoneManager() *PhoneManager {
	return new(PhoneManager)
}

// RegisterPhoneToAccount This function registers the accountID for the phoneNumber
func (pm *PhoneManager) RegisterPhoneToAccount(accountID, phoneNumber string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if _, err := db.C(global.CollectionPhones).UpsertId(
		phoneNumber,
		bson.M{"$set": bson.M{"owner_id": accountID}},
	); err != nil {
		log.Sugar().Info("Model::PhoneManager::RegisterPhoneToAccount::Error::1::", err.Error())
	}
	return
}

// UnRegisterPhoneToAccount  un-registers the accountID for the phoneNumber, This function must be called when
// user changes his/her phone number
func (pm *PhoneManager) UnRegisterPhoneToAccount(accountID, phoneNumber string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if _, err := db.C(global.CollectionPhones).UpsertId(
		phoneNumber,
		bson.M{"$unset": bson.M{"owner_id": ""}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return
}

// AddContactToPhone If 'phoneNumber' is in contacts of 'accountID' then add 'accountID' to the list of accounts
// which attached to 'phoneNumber'
func (pm *PhoneManager) AddContactToPhone(accountID, phoneNumber string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if _, err := db.C(global.CollectionPhones).UpsertId(
		phoneNumber,
		bson.M{"$addToSet": bson.M{"contacts": accountID}},
	); err != nil {
		log.Sugar().Info("Model::PhoneManager::AddContactToPhone::Error::1::", err.Error())
	}
	return
}

// RemoveContactFromPhone
// If 'phoneNumber' is not in contacts of the 'accountID' anymore then remove it from the list
func (pm *PhoneManager) RemoveContactFromPhone(accountID, phoneNumber string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if _, err := db.C(global.CollectionPhones).UpsertId(
		phoneNumber,
		bson.M{"$pull": bson.M{"contacts": accountID}},
	); err != nil {
		log.Sugar().Info("Model::PhoneManager::RemoveContactFromPhone::Error::1::", err.Error())
	}
	return
}

// GetContactsByPhoneNumber
// Returns an array of accountIDs who have this number in their contact list
func (pm *PhoneManager) GetContactsByPhoneNumber(phoneNumber string) []string {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	c := new(PhoneContacts)
	db.C(global.CollectionPhones).FindId(phoneNumber).One(c)
	return c.Contacts
}

// GetContactsByAccountID
// Returns an array of account ids who have the number owned by 'accountID'
func (pm *PhoneManager) GetContactsByAccountID(accountID string) []string {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	c := new(PhoneContacts)
	if err := db.C(global.CollectionPhones).Find(bson.M{"owner_id": accountID}).One(c); err != nil {
		return []string{}
	}
	return c.Contacts
}
