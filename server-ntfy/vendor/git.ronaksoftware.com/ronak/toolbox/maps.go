package ronak

import (
    "sync"
    "fmt"
    "github.com/gomodule/redigo/redis"
)

/*
    Creation Time: 2018 - Apr - 07
    Created by:  Ehsan N. Moosa (ehsan)
    Maintainers:
        1.  Ehsan N. Moosa (ehsan)
    Auditor: Ehsan N. Moosa
    Copyright Ronak Software Group 2018
*/

type Map interface {
    Set(key string, value interface{})
    Get(key string) (interface{}, bool)
}

type Int64Map interface {
    Set(key int64, value interface{})
    Get(key int64) (interface{}, bool)
}

// SimpleMap
type SimpleMap struct {
    sync.Mutex
    keyPairs map[string]interface{}
}

func NewSimpleMap(initialCapacity int) *SimpleMap {
    k := new(SimpleMap)
    if initialCapacity == 0 {
        initialCapacity = 100
    }
    k.keyPairs = make(map[string]interface{}, initialCapacity)
    return k
}

func (m *SimpleMap) Set(key string, value interface{}) {
    m.Lock()
    defer m.Unlock()

    m.keyPairs[key] = value
}

func (m *SimpleMap) get(key string) (interface{}, bool) {
    m.Lock()
    defer m.Unlock()

    if v, ok := m.keyPairs[key]; ok {
        return v, ok
    } else {
        return v, ok
    }
}

func (m *SimpleMap) Get(key string) (interface{}, bool) {
    return m.get(key)
}

func (m *SimpleMap) GetInt(key string) (int, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int), ok
    }
}

func (m *SimpleMap) GetInt32(key string) (int32, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int32), ok
    }
}

func (m *SimpleMap) GetInt64(key string) (int64, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int64), ok
    }
}

func (m *SimpleMap) GetUInt(key string) (uint, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(uint), ok
    }
}

func (m *SimpleMap) GetUInt32(key string) (uint32, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(uint32), ok
    }
}

func (m *SimpleMap) GetUInt64(key string) (uint64, bool) {
    if v, ok := m.get(key); ok {
        return 0, ok
    } else {
        return v.(uint64), ok
    }
}

func (m *SimpleMap) GetString(key string) (string, bool) {
    if v, ok := m.get(key); ok {
        return v.(string), true
    } else {
        return "", false
    }
}

func (m *SimpleMap) GetBytes(key string) ([]byte, bool) {
    if v, ok := m.get(key); ok {
        return v.([]byte), true
    } else {
        return nil, false
    }
}

// SharedMap
// SharedMap has all the capabilities of the golang map but it is implemented using Redis backend, which
// let it be shared among different processes or services.
// Each SharedMap must have a unique name which is chosen when creating one.
// If you are using a clustered Redis then you can set part of your map name in curly brackets
// Example:
//  sharedMap := NewSharedMap(redisCache, fmt.Sprintf("MAP.{%s}", mapName))
type SharedMap struct {
    Name       string
    redisCache RedisCache
}

// NewSharedMap
// This is the constructor for ShareMap, which uniquely identified by 'mapName'
func NewSharedMap(redisCache RedisCache, mapName string) *SharedMap {
    m := new(SharedMap)
    m.Name = mapName
    m.redisCache = redisCache
    return m
}

// DeleteSharedMap
// Since SharedMap has its data in a Redis cache, if in any case you need to delete stored data from Redis
// you must call this function.
func DeleteSharedMap(redisCache RedisCache, mapName string) {
    c := redisCache.GetConn()
    defer c.Close()

    c.Del(mapName)
}

func (m *SharedMap) Set(key string, value interface{}) {
    c := m.redisCache.GetConn()
    defer c.Close()

    c.HSet(m.Name, key, value)
}

func (m *SharedMap) get(key string) (interface{}, bool) {
    c := m.redisCache.GetConn()
    defer c.Close()

    if v, err := c.HGet(m.Name, key); err != nil {

        return nil, false
    } else {
        return v, true
    }
}

func (m *SharedMap) Get(key string) (interface{}, bool) {
    return m.get(key)
}

func (m *SharedMap) GetInt(key string) (int, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int), ok
    }
}

func (m *SharedMap) GetInt32(key string) (int32, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int32), ok
    }
}

func (m *SharedMap) GetInt64(key string) (int64, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int64), ok
    }
}

func (m *SharedMap) GetUInt(key string) (uint, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(uint), ok
    }
}

func (m *SharedMap) GetUInt32(key string) (uint32, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(uint32), ok
    }
}

func (m *SharedMap) GetUInt64(key string) (uint64, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(uint64), ok
    }
}

func (m *SharedMap) GetString(key string) (string, bool) {
    if v, ok := m.get(key); ok {
        return v.(string), true
    } else {
        return "", false
    }
}

func (m *SharedMap) GetBytes(key string) ([]byte, bool) {
    if v, ok := m.get(key); ok {
        return v.([]byte), true
    } else {
        return nil, false
    }
}

