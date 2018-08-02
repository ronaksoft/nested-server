package nested

import (
    "github.com/globalsign/mgo/bson"
    "log"
)

// Phone Manager and Methods
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

func NewPhoneManager() *PhoneManager {
    return new(PhoneManager)
}

// Description:
// This function registers the accountID for the phoneNumber
func (pm *PhoneManager) RegisterPhoneToAccount(accountID, phoneNumber string) {
    _funcName := "PhoneManager::RegisterPhoneToAccount"
    _Log.FunctionStarted(_funcName, accountID, phoneNumber)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if _, err := db.C(COLLECTION_PHONES).UpsertId(
        phoneNumber,
        bson.M{"$set": bson.M{"owner_id": accountID}},
    ); err != nil {
        log.Println("Model::PhoneManager::RegisterPhoneToAccount::Error::1::", err.Error())
    }
    return
}

// Description:
// UnRegisterPhoneToAccount  un-registers the accountID for the phoneNumber, This function must be called when
// user changes his/her phone number
func (pm *PhoneManager) UnRegisterPhoneToAccount(accountID, phoneNumber string) {
    _funcName := "PhoneManager::UnRegisterPhoneToAccount"
    _Log.FunctionStarted(_funcName, accountID, phoneNumber)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if _, err := db.C(COLLECTION_PHONES).UpsertId(
        phoneNumber,
        bson.M{"$unset": bson.M{"owner_id": ""}},
    ); err != nil {
        log.Println("Model::PhoneManager::UnRegisterPhoneToAccount::Error::1::", err.Error())
    }
    return
}

// Description:
// If 'phoneNumber' is in contacts of 'accountID' then add 'accountID' to the list of accounts
// which attached to 'phoneNumber'
func (pm *PhoneManager) AddContactToPhone(accountID, phoneNumber string) {
    _funcName := "PhoneManager::AddContactToPhone"
    _Log.FunctionStarted(_funcName, accountID, phoneNumber)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if _, err := db.C(COLLECTION_PHONES).UpsertId(
        phoneNumber,
        bson.M{"$addToSet": bson.M{"contacts": accountID}},
    ); err != nil {
        log.Println("Model::PhoneManager::AddContactToPhone::Error::1::", err.Error())
    }
    return
}

// Description:
// If 'phoneNumber' is not in contacts of the 'accountID' anymore then remove it from the list
func (pm *PhoneManager) RemoveContactFromPhone(accountID, phoneNumber string) {
    _funcName := "PhoneManager::RemoveContactFromPhone"
    _Log.FunctionStarted(_funcName, accountID, phoneNumber)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if _, err := db.C(COLLECTION_PHONES).UpsertId(
        phoneNumber,
        bson.M{"$pull": bson.M{"contacts": accountID}},
    ); err != nil {
        log.Println("Model::PhoneManager::RemoveContactFromPhone::Error::1::", err.Error())
    }
    return
}

// Description:
// Returns an array of accountIDs who have this number in their contact list
func (pm *PhoneManager) GetContactsByPhoneNumber(phoneNumber string) []string {
    _funcName := "PhoneManager::GetContactsByPhoneNumber"
    _Log.FunctionStarted(_funcName, phoneNumber)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    c := new(PhoneContacts)
    db.C(COLLECTION_PHONES).FindId(phoneNumber).One(c)
    return c.Contacts
}

// Description:
// Returns an array of account ids who have the number owned by 'accountID'
func (pm *PhoneManager) GetContactsByAccountID(accountID string) []string {
    _funcName := "PhoneManager::GetContactsByAccountID"
    _Log.FunctionStarted(_funcName, accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    c := new(PhoneContacts)
    if err := db.C(COLLECTION_PHONES).Find(bson.M{"owner_id": accountID}).One(c); err != nil {
        return []string{}
    }
    return c.Contacts
}
