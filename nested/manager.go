package nested

import (
	"crypto/tls"
	"encoding/gob"
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/cache"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net"
	"time"
)

var (
	_Manager      *Manager
	_MongoSession *mgo.Session
	_MongoDB      *mgo.Database
	_MongoStore   *mgo.GridFS
	_Cache        *cache.Manager
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
}

// Manager is the wrapper around all the other managers
type Manager struct {
	Account       *AccountManager
	App           *AppManager
	Contact       *ContactManager
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
}

func NewManager(instanceID, mongoDSN, redisDSN string, logLevel int) (*Manager, error) {
	err := initMongoDB(mongoDSN, instanceID)
	if err != nil {
		return nil, err
	}

	err = initCache(redisDSN)
	if err != nil {
		return nil, err
	}

	_Manager = &Manager{
		Account:       newAccountManager(),
		App:           newAppManager(),
		Contact:       newContactManager(),
		File:          newFileManager(),
		Group:         newGroupManager(),
		Hook:          newHookManager(),
		Label:         newLabelManager(),
		License:       newLicenceManager(),
		Notification:  newNotificationManager(),
		Phone:         newPhoneManager(),
		Place:         newPlaceManager(),
		PlaceActivity: newPlaceActivityManager(),
		Post:          newPostManager(),
		PostActivity:  newPostActivityManager(),
		Report:        newReportManager(),
		Search:        newSearchManager(),
		Session:       newSessionManager(),
		Store:         newStoreManager(),
		System:        newSystemManager(),
		Task:          newTaskManager(),
		TaskActivity:  newTaskActivityManager(),
		TimeBucket:    newTimeBucketManager(),
		Token:         newTokenManager(),
		Verification:  newVerificationManager(),
	}

	// Set Log Level
	log.SetLevel(zapcore.Level(logLevel))

	return _Manager, nil
}
func initMongoDB(mongoDSN, instanceID string) error {
	// Initial MongoDB
	tlsConfig := new(tls.Config)
	tlsConfig.InsecureSkipVerify = true
	if dialInfo, err := mgo.ParseURL(mongoDSN); err != nil {
		log.Warn("Got error on parsing MongoDB DSN", zap.Error(err))
		return err
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
			log.Warn("Got error on dialing TLS to MongoDB", zap.Error(err))
			if mongoSession, err = mgo.Dial(mongoDSN); err != nil {
				log.Warn("Got error on dialing plain to MongoDB", zap.Error(err), zap.String("DSN", mongoDSN))
				return err
			} else {
				log.Info("MongoDB Connected")
				_MongoSession = mongoSession
			}
		} else {
			log.Info("MongoDB(TLS) Connected")
			_MongoSession = mongoSession
		}
	}

	// Set connection pool limit
	global.DbName = fmt.Sprintf("nested-%s", instanceID)
	global.StoreName = fmt.Sprintf("nested_store-%s", instanceID)
	_MongoDB = _MongoSession.DB(global.DbName)
	_MongoStore = _MongoSession.DB(global.StoreName).GridFS("fs")
	return nil
}
func initCache(redisDSN string) error {
	// Initialize Cache Redis
	if c, err := cache.New(redisDSN); err != nil {
		log.Warn("Redis Pool Connection Error", zap.Error(err))
		return err
	} else {
		_Cache = c
	}
	return nil
}

func (m *Manager) RefreshDbConnection() {
	_MongoDB.Session.Refresh()
}

func (m *Manager) Shutdown() {
	_MongoSession.Close()
}

func (m *Manager) RegisterBundle(bundleID string) {
	if _, err := _MongoDB.C(global.CollectionSystemInternal).Upsert(
		bson.M{"_id": "bundles"},
		bson.M{"$addToSet": bson.M{"bundle_ids": bundleID}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
}

func (m *Manager) GetBundles() []string {
	r := struct {
		ID        string   `bson:"_id"`
		BundleIDs []string `bson:"bundle_ids"`
	}{}
	if err := _MongoDB.C(global.CollectionSystemInternal).FindId("bundles").One(&r); err != nil {
		log.Warn("Got error", zap.Error(err))
		return []string{}
	} else {
		return r.BundleIDs
	}

}

func (m *Manager) DB() *mgo.Session {
	return _MongoSession
}

func (m *Manager) Cache() *cache.Manager {
	return _Cache
}

// ModelCheckHealth checks the whole database in a time-consuming manner
// do not use it for regular checks
func (m *Manager) ModelCheckHealth() {
	RunDoctor(nil)
}

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
	if n > global.DefaultMaxResultLimit || n <= 0 {
		p.limit = global.DefaultMaxResultLimit
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
