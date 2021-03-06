package nested

import (
	"crypto/tls"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	_Log          *zap.Logger
	_LogLevel     zap.AtomicLevel
	_Manager      *Manager
	_MongoSession *mgo.Session
	_MongoDB      *mgo.Database
	_MongoStore   *mgo.GridFS
	_Cache        *CacheManager
)

type (
	Picture struct {
		Original UniversalID `json:"org" bson:"org"`
		Preview  UniversalID `json:"pre" bson:"pre"`
		X128     UniversalID `json:"x128" bson:"x128"`
		X64      UniversalID `json:"x64" bson:"x64"`
		X32      UniversalID `json:"x32" bson:"x32"`
	}
	UniversalID string

	MS map[string]string
	MI map[string]int
)

func init() {
	gob.Register(Task{})
	gob.Register(Post{})
	gob.Register(Comment{})
	gob.Register(Account{})
	gob.Register(Place{})
	gob.Register(License{})

	// Initialize Logger
	_LogLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	config := zap.NewProductionConfig()
	config.Encoding = "console"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.Level = _LogLevel
	if v, err := config.Build(); err != nil {
		os.Exit(1)
	} else {
		_Log = v
	}

}

// Manager
type Manager struct {
	Account       *AccountManager
	App           *AppManager
	Cache         *CacheManager
	Contact       *ContactManager
	Device        *DeviceManager
	File          *FileManager
	Group         *GroupManager
	Hook          *HookManager
	Label         *LabelManager
	License       *LicenseManager
	Notification  *NotificationManager
	Phone         *PhoneManager
	Place         *PlaceManager
	PlaceActivity *PlaceActivityManager
	Post          *PostManager
	PostActivity  *PostActivityManager
	Report        *ReportManager
	Search        *SearchManager
	Session       *SessionManager
	Store         *StoreManager
	System        *SystemManager
	Task          *TaskManager
	TaskActivity  *TaskActivityManager
	TimeBucket    *TimeBucketManager
	Token         *TokenManager
	Verification  *VerificationManager
	Websocket     *WebsocketManager
}

func NewManager(instanceID, mongoDSN, redisDSN string, logLevel int) (*Manager, error) {
	// Initial MongoDB
	tlsConfig := new(tls.Config)
	tlsConfig.InsecureSkipVerify = true
	if dialInfo, err := mgo.ParseURL(mongoDSN); err != nil {
		_Log.Warn(err.Error())
		return nil, err
	} else {
		dialInfo.Timeout = 5 * time.Second
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			if conn, err := tls.Dial("tcp", addr.String(), tlsConfig); err != nil {
				return conn, err
			} else {
				return conn, nil
			}
		}
		if mongoSession, err := mgo.DialWithInfo(dialInfo); err != nil {
			_Log.Warn(err.Error())
			if mongoSession, err = mgo.Dial(mongoDSN); err != nil {
				_Log.Warn(err.Error(), zap.String("DSN", mongoDSN))
				return nil, err
			} else {
				_Log.Info("Model::NewManager::MongoDB Connected")
				_MongoSession = mongoSession
			}
		} else {
			_Log.Info("Model::NewManager::MongoDB(TLS) Connected")
			_MongoSession = mongoSession
		}
	}

	// Set connection pool limit
	DB_NAME = fmt.Sprintf("nested-%s", instanceID)
	STORE_NAME = fmt.Sprintf("nested_store-%s", instanceID)
	_MongoDB = _MongoSession.DB(DB_NAME)
	_MongoStore = _MongoSession.DB(STORE_NAME).GridFS("fs")

	// Initialize Cache Redis
	if c, err := NewCacheManager(redisDSN); err != nil {
		_Log.Info("Redis Pool Connection Error")
		return nil, err
	} else {
		_Cache = c
	}

	_Manager = new(Manager)
	_Manager.Account = NewAccountManager()
	_Manager.App = NewAppManager()
	_Manager.Cache = _Cache
	_Manager.Contact = NewContactManager()
	_Manager.Device = NewDeviceManager()
	_Manager.File = NewFileManager()
	_Manager.Group = NewGroupManager()
	_Manager.Hook = NewHookManager()
	_Manager.Label = NewLabelManager()
	_Manager.License = NewLicenceManager()
	_Manager.Notification = NewNotificationManager()
	_Manager.Phone = NewPhoneManager()
	_Manager.Place = NewPlaceManager()
	_Manager.PlaceActivity = NewPlaceActivityManager()
	_Manager.Post = NewPostManager()
	_Manager.PostActivity = NewPostActivityManager()
	_Manager.Report = NewReportManager()
	_Manager.Search = NewSearchManager()
	_Manager.Session = NewSessionManager()
	_Manager.Store = NewStoreManager()
	_Manager.System = NewSystemManager()
	_Manager.Task = NewTaskManager()
	_Manager.TaskActivity = NewTaskActivityManager()
	_Manager.TimeBucket = NewTimeBucketManager()
	_Manager.Token = NewTokenManager()
	_Manager.Verification = NewVerificationManager()
	_Manager.Websocket = NewWebsocketManager()

	// Load the system constants
	_Manager.System.LoadIntegerConstants()
	_Manager.System.LoadStringConstants()
	_Manager.License.Load()

	// Set Log Level
	_Manager.SetLogLevel(logLevel)

	return _Manager, nil
}

