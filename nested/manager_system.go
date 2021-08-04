package nested

import (
	"encoding/json"
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"github.com/globalsign/mgo/bson"
	"github.com/gomodule/redigo/redis"
)

const (
	SYS_INFO_USERAPI = "userapi"
	SYS_INFO_GATEWAY = "gateway"
	SYS_INFO_MSGAPI  = "msgapi"
	SYS_INFO_STORAGE = "storage"
	SYS_INFO_ROUTER  = "router"
)

type SystemConstants struct {
	Integers MI `bson:"integers"`
	Strings  MS `bson:"strings"`
}
type SystemManager struct{}
type MessageTemplate struct {
	Subject string `bson:"subject" json:"subject"`
	Body    string `bson:"body" json:"body"`
}

func NewSystemManager() *SystemManager {
	return new(SystemManager)
}

// GetIntegerConstants returns a map with integer values
func (sm *SystemManager) GetIntegerConstants() MI {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	r := new(SystemConstants)
	if err := db.C(global.CollectionSystemInternal).FindId("constants").One(r); err != nil {
		log.Sugar().Info("Model::SystemManager::GetIntegerConstants::Error 1::", err.Error())
		return nil
	}
	if r.Integers == nil {
		r.Integers = MI{}
	}
	if r.Strings == nil {
		r.Strings = MS{}
	}

	// Place Constants
	if _, ok := r.Integers[global.SystemConstantsPlaceMaxChildren]; !ok {
		r.Integers[global.SystemConstantsPlaceMaxChildren] = global.DefaultPlaceMaxChildren
	}
	if _, ok := r.Integers[global.SystemConstantsPlaceMaxCreators]; !ok {
		r.Integers[global.SystemConstantsPlaceMaxCreators] = global.DefaultPlaceMaxCreators
	}
	if _, ok := r.Integers[global.SystemConstantsPlaceMaxKeyHolders]; !ok {
		r.Integers[global.SystemConstantsPlaceMaxKeyHolders] = global.DefaultPlaceMaxKeyHolders
	}
	if _, ok := r.Integers[global.SystemConstantsPlaceMaxLevel]; !ok {
		r.Integers[global.SystemConstantsPlaceMaxLevel] = global.DefaultPlaceMaxLevel
	}

	// Post Constants
	if _, ok := r.Integers[global.SystemConstantsPostMaxAttachments]; !ok {
		r.Integers[global.SystemConstantsPostMaxAttachments] = global.DefaultPostMaxAttachments
	}
	if _, ok := r.Integers[global.SystemConstantsPostMaxTargets]; !ok {
		r.Integers[global.SystemConstantsPostMaxTargets] = global.DefaultPostMaxTargets
	}
	if _, ok := r.Integers[global.SystemConstantsPostMaxLabels]; !ok {
		r.Integers[global.SystemConstantsPostMaxLabels] = global.DefaultPostMaxLabels
	}
	if _, ok := r.Integers[global.SystemConstantsPostRetractTime]; !ok {
		r.Integers[global.SystemConstantsPostRetractTime] = int(global.DefaultPostRetractTime)
	}

	// Account Constants
	if _, ok := r.Integers[global.SystemConstantsAccountGrandPlaceLimit]; !ok {
		r.Integers[global.SystemConstantsAccountGrandPlaceLimit] = global.DefaultAccountGrandPlaces
	}

	// Label Constants
	if _, ok := r.Integers[global.SystemConstantsLabelMaxMembers]; !ok {
		r.Integers[global.SystemConstantsLabelMaxMembers] = global.DefaultLabelMaxMembers
	}

	// Misc Constants
	if _, ok := r.Integers[global.SystemConstantsCacheLifetime]; !ok {
		r.Integers[global.SystemConstantsCacheLifetime] = global.CacheLifetime
	}
	if _, ok := r.Integers[global.SystemConstantsRegisterMode]; !ok {
		r.Integers[global.SystemConstantsRegisterMode] = global.RegisterMode
	}

	return r.Integers
}

