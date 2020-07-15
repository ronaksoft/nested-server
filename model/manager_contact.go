package nested

import (
	"github.com/globalsign/mgo/bson"
)

type Contacts struct {
	ID               string   `bson:"_id" json:"_id"`
	Hash             string   `bson:"hash" json:"hash"`
	Contacts         []string `bson:"contacts" json:"contacts"`
	MutualContacts   []string `bson:"mutual_contacts" json:"mutual_contacts"`
	FavoriteContacts []string `bson:"favorite_contacts" json:"favorite_contacts"`
}

type ContactManager struct{}

func NewContactManager() *ContactManager { return new(ContactManager) }

func (cm *ContactManager) AddContact(accountID, contactID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(DB_NAME)
	defer dbSession.Close()

	if cm.IsContact(contactID, accountID) {
		cm.AddMutualContact(accountID, contactID)
		return true
	}

	if _, err := db.C(COLLECTION_CONTACTS).UpsertId(
		accountID,
		bson.M{
			"$addToSet": bson.M{"contacts": contactID},
			"$set":      bson.M{"hash": RandomID(8)},
		},
	); err != nil {
		_Log.Warn(err.Error())
	}

	_Manager.Account.UpdateAccountConnection(accountID, []string{contactID}, 1)

	return true
}

func (cm *ContactManager) AddMutualContact(accountID1, accountID2 string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(DB_NAME)
	defer dbSession.Close()

	bulk := db.C(COLLECTION_CONTACTS).Bulk()

	bulk.Upsert(
		bson.M{"_id": accountID1},
		bson.M{
			"$addToSet": bson.M{"contacts": accountID2, "mutual_contacts": accountID2},
			"$set":      bson.M{"hash": RandomID(8)},
		},
		bson.M{"_id": accountID2},
		bson.M{
			"$addToSet": bson.M{"contacts": accountID1, "mutual_contacts": accountID1},
			"$set":      bson.M{"hash": RandomID(8)},
		},
	)
	if _, err := bulk.Run(); err != nil {
		_Log.Warn(err.Error())
		return false
	}
	return true
}

func (cm *ContactManager) AddContactToFavorite(accountID, contactID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(DB_NAME)
	defer dbSession.Close()

	if err := db.C(COLLECTION_CONTACTS).Update(
		bson.M{"_id": accountID, "contacts": contactID},
		bson.M{
			"$addToSet": bson.M{"favorite_contacts": contactID},
			"$set":      bson.M{"hash": RandomID(8)},
		},
	); err != nil {
		_Log.Warn(err.Error())
		return false
	}

	_Manager.Account.UpdateAccountConnection(accountID, []string{contactID}, 5)
	return true
}

func (cm *ContactManager) IsContact(accountID, contactID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(DB_NAME)
	defer dbSession.Close()

	if n, _ := db.C(COLLECTION_CONTACTS).Find(
		bson.M{"_id": accountID, "contacts": contactID},
	).Count(); n > 0 {
		return true
	}
	return false
}

func (cm *ContactManager) RemoveContact(accountID, contactID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(DB_NAME)
	defer dbSession.Close()

	bulk := db.C(COLLECTION_CONTACTS).Bulk()
	bulk.Update(
		bson.M{"_id": accountID},
		bson.M{
			"$pull": bson.M{"contacts": contactID, "mutual_contacts": contactID, "favorite_contacts": contactID},
			"$set":  bson.M{"hash": RandomID(8)},
		},
		bson.M{"_id": contactID},
		bson.M{
			"$pull": bson.M{"mutual_contacts": accountID},
			"$set":  bson.M{"hash": RandomID(8)},
		},
	)
	if _, err := bulk.Run(); err != nil {
		_Log.Warn(err.Error())
		return false
	}
	_Manager.Account.UpdateAccountConnection(accountID, []string{contactID}, -1)
	return true
}

func (cm *ContactManager) RemoveContactFromFavorite(accountID, contactID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(DB_NAME)
	defer dbSession.Close()

	if err := db.C(COLLECTION_CONTACTS).Update(
		bson.M{"_id": accountID, "favorite_contacts": contactID},
		bson.M{
			"$pull": bson.M{"favorite_contacts": contactID},
			"$set":  bson.M{"hash": RandomID(8)},
		},
	); err != nil {
		_Log.Warn(err.Error())
		return false
	}
	_Manager.Account.UpdateAccountConnection(accountID, []string{contactID}, -5)
	return true
}

func (cm *ContactManager) GetContacts(accountID string) Contacts {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(DB_NAME)
	defer dbSession.Close()

	c := Contacts{}
	if err := db.C(COLLECTION_CONTACTS).FindId(accountID).One(&c); err != nil {
		_Log.Warn(err.Error())
	}
	return c
}
