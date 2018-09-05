package nested

import (
    "encoding/json"
    "fmt"
    "github.com/gomodule/redigo/redis"
    "github.com/globalsign/mgo/bson"
    "log"
)

const (
    SYS_INFO_USERAPI = "userapi"
    SYS_INFO_GATEWAY = "gateway"
    SYS_INFO_MSGAPI  = "msgapi"
    SYS_INFO_STORAGE = "storage"
    SYS_INFO_ROUTER  = "router"
)

// System Keys
//  1.  message_templates
//  2.  constants
//      2.1.    model_version
//      2.2.    cache_lifetime
//      2.3.    post_max_targets
//      2.4.    post_max_attachments
//      2.5.    post_max_labels
//      2.6.    post_retract_time
//      2.7.    place_max_children
//      2.8.    place_max_keyholders
//      2.9.    place_max_creators
//      2.10.   place_max_level
//      2.11.   label_max_members
//      2.12.   register_mode
//  3.  counters
//      3.1.
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
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    r := new(SystemConstants)
    if err := db.C(COLLECTION_SYSTEM_INTERNAL).FindId("constants").One(r); err != nil {
        log.Println("Model::SystemManager::GetIntegerConstants::Error 1::", err.Error())
        return nil
    }
    if r.Integers == nil {
        r.Integers = MI{}
    }
    if r.Strings == nil {
        r.Strings = MS{}
    }

    // Place Constants
    if _, ok := r.Integers[SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN]; !ok {
        r.Integers[SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN] = DEFAULT_PLACE_MAX_CHILDREN
    }
    if _, ok := r.Integers[SYSTEM_CONSTANTS_PLACE_MAX_CREATORS]; !ok {
        r.Integers[SYSTEM_CONSTANTS_PLACE_MAX_CREATORS] = DEFAULT_PLACE_MAX_CREATORS
    }
    if _, ok := r.Integers[SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS]; !ok {
        r.Integers[SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS] = DEFAULT_PLACE_MAX_KEYHOLDERS
    }
    if _, ok := r.Integers[SYSTEM_CONSTANTS_PLACE_MAX_LEVEL]; !ok {
        r.Integers[SYSTEM_CONSTANTS_PLACE_MAX_LEVEL] = DEFAULT_PLACE_MAX_LEVEL
    }

    // Post Constants
    if _, ok := r.Integers[SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS]; !ok {
        r.Integers[SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS] = DEFAULT_POST_MAX_ATTACHMENTS
    }
    if _, ok := r.Integers[SYSTEM_CONSTANTS_POST_MAX_TARGETS]; !ok {
        r.Integers[SYSTEM_CONSTANTS_POST_MAX_TARGETS] = DEFAULT_POST_MAX_TARGETS
    }
    if _, ok := r.Integers[SYSTEM_CONSTANTS_POST_MAX_LABELS]; !ok {
        r.Integers[SYSTEM_CONSTANTS_POST_MAX_LABELS] = DEFAULT_POST_MAX_LABELS
    }
    if _, ok := r.Integers[SYSTEM_CONSTANTS_POST_RETRACT_TIME]; !ok {
        r.Integers[SYSTEM_CONSTANTS_POST_RETRACT_TIME] = int(DEFAULT_POST_RETRACT_TIME)
    }

    // Account Constants
    if _, ok := r.Integers[SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT]; !ok {
        r.Integers[SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT] = DEFAULT_ACCOUNT_GRAND_PLACES
    }

    // Label Constants
    if _, ok := r.Integers[SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS]; !ok {
        r.Integers[SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS] = DEFAULT_LABEL_MAX_MEMBERS
    }

    // Misc Constants
    if _, ok := r.Integers[SYSTEM_CONSTANTS_CACHE_LIFETIME]; !ok {
        r.Integers[SYSTEM_CONSTANTS_CACHE_LIFETIME] = CACHE_LIFETIME
    }
    if _, ok := r.Integers[SYSTEM_CONSTANTS_REGISTER_MODE]; !ok {
        r.Integers[SYSTEM_CONSTANTS_REGISTER_MODE] = REGISTER_MODE
    }

    return r.Integers
}

