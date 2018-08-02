package ronak

import "github.com/gomodule/redigo/redis"

/*
    Creation Time: 2018 - Apr - 19
    Created by:  (ehsan)
    Maintainers:
        1.  (ehsan)
    Auditor: Ehsan N. Moosa
    Copyright Ronak Software Group 2018
*/

type RedisMapManager struct {
    redisCache *RedisCache
}

func NewRedisMapManager(redisCache *RedisCache) *RedisMapManager {
    m := new(RedisMapManager)
    m.redisCache = redisCache
    return m
}

func (m *RedisMapManager) Exists(mapName string) bool {
    c := m.redisCache.GetConn()
    defer c.Close()

    if b, err := c.Exists(mapName); err == nil && b {
        return true
    }
    return false
}

func (m *RedisMapManager) Set(mapName string, key, value interface{}) (new bool, err error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    new, err = redis.Bool(c.HSet(mapName, key, value))

    return
}

func (m *RedisMapManager) SetMulti(mapName string, kv M) error {
    c := m.redisCache.GetConn()
    defer c.Close()

    _, err := c.HMSet(mapName, kv)

    return err
}

func (m *RedisMapManager) get(mapName string, key interface{}) (reply interface{}, err error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    reply, err = c.HGet(mapName, key)

    return
}

func (m *RedisMapManager) Get(mapName string, key interface{}) ([]byte, error) {
    return redis.Bytes(m.get(mapName, key))
}

func (m *RedisMapManager) GetInt(mapName string, key interface{}) (int, error) {
    return redis.Int(m.get(mapName, key))
}

func (m *RedisMapManager) GetInt64(mapName string, key interface{}) (int64, error) {
    return redis.Int64(m.get(mapName, key))
}

func (m *RedisMapManager) GetString(mapName string, key interface{}) (string, error) {
    return redis.String(m.get(mapName, key))
}

func (m *RedisMapManager) GetMultiInt64(mapName string, keys ...interface{}) (map[string]int64, error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    if arr, err := redis.Int64s(c.HMGet(mapName, keys...)); err != nil {
        return nil, err
    } else {
        reply := make(map[string]int64)
        for idx, v := range arr {
            reply[keys[idx].(string)] = v
        }
        return reply, nil
    }

}

func (m *RedisMapManager) GetMultiString(mapName string, keys ...interface{}) (map[string]string, error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    if arr, err := redis.Strings(c.HMGet(mapName, keys...)); err != nil {
        return nil, err
    } else {
        reply := make(map[string]string)
        for idx, v := range arr {
            reply[keys[idx].(string)] = v
        }
        return reply, nil
    }
}

func (m *RedisMapManager) Inc(mapName string, key string, n int) (reply int64, err error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    reply, err = redis.Int64(c.HIncrementBy(mapName, key, n))

    return
}

func (m *RedisMapManager) Delete(mapName string) error {
    c := m.redisCache.GetConn()
    defer c.Close()

    _, err := c.Del(mapName)
    return err

}

func (m *RedisMapManager) DeleteKey(mapName string, key string) error {
    c := m.redisCache.GetConn()
    defer c.Close()

    _, err := c.HDel(mapName, key)
    return err

}