func (m *SharedMap) Inc(key string, value int64) (int64, bool) {
    c := m.redisCache.GetConn()
    defer c.Close()

    if n, err := redis.Int64(c.HIncrementBy(m.Name, key, value)); err != nil {
        return 0, false
    } else {
        return n, true
    }
}

func (m *SharedMap) Keys() (keys []string, err error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    keys, err = redis.Strings(c.HKeys(m.Name))
    return
}

// SimpleInt64Map
type SimpleInt64Map struct {
    sync.Mutex
    keyPairs map[int64]interface{}
}

func NewSimpleInt64Map(initialCapacity int) *SimpleInt64Map {
    m := new(SimpleInt64Map)
    if initialCapacity == 0 {
        initialCapacity = 100
    }
    m.keyPairs = make(map[int64]interface{}, initialCapacity)
    return m
}

func (m *SimpleInt64Map) Set(key int64, value interface{}) {
    m.Lock()
    defer m.Unlock()

    m.keyPairs[key] = value
}

func (m *SimpleInt64Map) get(key int64) (interface{}, bool) {
    m.Lock()
    defer m.Unlock()

    if v, ok := m.keyPairs[key]; ok {
        return v, ok
    } else {
        return v, ok
    }
}

func (m *SimpleInt64Map) Get(key int64) (interface{}, bool) {
    return m.get(key)
}

func (m *SimpleInt64Map) GetInt(key int64) (int, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int), ok
    }

}

func (m *SimpleInt64Map) GetInt32(key int64) (int32, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int32), ok
    }

}

func (m *SimpleInt64Map) GetInt64(key int64) (int64, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int64), ok
    }
}

func (m *SimpleInt64Map) GetUInt(key int64) (uint, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(uint), ok
    }
}

func (m *SimpleInt64Map) GetUInt32(key int64) (uint32, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(uint32), ok
    }
}

func (m *SimpleInt64Map) GetUInt64(key int64) (uint64, bool) {
    if v, ok := m.Get(key); ok {
        return 0, ok
    } else {
        return v.(uint64), ok
    }
}

func (m *SimpleInt64Map) GetString(key int64) (string, bool) {
    if v, ok := m.get(key); ok {
        return v.(string), true
    } else {
        return "", false
    }
}

func (m *SimpleInt64Map) GetBytes(key int64) ([]byte, bool) {
    if v, ok := m.get(key); ok {
        return v.([]byte), true
    } else {
        return nil, false
    }
}

// SharedMap
type SharedInt64Map struct {
    id         string
    redisCache RedisCache
}

// NewSharedMap
// This is the constructor for ShareMap, which uniquely identified by 'mapName'
func NewSharedInt64Map(redisCache RedisCache, prefix string, mapID int64) *SharedInt64Map {
    m := new(SharedInt64Map)
    m.id = fmt.Sprintf("%s.%d", prefix, mapID)
    m.redisCache = redisCache
    return m
}

func (m *SharedInt64Map) Set(key int64, value interface{}) {
    c := m.redisCache.GetConn()
    defer c.Close()

    c.HSet(m.id, key, value)
}

func (m *SharedInt64Map) get(key int64) (interface{}, bool) {
    c := m.redisCache.GetConn()
    defer c.Close()

    if v, err := c.HGet(m.id, key); err != nil {
        return nil, false
    } else {
        return v, true
    }
}

func (m *SharedInt64Map) Get(key int64) (interface{}, bool) {
    return m.get(key)
}

func (m *SharedInt64Map) GetInt(key int64) (int, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int), ok
    }
}

func (m *SharedInt64Map) GetInt32(key int64) (int32, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int32), ok
    }
}

func (m *SharedInt64Map) GetInt64(key int64) (int64, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(int64), ok
    }
}

func (m *SharedInt64Map) GetUInt(key int64) (uint, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(uint), ok
    }
}

func (m *SharedInt64Map) GetUInt32(key int64) (uint32, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(uint32), ok
    }
}

func (m *SharedInt64Map) GetUInt64(key int64) (uint64, bool) {
    if v, ok := m.get(key); !ok {
        return 0, ok
    } else {
        return v.(uint64), ok
    }
}

func (m *SharedInt64Map) GetString(key int64) (string, bool) {
    if v, ok := m.get(key); ok {
        return v.(string), true
    } else {
        return "", false
    }
}

func (m *SharedInt64Map) GetBytes(key int64) ([]byte, bool) {
    if v, ok := m.get(key); ok {
        return v.([]byte), true
    } else {
        return nil, false
    }
}

func (m *SharedInt64Map) Inc(key int64, value int64) (int64, bool) {
    c := m.redisCache.GetConn()
    defer c.Close()

    if n, err := redis.Int64(c.HIncrementBy(m.id, key, value)); err != nil {
        return 0, false
    } else {
        return n, true
    }
}

func (m *SharedInt64Map) Keys() (keys []int64, err error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    keys, err = redis.Int64s(c.HKeys(m.id))
    return
}
