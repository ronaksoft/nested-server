package ronak

import (
    "github.com/gomodule/redigo/redis"
)


// RedisCounterManager
type RedisCounterManager struct {
    redisCache *RedisCache
}

func NewCounterManager(cache *RedisCache) *RedisCounterManager {
    m := new(RedisCounterManager)
    m.redisCache = cache
    return m
}

func (m *RedisCounterManager) Exists(counterName string) bool {
    c := m.redisCache.GetConn()
    defer c.Close()

    if b, err := c.Exists(counterName); err == nil && b {
        return true
    }
    return false
}

func (m *RedisCounterManager) Inc(counterName string, n int64) (v int64, err error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    v, err = redis.Int64(c.IncBy(counterName, n))
    return
}

func (m *RedisCounterManager) Get(counterName string) (v int64, err error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    v, err = redis.Int64(c.Get(counterName))
    return
}

