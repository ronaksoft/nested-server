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
	if err := db.C(global.COLLECTION_SYSTEM_INTERNAL).FindId("constants").One(r); err != nil {
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
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN] = global.DefaultPlaceMaxChildren
	}
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_PLACE_MAX_CREATORS]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_PLACE_MAX_CREATORS] = global.DefaultPlaceMaxCreators
	}
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS] = global.DefaultPlaceMaxKeyHolders
	}
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_PLACE_MAX_LEVEL]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_PLACE_MAX_LEVEL] = global.DefaultPlaceMaxLevel
	}

	// Post Constants
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS] = global.DefaultPostMaxAttachments
	}
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_POST_MAX_TARGETS]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_POST_MAX_TARGETS] = global.DefaultPostMaxTargets
	}
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_POST_MAX_LABELS]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_POST_MAX_LABELS] = global.DefaultPostMaxLabels
	}
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_POST_RETRACT_TIME]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_POST_RETRACT_TIME] = int(global.DefaultPostRetractTime)
	}

	// Account Constants
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT] = global.DefaultAccountGrandPlaces
	}

	// Label Constants
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS] = global.DefaultLabelMaxMembers
	}

	// Misc Constants
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_CACHE_LIFETIME]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_CACHE_LIFETIME] = global.CacheLifetime
	}
	if _, ok := r.Integers[global.SYSTEM_CONSTANTS_REGISTER_MODE]; !ok {
		r.Integers[global.SYSTEM_CONSTANTS_REGISTER_MODE] = global.RegisterModeAdminOnly
	}

	return r.Integers
}

// GetStringConstants returns a map with string values
func (sm *SystemManager) GetStringConstants() MS {
	r := new(SystemConstants)
	if err := _MongoDB.C(global.COLLECTION_SYSTEM_INTERNAL).FindId("constants").One(r); err != nil {
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
	if _, ok := r.Strings[global.SYSTEM_CONSTANTS_COMPANY_NAME]; !ok {
		r.Strings[global.SYSTEM_CONSTANTS_COMPANY_NAME] = global.DefaultCompanyName
	}
	if _, ok := r.Strings[global.SYSTEM_CONSTANTS_COMPANY_DESC]; !ok {
		r.Strings[global.SYSTEM_CONSTANTS_COMPANY_DESC] = global.DefaultCompanyDesc
	}
	if _, ok := r.Strings[global.SYSTEM_CONSTANTS_COMPANY_LOGO]; !ok {
		r.Strings[global.SYSTEM_CONSTANTS_COMPANY_LOGO] = global.DefaultCompanyLogo
	}
	if _, ok := r.Strings[global.SYSTEM_CONSTANTS_SYSTEM_LANG]; !ok {
		r.Strings[global.SYSTEM_CONSTANTS_SYSTEM_LANG] = global.DefaultSystemLang
	}
	if _, ok := r.Strings[global.SYSTEM_CONSTANTS_MAGIC_NUMBER]; !ok {
		r.Strings[global.SYSTEM_CONSTANTS_MAGIC_NUMBER] = global.DefaultMagicNumber
	}
	if _, ok := r.Strings[global.SYSTEM_CONSTANTS_LICENSE_KEY]; !ok {
		r.Strings[global.SYSTEM_CONSTANTS_LICENSE_KEY] = ""
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
	if err := db.C(global.COLLECTION_SYSTEM_INTERNAL).FindId("counters").One(m); err != nil {
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
		case global.SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_LL,
				global.SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_UL,
			)
		case global.SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_LL,
				global.SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_UL,
			)
		case global.SYSTEM_CONSTANTS_PLACE_MAX_CREATORS:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_LL,
				global.SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_UL,
			)
		case global.SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_LL,
				global.SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_UL,
			)
		case global.SYSTEM_CONSTANTS_PLACE_MAX_LEVEL:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SYSTEM_CONSTANTS_PLACE_MAX_LEVEL_LL,
				global.SYSTEM_CONSTANTS_PLACE_MAX_LEVEL_UL,
			)
		case global.SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SYSTEM_CONSTANTS_POST_MAX_TARGETS_LL,
				global.SYSTEM_CONSTANTS_POST_MAX_TARGETS_UL,
			)
		case global.SYSTEM_CONSTANTS_POST_MAX_TARGETS:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SYSTEM_CONSTANTS_POST_MAX_TARGETS_LL,
				global.SYSTEM_CONSTANTS_POST_MAX_TARGETS_UL,
			)
		case global.SYSTEM_CONSTANTS_POST_MAX_LABELS:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SYSTEM_CONSTANTS_POST_MAX_LABELS_LL,
				global.SYSTEM_CONSTANTS_POST_MAX_LABELS_UL,
			)
		case global.SYSTEM_CONSTANTS_POST_RETRACT_TIME:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SYSTEM_CONSTANTS_POST_RETRACT_TIME_LL,
				global.SYSTEM_CONSTANTS_POST_RETRACT_TIME_UL,
			)
		case global.SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS_LL,
				global.SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS_UL,
			)
		case global.SYSTEM_CONSTANTS_CACHE_LIFETIME:
			q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
				iVal,
				global.SYSTEM_CONSTANTS_CACHE_LIFETIME_LL,
				global.SYSTEM_CONSTANTS_CACHE_LIFETIME_UL,
			)
		case global.SYSTEM_CONSTANTS_REGISTER_MODE:
			switch iVal {
			case global.REGISTER_MODE_ADMIN_ONLY, global.REGISTER_MODE_EVERYONE:
			default:
				iVal = global.REGISTER_MODE_ADMIN_ONLY
			}
			q[fmt.Sprintf("integers.%s", key)] = iVal
		}

	}
	_MongoDB.C(global.COLLECTION_SYSTEM_INTERNAL).UpdateId(
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
		case global.SYSTEM_CONSTANTS_COMPANY_NAME, global.SYSTEM_CONSTANTS_COMPANY_DESC,
			global.SYSTEM_CONSTANTS_COMPANY_LOGO, global.SYSTEM_CONSTANTS_MAGIC_NUMBER,
			global.SYSTEM_CONSTANTS_SYSTEM_LANG, global.SYSTEM_CONSTANTS_LICENSE_KEY:
			q[fmt.Sprintf("strings.%s", key)] = sVal
		}
	}
	_MongoDB.C(global.COLLECTION_SYSTEM_INTERNAL).UpdateId(
		"constants",
		bson.M{"$set": q},
	)
	sm.LoadStringConstants()
}