// GetStringConstants returns a map with string values
func (sm *SystemManager) GetStringConstants() MS {
	r := new(SystemConstants)
	if err := _MongoDB.C(global.CollectionSystemInternal).FindId("constants").One(r); err != nil {
		log.Sugar().Info("Model::SystemManager::GetIntegerConstants::Error 1::", err.Error())
		return nil
	}
	if r.Integers == nil {
		r.Integers = MI{}
	}
	if r.Strings == nil {
		r.Strings = MS{}
	}
	// Company Constants
	if _, ok := r.Strings[global.SystemConstantsCompanyName]; !ok {
		r.Strings[global.SystemConstantsCompanyName] = global.DefaultCompanyName
	}
	if _, ok := r.Strings[global.SystemConstantsCompanyDesc]; !ok {
		r.Strings[global.SystemConstantsCompanyDesc] = global.DefaultCompanyDesc
	}
	if _, ok := r.Strings[global.SystemConstantsCompanyLogo]; !ok {
		r.Strings[global.SystemConstantsCompanyLogo] = global.DefaultCompanyLogo
	}
	if _, ok := r.Strings[global.SystemConstantsSystemLang]; !ok {
		r.Strings[global.SystemConstantsSystemLang] = global.DefaultSystemLang
	}
	if _, ok := r.Strings[global.SystemConstantsMagicNumber]; !ok {
		r.Strings[global.SystemConstantsMagicNumber] = global.DefaultMagicNumber
	}
	if _, ok := r.Strings[global.SystemConstantsLicenseKey]; !ok {
		r.Strings[global.SystemConstantsLicenseKey] = ""
	}
	return r.Strings
}

// GetCounters returns the counts of
//  1. Active Accounts
//  2. Disabled Accounts
//  3. Personal Places
//  4. Grand Places
//  5. Private Places
//  6. Common Places
func (sm *SystemManager) GetCounters() MI {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	m := MI{}
	if err := db.C(global.CollectionSystemInternal).FindId("counters").One(m); err != nil {
		log.Warn(err.Error())
	}
	return m
}

// SetIntegerConstants set system wide integer setting parameters, they constants until admin
// reset them again
func (sm *SystemManager) SetIntegerConstants(m tools.M) {
	q := bson.M{}
	for key, v := range m {
		var iVal int
		switch v.(type) {
		case int:
			iVal = v.(int)
		case float64:
			iVal = int(v.(float64))
		}
		switch key {
		case global.SystemConstantsAccountGrandPlaceLimit:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SystemConstantsAccountGrandPlaceLimitLL,
				global.SystemConstantsAccountGrandPlaceLimitUL,
			)
		case global.SystemConstantsPlaceMaxChildren:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SystemConstantsPlaceMaxChildrenLL,
				global.SystemConstantsPlaceMaxChildrenUL,
			)
		case global.SystemConstantsPlaceMaxCreators:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SystemConstantsPlaceMaxCreatorsLL,
				global.SystemConstantsPlaceMaxCreatorsUl,
			)
		case global.SystemConstantsPlaceMaxKeyHolders:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SystemConstantsPlaceMaxKeyHoldersLL,
				global.SystemConstantsPlaceMaxKeyHoldersUL,
			)
		case global.SystemConstantsPlaceMaxLevel:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SystemConstantsPlaceMaxLevelLL,
				global.SystemConstantsPlaceMaxLevelUL,
			)
		case global.SystemConstantsPostMaxAttachments:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SystemConstantsPostMaxTargetsLL,
				global.SystemConstantsPostMaxTargetsUL,
			)
		case global.SystemConstantsPostMaxTargets:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SystemConstantsPostMaxTargetsLL,
				global.SystemConstantsPostMaxTargetsUL,
			)
		case global.SystemConstantsPostMaxLabels:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SystemConstantsPostMaxLabelsLL,
				global.SystemConstantsPostMaxLabelsUL,
			)
		case global.SystemConstantsPostRetractTime:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SystemConstantsPostRetractTimeLL,
				global.SystemConstantsPostRetractTimeUL,
			)
		case global.SystemConstantsLabelMaxMembers:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SystemConstantsLabelMaxMembersLL,
				global.SystemConstantsLabelMaxMembersUL,
			)
		case global.SystemConstantsCacheLifetime:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SystemConstantsCacheLifetimeLL,
				global.SystemConstantsCacheLifetimeUL,
			)
		case global.SystemConstantsRegisterMode:
			switch iVal {
			case global.RegisterModeAdminOnly, global.RegisterModeEveryone:
			default:
				iVal = global.RegisterModeAdminOnly
			}
			q[fmt.Sprintf("integers.%s", key)] = iVal
		}

	}
	_MongoDB.C(global.CollectionSystemInternal).UpdateId(
		"constants",
		bson.M{"$set": q},
	)
	sm.LoadIntegerConstants()
}

