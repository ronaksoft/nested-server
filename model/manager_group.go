package nested

import (
    "github.com/globalsign/mgo/bson"
)

const (
    NOTIFICATION_GROUP = "_ntfy"
)

type GroupManager struct{}

func NewGroupManager() *GroupManager {
    return new(GroupManager)
}

// CreatePlaceGroup creates a group object in database for "placeID" and name it "name" and returns the id of the group
func (gm *GroupManager) CreatePlaceGroup(placeID, name string) string {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    groupID := bson.NewObjectId().Hex() + RandomID(24)
    if err := db.C(COLLECTION_PLACES_GROUPS).Insert(bson.M{
        "_id":   groupID,
        "items": []string{},
    }); err != nil {
        _Log.Warn(err.Error())
    }
    if err := db.C("places").UpdateId(
        placeID,
        bson.M{
            "$set": bson.M{"groups." + name: groupID},
        },
    ); err != nil {
        _Log.Warn(err.Error())
    }
    return groupID
}

// AddItems adds items in the "items" array to the group identified by "groupID"
func (gm *GroupManager) AddItems(groupID string, items []string) {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_PLACES_GROUPS).Update(
        bson.M{"_id": groupID},
        bson.M{"$addToSet": bson.M{
            "items": bson.M{"$each": items},
        }},
    ); err != nil {
        _Log.Warn(err.Error())
    }
}

// RemoveItems removes items in the "items" array from the group identified by "groupID"
func (gm *GroupManager) RemoveItems(groupID string, items []string) {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_PLACES_GROUPS).Update(
        bson.M{"_id": groupID},
        bson.M{"$pullAll": bson.M{"items": items}},
    ); err != nil {
        _Log.Warn(err.Error())
    }
}

// GetItems returns an array of items from "groupID"
func (gm *GroupManager) GetItems(groupID string) []string {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    v := struct {
        ID    string   `json:"_id" bson:"_id"`
        Items []string `json:"items" bson:"items"`
    }{}
    if err := db.C(COLLECTION_PLACES_GROUPS).FindId(groupID).One(&v); err != nil {
        return []string{}
    }
    return v.Items
}

// ItemExists returns true if the item exists in group identified by "groupID"
func (gm *GroupManager) ItemExists(groupID string, item string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    n, _ := db.C(COLLECTION_PLACES_GROUPS).Find(bson.M{
        "_id":   groupID,
        "items": item,
    }).Count()
    if n > 0 {
        return true
    }
    return false
}