func (sm *SystemManager) LoadIntegerConstants() {
	iConstants := sm.GetIntegerConstants()
	// Place Constants
	global.DefaultPlaceMaxChildren = ClampInteger(
		iConstants[global.SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN],
		global.SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_LL,
		global.SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_UL,
	)
	global.DefaultPlaceMaxCreators = ClampInteger(
		iConstants[global.SYSTEM_CONSTANTS_PLACE_MAX_CREATORS],
		global.SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_LL,
		global.SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_UL,
	)
	global.DefaultPlaceMaxKeyHolders = ClampInteger(
		iConstants[global.SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS],
		global.SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_LL,
		global.SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_UL,
	)
	global.DefaultPlaceMaxLevel = ClampInteger(
		iConstants[global.SYSTEM_CONSTANTS_PLACE_MAX_LEVEL],
		global.SYSTEM_CONSTANTS_PLACE_MAX_LEVEL_LL,
		global.SYSTEM_CONSTANTS_PLACE_MAX_LEVEL_UL,
	)

	// Post Constants
	global.DefaultPostMaxAttachments = ClampInteger(
		iConstants[global.SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS],
		global.SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS_LL,
		global.SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS_UL,
	)
	global.DefaultPostMaxTargets = ClampInteger(
		iConstants[global.SYSTEM_CONSTANTS_POST_MAX_TARGETS],
		global.SYSTEM_CONSTANTS_POST_MAX_TARGETS_LL,
		global.SYSTEM_CONSTANTS_POST_MAX_TARGETS_UL,
	)
	global.DefaultPostRetractTime = uint64(ClampInteger(
		iConstants[global.SYSTEM_CONSTANTS_POST_RETRACT_TIME],
		global.SYSTEM_CONSTANTS_POST_RETRACT_TIME_LL,
		global.SYSTEM_CONSTANTS_POST_RETRACT_TIME_UL,
	))
	global.DefaultPostMaxLabels = ClampInteger(
		iConstants[global.SYSTEM_CONSTANTS_POST_MAX_LABELS],
		global.SYSTEM_CONSTANTS_POST_MAX_LABELS_LL,
		global.SYSTEM_CONSTANTS_POST_MAX_LABELS_UL,
	)

	// Account Constants
	global.DefaultAccountGrandPlaces = ClampInteger(
		iConstants[global.SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT],
		global.SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_LL,
		global.SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_UL,
	)

	// Label Constants
	global.DefaultLabelMaxMembers = ClampInteger(
		iConstants[global.SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS],
		global.SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS_LL,
		global.SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS_UL,
	)

	// Misc Constants
	global.CacheLifetime = ClampInteger(
		iConstants[global.SYSTEM_CONSTANTS_CACHE_LIFETIME],
		global.SYSTEM_CONSTANTS_CACHE_LIFETIME_LL,
		global.SYSTEM_CONSTANTS_CACHE_LIFETIME_UL,
	)

	switch iConstants[global.SYSTEM_CONSTANTS_REGISTER_MODE] {
	case global.REGISTER_MODE_ADMIN_ONLY, global.REGISTER_MODE_EVERYONE:
		global.RegisterModeAdminOnly = iConstants[global.SYSTEM_CONSTANTS_REGISTER_MODE]
	default:
		global.RegisterModeAdminOnly = global.REGISTER_MODE_ADMIN_ONLY
	}
}