// GetStringConstants returns a map with string values
func (sm *SystemManager) GetStringConstants() MS {
    r := new(SystemConstants)
    if err := _MongoDB.C(COLLECTION_SYSTEM_INTERNAL).FindId("constants").One(r); err != nil {
        log.Println("Model::SystemManager::GetIntegerConstants::Error 1::", err.Error())
        return nil
    }
    if r.Integers == nil {
        r.Integers = MI{}
    }
    if r.Strings == nil {
        r.Strings = MS{}
    }
    // Company Constants
    if _, ok := r.Strings[SYSTEM_CONSTANTS_COMPANY_NAME]; !ok {
        r.Strings[SYSTEM_CONSTANTS_COMPANY_NAME] = DEFAULT_COMPANY_NAME
    }
    if _, ok := r.Strings[SYSTEM_CONSTANTS_COMPANY_DESC]; !ok {
        r.Strings[SYSTEM_CONSTANTS_COMPANY_DESC] = DEFAULT_COMPANY_DESC
    }
    if _, ok := r.Strings[SYSTEM_CONSTANTS_COMPANY_LOGO]; !ok {
        r.Strings[SYSTEM_CONSTANTS_COMPANY_LOGO] = DEFAULT_COMPANY_LOGO
    }
    if _, ok := r.Strings[SYSTEM_CONSTANTS_SYSTEM_LANG]; !ok {
        r.Strings[SYSTEM_CONSTANTS_SYSTEM_LANG] = DEFAULT_SYSTEM_LANG
    }
    if _, ok := r.Strings[SYSTEM_CONSTANTS_MAGIC_NUMBER]; !ok {
        r.Strings[SYSTEM_CONSTANTS_MAGIC_NUMBER] = DEFAULT_MAGIC_NUMBER
    }
    if _, ok := r.Strings[SYSTEM_CONSTANTS_LICENSE_KEY]; !ok {
        r.Strings[SYSTEM_CONSTANTS_LICENSE_KEY] = ""
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
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    m := MI{}
    if err := db.C(COLLECTION_SYSTEM_INTERNAL).FindId("counters").One(m); err != nil {
        _Log.Warn(err.Error())
    }
    return m
}

// SetIntegerConstants set system wide integer setting parameters, they constants until admin
// reset them again
func (sm *SystemManager) SetIntegerConstants(m M) {
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
        case SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT:
            q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
                iVal,
                SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_LL,
                SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_UL,
            )
        case SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN:
            q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
                iVal,
                SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_LL,
                SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_UL,
            )
        case SYSTEM_CONSTANTS_PLACE_MAX_CREATORS:
            q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
                iVal,
                SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_LL,
                SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_UL,
            )
        case SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS:
            q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
                iVal,
                SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_LL,
                SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_UL,
            )
        case SYSTEM_CONSTANTS_PLACE_MAX_LEVEL:
            q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
                iVal,
                SYSTEM_CONSTANTS_PLACE_MAX_LEVEL_LL,
                SYSTEM_CONSTANTS_PLACE_MAX_LEVEL_UL,
            )
        case SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS:
            q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
                iVal,
                SYSTEM_CONSTANTS_POST_MAX_TARGETS_LL,
                SYSTEM_CONSTANTS_POST_MAX_TARGETS_UL,
            )
        case SYSTEM_CONSTANTS_POST_MAX_TARGETS:
            q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
                iVal,
                SYSTEM_CONSTANTS_POST_MAX_TARGETS_LL,
                SYSTEM_CONSTANTS_POST_MAX_TARGETS_UL,
            )
        case SYSTEM_CONSTANTS_POST_MAX_LABELS:
            q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
                iVal,
                SYSTEM_CONSTANTS_POST_MAX_LABELS_LL,
                SYSTEM_CONSTANTS_POST_MAX_LABELS_UL,
            )
        case SYSTEM_CONSTANTS_POST_RETRACT_TIME:
            q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
                iVal,
                SYSTEM_CONSTANTS_POST_RETRACT_TIME_LL,
                SYSTEM_CONSTANTS_POST_RETRACT_TIME_UL,
            )
        case SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS:
            q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
                iVal,
                SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS_LL,
                SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS_UL,
            )
        case SYSTEM_CONSTANTS_CACHE_LIFETIME:
            q[fmt.Sprintf("integers.%s", key)] = ClampInteger(
                iVal,
                SYSTEM_CONSTANTS_CACHE_LIFETIME_LL,
                SYSTEM_CONSTANTS_CACHE_LIFETIME_UL,
            )
        case SYSTEM_CONSTANTS_REGISTER_MODE:
            switch iVal {
            case REGISTER_MODE_ADMIN_ONLY, REGISTER_MODE_EVERYONE:
            default:
                iVal = REGISTER_MODE_ADMIN_ONLY
            }
            q[fmt.Sprintf("integers.%s", key)] = iVal
        }

    }
    _MongoDB.C(COLLECTION_SYSTEM_INTERNAL).UpdateId(
        "constants",
        bson.M{"$set": q},
    )
    sm.LoadIntegerConstants()
}

