package ronak

import (
    "sync"
    "encoding/gob"
    "github.com/gomodule/redigo/redis"
    "fmt"
)

func init() {
    gob.Register(SimpleSession{})
}

// Manager
// This is to manage sessions of the system.
type SessionManager struct {
    sync.Mutex
    sessions   map[int64]Session
    redisCache *RedisCache
    name       string
}

// NewSessionManager
// This function returns a pointer to SessionManager object. If redisCache was 'nil' then NewAdvancedSession
// function returns nil, hence developer cannot create AdvancedSessions. AdvancedSessions could be enabled
// later by calling SetRedisCache and passing a non-nil value.
func NewSessionManager(redisCache *RedisCache, name string) *SessionManager {
    m := new(SessionManager)
    m.name = name
    m.sessions = make(map[int64]Session)
    m.redisCache = redisCache
    return m
}

func (s *SessionManager) SetRedisCache(redisCache *RedisCache) {
    s.redisCache = redisCache
}

func (s *SessionManager) NewSimpleSession(sessionID int64) *SimpleSession {
    s.Lock()
    defer s.Unlock()

    if session, ok := s.sessions[sessionID]; ok {
        return session.(*SimpleSession)
    } else {
        session = NewSimpleSession(sessionID)
        s.sessions[sessionID] = session
        return session.(*SimpleSession)
    }
}

// NewAdvancedSession
// Returns pointer to an AdvancedSession, make sure partitionKey could be any of:
// int, int32, int64, uint, uint32, uint64, string
func (s *SessionManager) NewAdvancedSession(sessionID int64) *AdvancedSession {
    s.Lock()
    defer s.Unlock()

    if s.redisCache == nil {
        return nil
    }

    if session, ok := s.sessions[sessionID]; ok {
        return session.(*AdvancedSession)
    } else {
        session = NewAdvancedSession(s.redisCache, s.name, sessionID)
        s.sessions[sessionID] = session
        return session.(*AdvancedSession)
    }
}

func (s *SessionManager) GetSimpleSession(sessionID int64) *SimpleSession {
    s.Lock()
    defer s.Unlock()

    if session, ok := s.sessions[sessionID]; ok {
        return session.(*SimpleSession)
    }
    return nil
}

func (s *SessionManager) GetAdvancedSession(sessionID int64) *AdvancedSession {
    s.Lock()
    defer s.Unlock()

    if session, ok := s.sessions[sessionID]; ok {
        return session.(*AdvancedSession)
    }
    return nil
}

func (s *SessionManager) RemoveSession(sessionID int64) {
    s.Lock()
    defer s.Unlock()

    delete(s.sessions, sessionID)
}

func (s *SessionManager) SaveSession(sessionID int64) error {
    // TODO:: implementation
    // This is only for compatibility, if you need to save sessions use cass session
    return nil
}

// Session Interface
type Session interface {
    KeyExists(key string) bool
    Get(key string) (interface{}, bool)
    Set(key string, value interface{})
    Inc(key string, value int) int
    serialize() []byte
}

// SimpleSession implements Session interface
type SimpleSession struct {
    ID int64
    kv *SimpleMap
}

func NewSimpleSession(sessionID int64) *SimpleSession {
    s := new(SimpleSession)
    s.ID = sessionID
    s.kv = NewSimpleMap(0)
    return s
}

func (s *SimpleSession) KeyExists(key string) bool {
    _, ok := s.kv.Get(key)
    return ok
}

func (s *SimpleSession) Get(key string) (interface{}, bool) {
    return s.kv.Get(key)
}

func (s *SimpleSession) Set(key string, value interface{}) {
    s.kv.Set(key, value)
}

func (s *SimpleSession) Inc(key string, value int) int {
    if v, ok := s.kv.GetInt(key); !ok {
        s.kv.Set(key, value)
        return value
    } else {
        v = v + value
        s.kv.Set(key, v)
        return v
    }
}

func (s *SimpleSession) serialize() []byte {
    return []byte{}
}

// AdvancedSession implements Session interface with more capabilities than SimpleSession
// SharedKeys are some keys which are stored in a RedisCache backend and they could be changed
// atomic. When the session required to be accessed from different services (one or
// multiple devices) the developer may use of SharedKeys
// Having a Redis server is essential otherwise use SimpleSession which handles all the keys
// locally in the memory of the server running this code.
type AdvancedSession struct {
    ID           int64
    redisCache   *RedisCache
    kv           *SimpleMap
    partitionKey string
}

func NewAdvancedSession(redisCache *RedisCache, partitionKey interface{}, sessionID int64) *AdvancedSession {
    s := new(AdvancedSession)
    s.redisCache = redisCache
    s.ID = sessionID
    s.partitionKey = "SESSION"
    switch partitionKey.(type) {
    case int, int32, int64, uint, uint64, uint32:
        s.partitionKey = fmt.Sprintf("{%d}.%d", partitionKey, sessionID)
    case string:
        s.partitionKey = fmt.Sprintf("{%s}.%d", partitionKey, sessionID)
    default:
        s.partitionKey = fmt.Sprintf("{SESSION}.%d", sessionID)
    }
    return s
}

func (s *AdvancedSession) KeyExists(key string) bool {
    _, ok := s.kv.Get(key)
    return ok
}

func (s *AdvancedSession) Get(key string) (interface{}, bool) {
    return s.kv.Get(key)
}

func (s *AdvancedSession) Set(key string, value interface{}) {
    s.kv.Set(key, value)
}

func (s *AdvancedSession) Inc(key string, value int) int {
    if v, ok := s.kv.GetInt(key); !ok {
        s.kv.Set(key, value)
        return value
    } else {
        v = v + value
        s.kv.Set(key, v)
        return v
    }
}

func (s *AdvancedSession) serialize() []byte {
    return []byte{}
}

func (s *AdvancedSession) GetSharedKey(key string) (v interface{}, err error) {
    c := s.redisCache.GetConn()
    defer c.Close()

    v, err = c.Get(fmt.Sprintf("%s.%s", s.partitionKey, key))
    return
}

func (s *AdvancedSession) SetSharedKey(key string, value interface{}) error {
    c := s.redisCache.GetConn()
    defer c.Close()

    _, err := c.Set(fmt.Sprintf("%s.%s", s.partitionKey, key), value)
    return err

}

func (s *AdvancedSession) IncSharedKey(key string, incValue int64) (v int64, err error) {
    c := s.redisCache.GetConn()
    defer c.Close()

    v, err = redis.Int64(c.IncBy(fmt.Sprintf("%s.%s", s.partitionKey, key), incValue))
    return
}

func (s *AdvancedSession) SetSharedMapKey(mapName, fieldKey string, fieldValue interface{}) error {
    c := s.redisCache.GetConn()
    defer c.Close()

    _, err := c.HSet(fmt.Sprintf("%s.MAP.%s", s.partitionKey, mapName), fieldKey, fieldValue)
    return err
}

func (s *AdvancedSession) GetSharedMapKey(mapName, fieldKey string) (v interface{}, err error) {
    c := s.redisCache.GetConn()
    defer c.Close()

    v, err = c.HGet(fmt.Sprintf("%s.MAP.%s", s.partitionKey, mapName), fieldKey)
    return
}

func (s *AdvancedSession) GetSharedMapAllKeys(mapName string) (fieldKeys []string, err error) {
    c := s.redisCache.GetConn()
    defer c.Close()

    fieldKeys, err = redis.Strings(c.HKeys(fmt.Sprintf("%s.MAP.%s", s.partitionKey, mapName)))
    return
}
