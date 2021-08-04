package cache

import (
	"git.ronaksoft.com/nested/server/pkg/log"
	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"
)

/*
   Creation Time: 2021 - Aug - 04
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type Manager struct {
	Pool *redis.Pool
}

func New(redisDSN string) (*Manager, error) {
	cm := new(Manager)
	if _, err := redis.Dial("tcp", redisDSN); err != nil {
		log.Warn("We got error on dialing Redis", zap.Error(err), zap.String("DSN", redisDSN))
		return nil, err
	} else {
		cm.Pool = &redis.Pool{
			MaxIdle:   10,
			MaxActive: 1000,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", redisDSN)
				if err != nil {
					log.Warn("We got error on dial redis pool conn", zap.Error(err))
				}
				return c, err
			},
		}
	}
	return cm, nil
}

func (cm *Manager) GetConn() redis.Conn {
	c := cm.Pool.Get()
	return c
}

func (cm *Manager) FlushCache() {
	c := cm.Pool.Get()
	defer c.Close()
	c.Do("FLUSHALL")
}