func (m *Manager) RefreshDbConnection() {
	_MongoDB.Session.Refresh()
}

func (m *Manager) Shutdown() {
	_MongoSession.Close()
	_Log.Sync()
}

func (m *Manager) SetLogLevel(level int) {
	_LogLevel.SetLevel(zapcore.Level(level))
}

func (m *Manager) RegisterBundle(bundleID string) {
	if _, err := _MongoDB.C(COLLECTION_SYSTEM_INTERNAL).Upsert(
		bson.M{"_id": "bundles"},
		bson.M{"$addToSet": bson.M{"bundle_ids": bundleID}},
	); err != nil {
		_Log.Warn(err.Error())
	}
}

func (m *Manager) GetBundles() []string {
	r := struct {
		ID        string   `bson:"_id"`
		BundleIDs []string `bson:"bundle_ids"`
	}{}
	if err := _MongoDB.C(COLLECTION_SYSTEM_INTERNAL).FindId("bundles").One(&r); err != nil {
		_Log.Warn(err.Error())
		return []string{}
	} else {
		return r.BundleIDs
	}

}

// ModelCheckHealth checks the whole database in a time-consuming manner
// do not use it for regular checks
func (m *Manager) ModelCheckHealth() {
	RunDoctor(nil)
}

// Pagination
type Pagination struct {
	skip   int
	limit  int
	After  int64
	Before int64
}

func NewPagination(skip, limit int, after, before int64) Pagination {
	p := Pagination{}
	p.SetSkip(skip)
	p.SetLimit(limit)
	p.After = after
	p.Before = before
	return p
}
func (p *Pagination) Reset() {
	p.SetSkip(0).SetLimit(0)
	p.After = 0
	p.Before = 0
}
func (p *Pagination) AddSkip(n int) {
	p.skip += n
}
func (p *Pagination) SetSkip(n int) *Pagination {
	if n >= 0 {
		p.skip = n
	}
	return p
}
func (p *Pagination) SetLimit(n int) *Pagination {
	if n > DEFAULT_MAX_RESULT_LIMIT || n <= 0 {
		p.limit = DEFAULT_MAX_RESULT_LIMIT
	} else {
		p.limit = n
	}
	return p
}
func (p *Pagination) GetSkip() int {
	return p.skip
}
func (p *Pagination) GetLimit() int {
	return p.limit
}
func (p *Pagination) FillQuery(q bson.M, sortItem, sortDir string) (bson.M, string) {
	if p.After > 0 && p.Before > 0 {
		switch x := q["$and"].(type) {
		case []bson.M:
			q["$and"] = append(x, bson.M{sortItem: bson.M{"$gt": p.After}}, bson.M{sortItem: bson.M{"$lt": p.Before}})
		default:
			q["$and"] = []bson.M{
				{sortItem: bson.M{"$gt": p.After}}, {sortItem: bson.M{"$lt": p.Before}},
			}
		}
	} else if p.After > 0 {
		sortDir = sortItem
		q[sortItem] = bson.M{"$gt": p.After}
	} else if p.Before > 0 {
		q[sortItem] = bson.M{"$lt": p.Before}
	}
	return q, sortDir
}

// Generic Map
type M map[string]interface{}

func (m M) KeysToArray() []string {
	arr := make([]string, 0, len(m))
	for k := range m {
		arr = append(arr, k)
	}
	return arr
}
func (m M) ValuesToArray() []interface{} {
	arr := make([]interface{}, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

// Boolean MAp
type MB map[string]bool

func (m MB) AddKeys(keys ...[]string) {
	for _, arr := range keys {
		for _, key := range arr {
			m[key] = true
		}
	}
}
func (m MB) KeysToArray() []string {
	arr := make([]string, 0, len(m))
	for k := range m {
		arr = append(arr, k)
	}
	return arr
}