// SetStringConstants set system wide string setting parameters, they constants until admin
// reset them again
func (sm *SystemManager) SetStringConstants(m M) {
    q := bson.M{}
    for key, v := range m {
        var sVal string
        switch v.(type) {
        case string:
            sVal = v.(string)
        }
        switch key {
        case SYSTEM_CONSTANTS_COMPANY_NAME, SYSTEM_CONSTANTS_COMPANY_DESC,
            SYSTEM_CONSTANTS_COMPANY_LOGO, SYSTEM_CONSTANTS_MAGIC_NUMBER,
            SYSTEM_CONSTANTS_SYSTEM_LANG, SYSTEM_CONSTANTS_LICENSE_KEY:
            q[fmt.Sprintf("strings.%s", key)] = sVal
        }
    }
    _MongoDB.C(COLLECTION_SYSTEM_INTERNAL).UpdateId(
        "constants",
        bson.M{"$set": q},
    )
    sm.LoadStringConstants()
}

func (sm *SystemManager) LoadIntegerConstants() {
    iConstants := sm.GetIntegerConstants()
    // Place Constants
    DEFAULT_PLACE_MAX_CHILDREN = ClampInteger(
        iConstants[SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN],
        SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_LL,
        SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_UL,
    )
    DEFAULT_PLACE_MAX_CREATORS = ClampInteger(
        iConstants[SYSTEM_CONSTANTS_PLACE_MAX_CREATORS],
        SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_LL,
        SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_UL,
    )
    DEFAULT_PLACE_MAX_KEYHOLDERS = ClampInteger(
        iConstants[SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS],
        SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_LL,
        SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_UL,
    )
    DEFAULT_PLACE_MAX_LEVEL = ClampInteger(
        iConstants[SYSTEM_CONSTANTS_PLACE_MAX_LEVEL],
        SYSTEM_CONSTANTS_PLACE_MAX_LEVEL_LL,
        SYSTEM_CONSTANTS_PLACE_MAX_LEVEL_UL,
    )

    // Post Constants
    DEFAULT_POST_MAX_ATTACHMENTS = ClampInteger(
        iConstants[SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS],
        SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS_LL,
        SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS_UL,
    )
    DEFAULT_POST_MAX_TARGETS = ClampInteger(
        iConstants[SYSTEM_CONSTANTS_POST_MAX_TARGETS],
        SYSTEM_CONSTANTS_POST_MAX_TARGETS_LL,
        SYSTEM_CONSTANTS_POST_MAX_TARGETS_UL,
    )
    DEFAULT_POST_RETRACT_TIME = uint64(ClampInteger(
        iConstants[SYSTEM_CONSTANTS_POST_RETRACT_TIME],
        SYSTEM_CONSTANTS_POST_RETRACT_TIME_LL,
        SYSTEM_CONSTANTS_POST_RETRACT_TIME_UL,
    ))
    DEFAULT_POST_MAX_LABELS = ClampInteger(
        iConstants[SYSTEM_CONSTANTS_POST_MAX_LABELS],
        SYSTEM_CONSTANTS_POST_MAX_LABELS_LL,
        SYSTEM_CONSTANTS_POST_MAX_LABELS_UL,
    )

    // Account Constants
    DEFAULT_ACCOUNT_GRAND_PLACES = ClampInteger(
        iConstants[SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT],
        SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_LL,
        SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_UL,
    )

    // Label Constants
    DEFAULT_LABEL_MAX_MEMBERS = ClampInteger(
        iConstants[SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS],
        SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS_LL,
        SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS_UL,
    )

    // Misc Constants
    CACHE_LIFETIME = ClampInteger(
        iConstants[SYSTEM_CONSTANTS_CACHE_LIFETIME],
        SYSTEM_CONSTANTS_CACHE_LIFETIME_LL,
        SYSTEM_CONSTANTS_CACHE_LIFETIME_UL,
    )

    switch iConstants[SYSTEM_CONSTANTS_REGISTER_MODE] {
    case REGISTER_MODE_ADMIN_ONLY, REGISTER_MODE_EVERYONE:
        REGISTER_MODE = iConstants[SYSTEM_CONSTANTS_REGISTER_MODE]
    default:
        REGISTER_MODE = REGISTER_MODE_ADMIN_ONLY
    }
}

func (sm *SystemManager) LoadStringConstants() {
    sConstants := sm.GetStringConstants()
    DEFAULT_COMPANY_NAME = sConstants[SYSTEM_CONSTANTS_COMPANY_NAME]
    DEFAULT_COMPANY_DESC = sConstants[SYSTEM_CONSTANTS_COMPANY_DESC]
    DEFAULT_COMPANY_LOGO = sConstants[SYSTEM_CONSTANTS_COMPANY_LOGO]
    DEFAULT_MAGIC_NUMBER = sConstants[SYSTEM_CONSTANTS_MAGIC_NUMBER]
}

