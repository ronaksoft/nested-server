package cache

import (
	"fmt"
	"git.ronaksoft.com/nested/server/model"
	"github.com/gomodule/redigo/redis"
)

type CacheManager struct {
	Pool *redis.Pool
}

func NewCacheManager(redisDSN string) (*CacheManager, error) {
	cm := new(CacheManager)
	if _, err := redis.Dial("tcp", redisDSN); err != nil {
		fmt.Println("Redis Pool Connection Error", err.Error())
		return nil, err
	} else {
		cm.Pool = &redis.Pool{
			MaxIdle:   10,
			MaxActive: 1000,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", redisDSN)
				if err != nil {
					fmt.Println("Redis Pool Connection Error", err.Error())
				}
				return c, err
			},
		}
	}
	return cm, nil
}

func (cm *CacheManager) getConn() redis.Conn {
	c := cm.Pool.Get()
	return c
}

func (cm *CacheManager) CountPostAdd() {
	c := cm.getConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", nested.ReportCounterPostAdd)
	c.Do("INCR", key)
}

func (cm *CacheManager) CountPostAttachCount(n int) {
	c := cm.getConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", nested.ReportCounterPostAttachCount)
	c.Do("INCRBY", key, n)
}

func (cm *CacheManager) CountPostAttachSize(n int64) {
	c := cm.getConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", nested.ReportCounterPostAttachSize)
	c.Do("INCRBY", key, n)
}

func (cm *CacheManager) CountPostPerPlace(placeIDs []string) {
	c := cm.getConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", nested.ReportCounterPostPerPlace)
	for _, placeID := range placeIDs {
		c.Send("HINCRBY", key, placeID, 1)
	}
	c.Flush()
}

func (cm *CacheManager) CountPostExternalAdd() {
	c := cm.getConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", nested.ReportCounterPostExternalAdd)
	c.Do("INCR", key)
}

func (cm *CacheManager) CountPostPerAccount(accountID string) {
	c := cm.getConn()
	defer c.Close()
	key := fmt.Sprintf("report:counter:%s", nested.ReportCounterPostPerAccount)
	c.Do("HINCRBY", key, accountID, 1)
}

func (cm *CacheManager) PlaceRemoveCache(placeID string) bool {
	c := cm.getConn()
	defer c.Close()
	keyID := fmt.Sprintf("place:gob:%s", placeID)
	c.Do("DEL", keyID)
	return true
}
