package nested

import (
    "github.com/globalsign/mgo/bson"
    "time"
)

const (
    DEVICE_OS_ANDROID  = "android"
    DEVICE_OS_FIREFOX  = "firefox"
    DEVICE_OS_CHROME   = "chrome"
    DEVICE_OS_IOS      = "ios"
    DEVICE_OS_SAFARI   = "safari"
)

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

type DeviceManager struct{}

func NewDeviceManager() *DeviceManager { return new(DeviceManager) }

func (dm *DeviceManager) GetByAccountID(accountID string) []Device {
    _funcName := "DeviceManager::GetAccountByID"
    _Log.FunctionStarted(_funcName, accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    var devices []Device
    if err := db.C(COLLECTION_ACCOUNTS_DEVICES).Find(bson.M{"uid": accountID}).Limit(DEFAULT_MAX_RESULT_LIMIT).All(&devices); err != nil {
        _Log.Error(_funcName, err.Error())
    }
    return devices
}

func (dm *DeviceManager) IncrementBadge(accountID string) {
    _funcName := "DeviceManager::IncrementBadge"
    _Log.FunctionStarted(_funcName, accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if _, err := db.C(COLLECTION_ACCOUNTS_DEVICES).UpdateAll(
        bson.M{"uid": accountID},
        bson.M{"$inc": bson.M{"badge": 1}},
    ); err != nil {
        _Log.Error(_funcName, err.Error())
    }
}

func (dm *DeviceManager) Register(deviceID, deviceToken, deviceOS, accountID string) bool {
    _funcName := "DeviceManager::Register"
    _Log.FunctionStarted(_funcName, deviceID, deviceToken, deviceOS, accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    // only supported devices
    switch deviceOS {
    case DEVICE_OS_ANDROID, DEVICE_OS_CHROME, DEVICE_OS_FIREFOX,
        DEVICE_OS_IOS, DEVICE_OS_SAFARI:
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
    if err := db.C(COLLECTION_ACCOUNTS_DEVICES).Insert(d); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }

    return true
}

func (dm *DeviceManager) Remove(deviceID string) bool {
    _funcName := "DeviceManager::Remove"
    _Log.FunctionStarted(_funcName, deviceID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_ACCOUNTS_DEVICES).Remove(bson.M{"_id": deviceID}); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }
    return true
}

func (dm *DeviceManager) SetAsConnected(deviceID, accountID string) bool {
    _funcName := "DeviceManager::SetAsConnected"
    _Log.FunctionStarted(_funcName, deviceID, accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    bulk := db.C(COLLECTION_ACCOUNTS_DEVICES).Bulk()
    bulk.UpdateAll(
        bson.M{"uid": accountID},
        bson.M{"$set": bson.M{"badge": 0}},
    )
    bulk.Update(
        bson.M{"_id": deviceID},
        bson.M{"$set": bson.M{"connected": true}},
    )
    if _, err := bulk.Run(); err != nil {
        _Log.Error(_funcName, err.Error())
        return false
    }
    return true
}

func (dm *DeviceManager) SetAsDisconnected(deviceID string) bool {
    _funcName := "DeviceManager::SetAsDisconnected"
    _Log.FunctionStarted(_funcName, deviceID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_ACCOUNTS_DEVICES).Update(
        bson.M{"_id": deviceID},
        bson.M{"$set": bson.M{"connected": false}},
    ); err != nil {
        _Log.Error(_funcName, err.Error(), deviceID)
        return false
    }
    return true
}

func (dm *DeviceManager) Update(deviceID, deviceToken, deviceOS, accountID string) bool {
    _funcName := "DeviceManager::Update"
    _Log.FunctionStarted(_funcName, deviceID, deviceToken, deviceOS, accountID)
    defer _Log.FunctionFinished(_funcName)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_ACCOUNTS_DEVICES).UpdateId(
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
