package session

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"time"
)

/*
   Creation Time: 2021 - Aug - 04
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/


type Device struct {
	ID           string `bson:"_id"`
	Token        string `bson:"_dt"`
	OS           string `bson:"os"`
	UID          string `bson:"uid"`
	Badge        int    `bson:"badge"`
	Connected    bool   `bson:"connected"`
	RegisteredOn int64  `bson:"registered_on"`
	UpdatedOn    int64  `bson:"updated_on"`
	TotalUpdates int    `bson:"total_updates"`
}

type DeviceManager struct {
	s *mgo.Session
}

func NewDeviceManager(s *mgo.Session) *DeviceManager {
	return &DeviceManager{
		s: s,
	}
}

func (dm *DeviceManager) GetByAccountID(accountID string) []Device {
	dbSession := dm.s.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	var devices []Device
	if err := db.C(global.COLLECTION_ACCOUNTS_DEVICES).Find(bson.M{"uid": accountID}).Limit(global.DEFAULT_MAX_RESULT_LIMIT).All(&devices); err != nil {
		log.Warn(err.Error())
	}
	return devices
}

func (dm *DeviceManager) IncrementBadge(accountID string) {
	dbSession := dm.s.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	if _, err := db.C(global.COLLECTION_ACCOUNTS_DEVICES).UpdateAll(
		bson.M{"uid": accountID},
		bson.M{"$inc": bson.M{"badge": 1}},
	); err != nil {
		log.Warn(err.Error())
	}
}

func (dm *DeviceManager) Register(deviceID, deviceToken, deviceOS, accountID string) bool {
	dbSession := dm.s.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	// only supported devices
	switch deviceOS {
	case global.PLATFORM_ANDROID, global.PLATFORM_CHROME, global.PLATFORM_FIREFOX,
		global.PLATFORM_IOS, global.PLATFORM_SAFARI:
	default:
		return false
	}
	d := Device{
		ID:           deviceID,
		Token:        deviceToken,
		OS:           deviceOS,
		UID:          accountID,
		Badge:        0,
		Connected:    false,
		RegisteredOn: time.Now().Unix(),
		UpdatedOn:    time.Now().Unix(),
		TotalUpdates: 0,
	}
	if err := db.C(global.COLLECTION_ACCOUNTS_DEVICES).Insert(d); err != nil {
		_ = db.C(global.COLLECTION_ACCOUNTS_DEVICES).UpdateId(
			deviceID,
			bson.M{
				"$set": bson.M{
					"_dt": deviceToken,
					"os":  deviceOS,
					"uid": accountID,
				},
			})
		return false
	}

	return true
}

func (dm *DeviceManager) Remove(deviceID string) bool {
	dbSession := dm.s.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_ACCOUNTS_DEVICES).Remove(bson.M{"_id": deviceID}); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

func (dm *DeviceManager) SetAsConnected(deviceID, accountID string) bool {
	dbSession := dm.s.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	bulk := db.C(global.COLLECTION_ACCOUNTS_DEVICES).Bulk()
	bulk.UpdateAll(
		bson.M{"uid": accountID},
		bson.M{"$set": bson.M{"badge": 0}},
	)
	bulk.Update(
		bson.M{"_id": deviceID},
		bson.M{"$set": bson.M{"connected": true}},
	)
	if _, err := bulk.Run(); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

func (dm *DeviceManager) SetAsDisconnected(deviceID string) bool {
	dbSession := dm.s.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_ACCOUNTS_DEVICES).Update(
		bson.M{"_id": deviceID},
		bson.M{"$set": bson.M{"connected": false}},
	); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

func (dm *DeviceManager) Update(deviceID, deviceToken, deviceOS, accountID string) bool {
	dbSession := dm.s.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_ACCOUNTS_DEVICES).UpdateId(
		deviceID,
		bson.M{
			"$set": bson.M{
				"uid":        accountID,
				"_dt":        deviceToken,
				"updated_on": time.Now().Unix(),
			},
			"$inc": bson.M{"total_updates": 1},
		},
	); err != nil {
		return dm.Register(deviceID, deviceToken, deviceOS, accountID)
	}
	return true
}
