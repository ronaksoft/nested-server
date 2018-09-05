package nested

import (
    "bytes"
    "encoding/gob"
    "fmt"

    "github.com/globalsign/mgo/bson"
    "github.com/gomodule/redigo/redis"
)

type Session struct {
    ID            bson.ObjectId   `json:"_id" bson:"_id"`
    SessionSecret string          `json:"_ss" bson:"_ss,omitempty"`
    CreatedOn     uint64          `json:"created_on" bson:"created_on"`
    LastAccess    uint64          `json:"last_access" bson:"last_access"`
    LastUpdate    uint64          `json:"last_update" bson:"last_update"`
    Expired       bool            `json:"expired" bson:"expired"`
    Security      SessionSecurity `json:"security" bson:"security"`
    AccountID     string          `json:"uid" bson:"uid,omitempty"`
    DeviceID      string          `json:"_did,omitempty" bson:"_did,omitempty"`
    DeviceToken   string          `json:"_dt,omitempty" bson:"_dt,omitempty"`
    DeviceOS      string          `json:"_os,omitempty" bson:"_os,omitempty"`
    ClientID      string          `json:"_cid" bson:"_cid"`
    ClientVersion int             `json:"_cver" bson:"_cver"`
}
type SessionSecurity struct {
    CreatorIP string `json:"creator_ip" bson:"creator_ip"`
    LastIP    string `json:"last_ip" bson:"last_ip"`
    UserAgent string `json:"ua" bson:"ua"`
}

// Session Manager and Methods
type SessionManager struct{}

func NewSessionManager() *SessionManager {
    return new(SessionManager)
}

func (sm *SessionManager) readFromCache(sessionID bson.ObjectId) *Session {
    session := new(Session)
    c := _Cache.Pool.Get()
    defer c.Close()
    keyID := fmt.Sprintf("session:gob:%s", sessionID.Hex())
    if gobSession, err := redis.Bytes(c.Do("GET", keyID)); err != nil {
        if err := _MongoDB.C(COLLECTION_SESSIONS).FindId(sessionID).One(session); err != nil {
            _Log.Warn(err.Error())
            return nil
        }
        gobSession := new(bytes.Buffer)
        if err := gob.NewEncoder(gobSession).Encode(session); err == nil {
            c.Do("SETEX", keyID, CACHE_LIFETIME, gobSession.Bytes())
        }
        return session
    } else if err := gob.NewDecoder(bytes.NewBuffer(gobSession)).Decode(session); err == nil {
        return session
    }
    return nil
}

func (sm *SessionManager) updateCache(sessionID bson.ObjectId) bool {
    session := new(Session)
    c := _Cache.Pool.Get()
    defer c.Close()
    keyID := fmt.Sprintf("session:gob:%s", sessionID.Hex())
    if err := _MongoDB.C(COLLECTION_SESSIONS).FindId(sessionID).One(session); err != nil {
        _Log.Warn(err.Error())
        c.Do("DEL", keyID)
        return false
    }
    gobSession := new(bytes.Buffer)
    if err := gob.NewEncoder(gobSession).Encode(session); err != nil {
        return false
    } else {
        c.Do("SETEX", keyID, CACHE_LIFETIME, gobSession.Bytes())
    }
    return true
}

// Create
// creates a new session object in database and returns its session key
// if anything wrong happens error will be set appropriately
func (sm *SessionManager) Create(in MS) (bson.ObjectId, error) {
    // _funcName
    //
    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    sk := bson.NewObjectId()
    creatorIP := in["ip"]
    userAgent := in["ua"]

    // Increment Counters
    _Manager.Report.CountSessionLogin()

    ts := Timestamp()
    if err := db.C(COLLECTION_SESSIONS).Insert(
        Session{
            ID:         sk,
            CreatedOn:  ts,
            LastUpdate: ts,
            LastAccess: ts,
            Security: SessionSecurity{
                CreatorIP: creatorIP,
                LastIP:    creatorIP,
                UserAgent: userAgent,
            },
            Expired: false,
        },
    ); err != nil {
        _Log.Warn(err.Error())
        return "", err
    }
    return sk, nil
}

// Expire expires the session and this session identified by sk will not be valid any more
func (sm *SessionManager) Expire(sk bson.ObjectId) {
    //
    // removed LOG Function
    defer _Manager.Session.updateCache(sk)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    db.C(COLLECTION_SESSIONS).UpdateId(sk, bson.M{"$set": bson.M{"expired": true}})
}

// GetByID return Session by sessionID
func (sm *SessionManager) GetByID(sessionID bson.ObjectId) (s *Session) {
    //
    // removed LOG Function

    return _Manager.Session.readFromCache(sessionID)
}

// GetByUser
// returns an array of active sessions of accountID
func (sm *SessionManager) GetByUser(accountID string, pg Pagination) []Session {
    // _funcName
    //
    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    s := make([]Session, 0)
    if err := db.C(COLLECTION_SESSIONS).Find(bson.M{
        "uid":     accountID,
        "expired": false,
    }).Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&s); err != nil {
        _Log.Warn(err.Error())
    }
    return s

}

// GetAccount
// returns Account for the session identified by SessionKey and SessionSecret
func (sm *SessionManager) GetAccount(sk bson.ObjectId) *Account {
    // _funcName
    //
    // removed LOG Function

    session := _Manager.Session.GetByID(sk)
    if session.AccountID != "" {
        account := _Manager.Account.GetByID(session.AccountID, nil)
        return account
    }
    return nil
}

// Set
// set key-values in session identified by SessionKey(sk)
// if everything was ok it return TRUE otherwise returns FALSE
func (sm *SessionManager) Set(sk bson.ObjectId, v bson.M) bool {
    // _funcName
    //
    // removed LOG Function
    defer _Manager.Session.updateCache(sk)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_SESSIONS).Update(
        bson.M{
            "_id":     sk,
            "expired": false,
        },
        bson.M{"$set": v},
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

// UpdateLastAccess updates the session document with the last access of the Account
func (sm *SessionManager) UpdateLastAccess(sk bson.ObjectId) bool {
    // _funcName
    //
    // removed LOG Function
    defer _Manager.Session.updateCache(sk)

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_SESSIONS).Update(
        bson.M{
            "_id":     sk,
            "expired": false,
        },
        bson.M{"$set": bson.M{"last_access": Timestamp()}},
    ); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

// Verify
// verifies if the SessionKey(sk) and SessionSecret(ss) are matched and the session
// with these keys are exists and valid
func (sm *SessionManager) Verify(sk bson.ObjectId, ss string) (r bool) {
    // _funcName
    //
    // removed LOG Function

    if session := _Manager.Session.GetByID(sk); session == nil {
        return false
    } else if session.Expired || session.SessionSecret != ss {
        return false
    }
    return true
}

/*
    Session
 */
// Login
func (s *Session) Login() {
    // _funcName
    //
    // removed LOG Function

    v := bson.M{
        "uid":   s.AccountID,
        "_did":  s.DeviceID,
        "_dt":   s.DeviceToken,
        "_ss":   s.SessionSecret,
        "_cver": s.ClientVersion,
        "_cid":  s.ClientID,
    }
    _Manager.Session.Set(s.ID, v)
}

// CloseOtherActives deletes all other actives sessions of the user
func (s *Session) CloseOtherActives() {
    //
    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    db.C(COLLECTION_SESSIONS).RemoveAll(bson.M{
        "uid": s.AccountID,
        "_id": bson.M{"$ne": s.ID},
    })

}
