package nested

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"github.com/globalsign/mgo/bson"
	"go.uber.org/zap"
)

const (
	NotificationGroup = "_ntfy"
)

type GroupManager struct{}

func newGroupManager() *GroupManager {
	return new(GroupManager)
}

// CreatePlaceGroup creates a group object in database for "placeID" and name it "name" and returns the id of the group
func (gm *GroupManager) CreatePlaceGroup(placeID, name string) string {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	groupID := bson.NewObjectId().Hex() + RandomID(24)
	if err := db.C(global.CollectionPlacesGroups).Insert(bson.M{
		"_id":   groupID,
		"items": []string{},
	}); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	if err := db.C("places").UpdateId(
		placeID,
		bson.M{
			"$set": bson.M{"groups." + name: groupID},
		},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return groupID
}

// AddItems adds items in the "items" array to the group identified by "groupID"
func (gm *GroupManager) AddItems(groupID string, items []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionPlacesGroups).Update(
		bson.M{"_id": groupID},
		bson.M{"$addToSet": bson.M{
			"items": bson.M{"$each": items},
		}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
}

// RemoveItems removes items in the "items" array from the group identified by "groupID"
func (gm *GroupManager) RemoveItems(groupID string, items []string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionPlacesGroups).Update(
		bson.M{"_id": groupID},
		bson.M{"$pullAll": bson.M{"items": items}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
}

// GetItems returns an array of items from "groupID"
func (gm *GroupManager) GetItems(groupID string) []string {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := struct {
		ID    string   `json:"_id" bson:"_id"`
		Items []string `json:"items" bson:"items"`
	}{}
	if err := db.C(global.CollectionPlacesGroups).FindId(groupID).One(&v); err != nil {
		return []string{}
	}
	return v.Items
}

// ItemExists returns true if the item exists in group identified by "groupID"
func (gm *GroupManager) ItemExists(groupID string, item string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	n, _ := db.C(global.CollectionPlacesGroups).Find(bson.M{
		"_id":   groupID,
		"items": item,
	}).Count()
	if n > 0 {
		return true
	}
	return false
}