// SetStringConstants set system wide string setting parameters, they constants until admin
// reset them again
func (sm *SystemManager) SetStringConstants(m tools.M) {
	q := bson.M{}
	for key, v := range m {
		var sVal string
		switch v.(type) {
		case string:
			sVal = v.(string)
		}
		switch key {
		case global.SystemConstantsCompanyName, global.SystemConstantsCompanyDesc,
			global.SystemConstantsCompanyLogo, global.SystemConstantsMagicNumber,
			global.SystemConstantsSystemLang, global.SystemConstantsLicenseKey:
			q[fmt.Sprintf("strings.%s", key)] = sVal
		}
	}
	_MongoDB.C(global.CollectionSystemInternal).UpdateId(
		"constants",
		bson.M{"$set": q},
	)
	sm.LoadStringConstants()
}

func (sm *SystemManager) LoadIntegerConstants() {
	iConstants := sm.GetIntegerConstants()
	// Place Constants
	global.DefaultPlaceMaxChildren = ClampInteger(
		iConstants[global.SystemConstantsPlaceMaxChildren],
		global.SystemConstantsPlaceMaxChildrenLL,
		global.SystemConstantsPlaceMaxChildrenUL,
	)
	global.DefaultPlaceMaxCreators = ClampInteger(
		iConstants[global.SystemConstantsPlaceMaxCreators],
		global.SystemConstantsPlaceMaxCreatorsLL,
		global.SystemConstantsPlaceMaxCreatorsUl,
	)
	global.DefaultPlaceMaxKeyHolders = ClampInteger(
		iConstants[global.SystemConstantsPlaceMaxKeyHolders],
		global.SystemConstantsPlaceMaxKeyHoldersLL,
		global.SystemConstantsPlaceMaxKeyHoldersUL,
	)
	global.DefaultPlaceMaxLevel = ClampInteger(
		iConstants[global.SystemConstantsPlaceMaxLevel],
		global.SystemConstantsPlaceMaxLevelLL,
		global.SystemConstantsPlaceMaxLevelUL,
	)

	// Post Constants
	global.DefaultPostMaxAttachments = ClampInteger(
		iConstants[global.SystemConstantsPostMaxAttachments],
		global.SystemConstantsPostMaxAttachmentsLL,
		global.SystemConstantsPostMaxAttachmentsUL,
	)
	global.DefaultPostMaxTargets = ClampInteger(
		iConstants[global.SystemConstantsPostMaxTargets],
		global.SystemConstantsPostMaxTargetsLL,
		global.SystemConstantsPostMaxTargetsUL,
	)
	global.DefaultPostRetractTime = uint64(ClampInteger(
		iConstants[global.SystemConstantsPostRetractTime],
		global.SystemConstantsPostRetractTimeLL,
		global.SystemConstantsPostRetractTimeUL,
	))
	global.DefaultPostMaxLabels = ClampInteger(
		iConstants[global.SystemConstantsPostMaxLabels],
		global.SystemConstantsPostMaxLabelsLL,
		global.SystemConstantsPostMaxLabelsUL,
	)

	// Account Constants
	global.DefaultAccountGrandPlaces = ClampInteger(
		iConstants[global.SystemConstantsAccountGrandPlaceLimit],
		global.SystemConstantsAccountGrandPlaceLimitLL,
		global.SystemConstantsAccountGrandPlaceLimitUL,
	)

	// Label Constants
	global.DefaultLabelMaxMembers = ClampInteger(
		iConstants[global.SystemConstantsLabelMaxMembers],
		global.SystemConstantsLabelMaxMembersLL,
		global.SystemConstantsLabelMaxMembersUL,
	)

	// Misc Constants
	global.CacheLifetime = ClampInteger(
		iConstants[global.SystemConstantsCacheLifetime],
		global.SystemConstantsCacheLifetimeLL,
		global.SystemConstantsCacheLifetimeUL,
	)

	switch iConstants[global.SystemConstantsRegisterMode] {
	case global.RegisterModeAdminOnly, global.RegisterModeEveryone:
		global.RegisterMode = iConstants[global.SystemConstantsRegisterMode]
	default:
		global.RegisterMode = global.RegisterModeAdminOnly
	}
}

