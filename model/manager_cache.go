package nested

import (
    "github.com/gomodule/redigo/redis"
    "log"
)

type CacheManager struct {
    Pool *redis.Pool
}

func NewCacheManager(redisDSN string) (*CacheManager, error) {
    cm := new(CacheManager)
    if _, err := redis.Dial("tcp", redisDSN); err != nil {
        log.Println("Redis Pool Connection Error", err.Error())
        return nil, err
    } else {
        cm.Pool = &redis.Pool{
            MaxIdle:   10,
            MaxActive: 1000,
            Dial: func() (redis.Conn, error) {
                c, err := redis.Dial("tcp", redisDSN)
                if err != nil {
                    log.Println("Redis Pool Connection Error", err.Error())
                }
                return c, err
            },
        }
    }
    return cm, nil
}

func (cm *CacheManager) getConn() redis.Conn {
    c := cm.Pool.Get()
    // c.Query("SELECT", 1001)
    return c
}

func (cm *CacheManager) FlushCache() {
    c := cm.Pool.Get()
    defer c.Close()
    c.Do("FLUSHALL")
}