func (sm *SystemManager) SetMessageTemplate(msgID, msgSubject, msgBody string) bool {
    // _funcName
    if _, err := _MongoDB.C(COLLECTION_SYSTEM_INTERNAL).UpsertId(
        "message_templates",
        bson.M{"$set": bson.M{
            msgID: bson.M{
                "subject": msgSubject,
                "body":    msgBody,
            }}},
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

func (sm *SystemManager) GetMessageTemplates() map[string]MessageTemplate {
    // _funcName

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    templates := make(map[string]MessageTemplate)
    if err := db.C(COLLECTION_SYSTEM_INTERNAL).FindId("message_templates").One(&templates); err != nil {
        _Log.Warn(err.Error())
    }
    return templates
}

func (sm *SystemManager) RemoveMessageTemplate(msgID string) {
    // _funcName

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_SYSTEM_INTERNAL).UpdateId(
        "message_templates",
        bson.M{
            "$unset": bson.M{msgID: ""},
        }); err != nil {
        _Log.Warn(err.Error())
    }

}

func (sm *SystemManager) SetSystemInfo(key, bundleID string, info M) bool {
    // _funcName
    switch key {
    case SYS_INFO_GATEWAY, SYS_INFO_MSGAPI, SYS_INFO_ROUTER,
        SYS_INFO_STORAGE, SYS_INFO_USERAPI:
    default:
        return false
    }
    keyID := fmt.Sprintf("sysinfo.%s", key)
    c := _Cache.getConn()
    defer c.Close()
    if jsonInfo, err := json.Marshal(info); err != nil {
        _Log.Warn(err.Error())
    } else {
        c.Do("HSET", keyID, bundleID, jsonInfo)
    }
    return true
}

func (sm *SystemManager) GetSystemInfo(key string) MS {
    // _funcName

    keyID := fmt.Sprintf("sysinfo.%s", key)
    c := _Cache.getConn()
    defer c.Close()
    info := MS{}
    if m, err := redis.StringMap(c.Do("HGETALL", keyID)); err != nil {
        _Log.Warn(err.Error())
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
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    for key := range params {
        switch key {
        case SYSTEM_COUNTERS_ENABLED_ACCOUNTS, SYSTEM_COUNTERS_DISABLED_ACCOUNTS,
            SYSTEM_COUNTERS_GRAND_PLACES, SYSTEM_COUNTERS_LOCKED_PLACES, SYSTEM_COUNTERS_UNLOCKED_PLACES,
            SYSTEM_COUNTERS_PERSONAL_PLACES:
        default:
            delete(params, key)
        }
    }
    if _, err := db.C(COLLECTION_SYSTEM_INTERNAL).UpsertId(
        "counters",
        bson.M{"$set": params},
    ); err != nil {
        log.Println("Model::SystemManager::SetCounter::Error 1::", err.Error())
        return false
    }
    return true
}

// Private methods
func (sm *SystemManager) setDataModelVersion(n int) bool {
    if _, err := _MongoDB.C(COLLECTION_SYSTEM_INTERNAL).UpsertId(
        "constants",
        bson.M{"$set": bson.M{SYSTEM_CONSTANTS_MODEL_VERSION: n}},
    ); err != nil {
        log.Println("Model::SystemManager::SetDataModelVersion::Error 1::", err.Error())
        return false
    }
    return true
}

func (sm *SystemManager) getDataModelVersion() int {
    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    r := MI{}
    if err := db.C(COLLECTION_SYSTEM_INTERNAL).FindId("constants").One(r); err != nil {
        return DEFAULT_MODEL_VERSION
    }
    model, _ := r[SYSTEM_CONSTANTS_MODEL_VERSION]
    return model
}

func (sm *SystemManager) incrementCounter(params MI) bool {
    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    for key := range params {
        switch key {
        case SYSTEM_COUNTERS_ENABLED_ACCOUNTS, SYSTEM_COUNTERS_DISABLED_ACCOUNTS,
            SYSTEM_COUNTERS_GRAND_PLACES, SYSTEM_COUNTERS_LOCKED_PLACES, SYSTEM_COUNTERS_UNLOCKED_PLACES:
        default:
            delete(params, key)
        }
    }
    if _, err := db.C(COLLECTION_SYSTEM_INTERNAL).UpsertId(
        "counters",
        bson.M{"$inc": params},
    ); err != nil {
        log.Println("Model::SystemManager::IncrementCounter::Error 1::", err.Error())
        return false
    }
    return true
}