func (sm *SystemManager) LoadStringConstants() {
	sConstants := sm.GetStringConstants()
	global.DefaultCompanyName = sConstants[global.SYSTEM_CONSTANTS_COMPANY_NAME]
	global.DefaultCompanyDesc = sConstants[global.SYSTEM_CONSTANTS_COMPANY_DESC]
	global.DefaultCompanyLogo = sConstants[global.SYSTEM_CONSTANTS_COMPANY_LOGO]
	global.DefaultMagicNumber = sConstants[global.SYSTEM_CONSTANTS_MAGIC_NUMBER]
}

func (sm *SystemManager) SetMessageTemplate(msgID, msgSubject, msgBody string) bool {
	if _, err := _MongoDB.C(global.COLLECTION_SYSTEM_INTERNAL).UpsertId(
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
	if err := db.C(global.COLLECTION_SYSTEM_INTERNAL).FindId("message_templates").One(&templates); err != nil {
		log.Warn(err.Error())
	}
	return templates
}

func (sm *SystemManager) RemoveMessageTemplate(msgID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_SYSTEM_INTERNAL).UpdateId(
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
		case global.SYSTEM_COUNTERS_ENABLED_ACCOUNTS, global.SYSTEM_COUNTERS_DISABLED_ACCOUNTS,
			global.SYSTEM_COUNTERS_GRAND_PLACES, global.SYSTEM_COUNTERS_LOCKED_PLACES, global.SYSTEM_COUNTERS_UNLOCKED_PLACES,
			global.SYSTEM_COUNTERS_PERSONAL_PLACES:
		default:
			delete(params, key)
		}
	}
	if _, err := db.C(global.COLLECTION_SYSTEM_INTERNAL).UpsertId(
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
	if _, err := _MongoDB.C(global.COLLECTION_SYSTEM_INTERNAL).UpsertId(
		"constants",
		bson.M{"$set": bson.M{global.SYSTEM_CONSTANTS_MODEL_VERSION: n}},
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
	if err := db.C(global.COLLECTION_SYSTEM_INTERNAL).FindId("constants").One(r); err != nil {
		return global.DefaultModelVersion
	}
	model, _ := r[global.SYSTEM_CONSTANTS_MODEL_VERSION]
	return model
}

func (sm *SystemManager) incrementCounter(params MI) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	for key := range params {
		switch key {
		case global.SYSTEM_COUNTERS_ENABLED_ACCOUNTS, global.SYSTEM_COUNTERS_DISABLED_ACCOUNTS,
			global.SYSTEM_COUNTERS_GRAND_PLACES, global.SYSTEM_COUNTERS_LOCKED_PLACES, global.SYSTEM_COUNTERS_UNLOCKED_PLACES:
		default:
			delete(params, key)
		}
	}
	if _, err := db.C(global.COLLECTION_SYSTEM_INTERNAL).UpsertId(
		"counters",
		bson.M{"$inc": params},
	); err != nil {
		log.Sugar().Info("Model::SystemManager::IncrementCounter::Error 1::", err.Error())
		return false
	}
	return true
}