func (sm *SystemManager) LoadStringConstants() {
	sConstants := sm.GetStringConstants()
	global.DefaultCompanyName = sConstants[global.SystemConstantsCompanyName]
	global.DefaultCompanyDesc = sConstants[global.SystemConstantsCompanyDesc]
	global.DefaultCompanyLogo = sConstants[global.SystemConstantsCompanyLogo]
	global.DefaultMagicNumber = sConstants[global.SystemConstantsMagicNumber]
}

func (sm *SystemManager) SetMessageTemplate(msgID, msgSubject, msgBody string) bool {
	if _, err := _MongoDB.C(global.CollectionSystemInternal).UpsertId(
		"message_templates",
		bson.M{"$set": bson.M{
			msgID: bson.M{
				"subject": msgSubject,
				"body":    msgBody,
			}}},
	); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

func (sm *SystemManager) GetMessageTemplates() map[string]MessageTemplate {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	templates := make(map[string]MessageTemplate)
	if err := db.C(global.CollectionSystemInternal).FindId("message_templates").One(&templates); err != nil {
		log.Warn(err.Error())
	}
	return templates
}

func (sm *SystemManager) RemoveMessageTemplate(msgID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.CollectionSystemInternal).UpdateId(
		"message_templates",
		bson.M{
			"$unset": bson.M{msgID: ""},
		}); err != nil {
		log.Warn(err.Error())
	}

}

func (sm *SystemManager) SetSystemInfo(key, bundleID string, info tools.M) bool {
	switch key {
	case SYS_INFO_GATEWAY, SYS_INFO_MSGAPI, SYS_INFO_ROUTER,
		SYS_INFO_STORAGE, SYS_INFO_USERAPI:
	default:
		return false
	}
	keyID := fmt.Sprintf("sysinfo.%s", key)
	c := _Cache.GetConn()
	defer c.Close()
	if jsonInfo, err := json.Marshal(info); err != nil {
		log.Warn(err.Error())
	} else {
		c.Do("HSET", keyID, bundleID, jsonInfo)
	}
	return true
}

func (sm *SystemManager) GetSystemInfo(key string) MS {
	keyID := fmt.Sprintf("sysinfo.%s", key)
	c := _Cache.GetConn()
	defer c.Close()
	info := MS{}
	if m, err := redis.StringMap(c.Do("HGETALL", keyID)); err != nil {
		log.Warn(err.Error())
		return nil
	} else {
		for k, v := range m {
			info[k] = v
		}
	}
	return info
}

func (sm *SystemManager) SetCounter(params MI) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	for key := range params {
		switch key {
		case global.SystemCountersEnabledAccounts, global.SystemCountersDisabledAccounts,
			global.SystemCountersGrandPlaces, global.SystemCountersLockedPlaces, global.SystemCountersUnlockedPlaces,
			global.SystemCountersPersonalPlaces:
		default:
			delete(params, key)
		}
	}
	if _, err := db.C(global.CollectionSystemInternal).UpsertId(
		"counters",
		bson.M{"$set": params},
	); err != nil {
		log.Sugar().Info("Model::SystemManager::SetCounter::Error 1::", err.Error())
		return false
	}
	return true
}

// Private methods
func (sm *SystemManager) setDataModelVersion(n int) bool {
	if _, err := _MongoDB.C(global.CollectionSystemInternal).UpsertId(
		"constants",
		bson.M{"$set": bson.M{global.SystemConstantsModelVersion: n}},
	); err != nil {
		log.Sugar().Info("Model::SystemManager::SetDataModelVersion::Error 1::", err.Error())
		return false
	}
	return true
}

func (sm *SystemManager) getDataModelVersion() int {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	r := MI{}
	if err := db.C(global.CollectionSystemInternal).FindId("constants").One(r); err != nil {
		return global.DefaultModelVersion
	}
	model, _ := r[global.SystemConstantsModelVersion]
	return model
}

func (sm *SystemManager) incrementCounter(params MI) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	for key := range params {
		switch key {
		case global.SystemCountersEnabledAccounts, global.SystemCountersDisabledAccounts,
			global.SystemCountersGrandPlaces, global.SystemCountersLockedPlaces, global.SystemCountersUnlockedPlaces:
		default:
			delete(params, key)
		}
	}
	if _, err := db.C(global.CollectionSystemInternal).UpsertId(
		"counters",
		bson.M{"$inc": params},
	); err != nil {
		log.Sugar().Info("Model::SystemManager::IncrementCounter::Error 1::", err.Error())
		return false
	}
	return true
}
