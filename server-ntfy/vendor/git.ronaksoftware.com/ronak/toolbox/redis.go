package ronak

import (
    "github.com/gomodule/redigo/redis"
    "time"
)

/*
    Creation Time: 2018 - Apr - 07
    Created by:  Ehsan N. Moosa (ehsan)
    Maintainers:
        1.  Ehsan N. Moosa (ehsan)
    Auditor: Ehsan N. Moosa
    Copyright Ronak Software Group 2018
*/

// RedisConfig
type RedisConfig struct {
    MaxIdleConnections   int
    MaxActiveConnections int
    Password             string
    Host                 string
    DialReadTimeout      time.Duration
    DialWriteTimeout     time.Duration
}

var (
    DefaultRedisConfig = RedisConfig{
        MaxIdleConnections:   10,
        MaxActiveConnections: 1000,
        DialReadTimeout:      5 * time.Second,
        DialWriteTimeout:     5 * time.Second,
    }
)

// RedisCache
type RedisCache struct {
    pool *redis.Pool
    conn redis.Conn
}

// NewRedisCache
// This is the constructor of RedisCache, it accepts RedisConfig as input, you can use
// DefaultRedisConfig for quick initialization, but make sure to add 'Conn' and 'Password' to it
//
// example:
// conf := ronak.DefaultRedisConfig
// conf.Conn = "your-host.com"
// conf.Password = "password123"
// c := NewRedisCache(conf)
func NewRedisCache(conf RedisConfig) *RedisCache {
    r := new(RedisCache)
    if len(conf.Password) == 0 {
        r.pool = &redis.Pool{
            MaxIdle:   conf.MaxIdleConnections,
            MaxActive: conf.MaxActiveConnections,
            Dial: func() (redis.Conn, error) {
                if c, err := redis.Dial(
                    "tcp",
                    conf.Host,
                    redis.DialReadTimeout(conf.DialReadTimeout),
                    redis.DialWriteTimeout(conf.DialWriteTimeout),
                ); err != nil {
                    _LOG.Fatal("NewRedisCache", err.Error())
                    return c, err
                } else {
                    return c, nil
                }
            },
        }
    } else {
        r.pool = &redis.Pool{
            MaxIdle:   10,
            MaxActive: 1000,
            Dial: func() (redis.Conn, error) {
                if c, err := redis.Dial(
                    "tcp",
                    conf.Host,
                    redis.DialPassword(conf.Password),
                    redis.DialReadTimeout(conf.DialReadTimeout),
                    redis.DialWriteTimeout(conf.DialWriteTimeout),
                ); err != nil {
                    _LOG.Fatal("NewRedisCache", err.Error())
                    return c, err
                } else {
                    return c, nil
                }
            },
        }
    }

    return r
}

func (r *RedisCache) GetConn() RedisConn {
    c := RedisConn{
        c: r.pool.Get(),
    }
    return c
}

func (r *RedisCache) Close() {
    if err := r.pool.Close(); err != nil {
        _LOG.Error("RedisCache::Close", err.Error())
    }
}

// RedisConn
type RedisConn struct {
    c redis.Conn
}

func (rc *RedisConn) Inc(keyName string) (reply interface{}, err error) {
    return rc.c.Do("INCR", keyName)
}

func (rc *RedisConn) IncBy(keyName string, n int64) (reply interface{}, err error) {
    return rc.c.Do("INCR", keyName, n)
}

//  ////////////////////
// Hash Functions
//  //////////////
func (rc *RedisConn) HSet(hashName string, fieldName, value interface{}) (reply interface{}, err error) {
    return rc.c.Do("HSET", hashName, fieldName, value)
}

func (rc *RedisConn) HMSet(hashName string, kv M) (reply interface{}, err error) {
    args := make([]interface{}, 0, len(kv)*2+1)
    args = append(args, hashName)
    for key, value := range kv {
        args = append(args, key, value)
    }
    return rc.c.Do("HMSET", args...)
}

func (rc *RedisConn) HGet(hashName string, fieldName interface{}) (reply interface{}, err error) {
    return rc.c.Do("HGET", hashName, fieldName)
}

func (rc *RedisConn) HMGet(hashName string, fieldNames ...interface{}) (reply interface{}, err error) {
    args := make([]interface{}, 0, len(fieldNames)+1)
    args = append(args, hashName)
    args = append(args, fieldNames...)
    return rc.c.Do("HMGET", args...)
}

func (rc *RedisConn) HGetAll(hashName string) (reply interface{}, err error) {
    return rc.c.Do("HGETALL", hashName)
}

func (rc *RedisConn) HIncrementBy(hashName, fieldName, incr interface{}) (reply interface{}, err error) {
    return rc.c.Do("HINCRBY", hashName, fieldName, incr)
}

func (rc *RedisConn) HKeys(hashName string) (reply interface{}, err error) {
    return rc.c.Do("HKEYS", hashName)
}

func (rc *RedisConn) HValues(hashName string) (reply interface{}, err error) {
    return rc.c.Do("HVALS", hashName)
}

func (rc *RedisConn) HDel(hashName string, fieldName string) (reply interface{}, err error) {
    return rc.c.Do("HDEL", hashName, fieldName)
}

func (rc *RedisConn) Del(keyName string) (reply interface{}, err error) {
    return rc.c.Do("DEL", keyName)
}

func (rc *RedisConn) Get(keyName string) (reply interface{}, err error) {
    return rc.c.Do("GET", keyName)
}

func (rc *RedisConn) Set(keyName string, value interface{}) (reply interface{}, err error) {
    return rc.c.Do("SET", keyName, value)
}

func (rc *RedisConn) SetEx(keyName string, value, ttl interface{}) (reply interface{}, err error) {
    return rc.c.Do("SETEX", keyName, ttl, value)
}

func (rc *RedisConn) SetNx(keyName string, value interface{}) (reply interface{}, err error) {
    return rc.c.Do("SETNX", keyName, value)
}

func (rc *RedisConn) RPop(listName string) (reply interface{}, err error) {
    return rc.c.Do("RPOP", listName)
}

func (rc *RedisConn) BRPop(listName string) (reply interface{}, err error) {
    return rc.c.Do("BRPOP", listName, 120)
}

func (rc *RedisConn) LPop(listName string) (reply interface{}, err error) {
    return rc.c.Do("LPOP", listName)
}

func (rc *RedisConn) RPush(listName string, item interface{}) (size int, err error) {
    return redis.Int(rc.c.Do("RPUSH", listName, item))
}

func (rc *RedisConn) LPush(listName string, item interface{}) (size int, err error) {
    return redis.Int(rc.c.Do("LPUSH", listName, item))
}

func (rc *RedisConn) SendLPush(listName string, item interface{}) error {
    return rc.c.Send("LPUSH", listName, item)
}

func (rc *RedisConn) LLen(listName string) (size int, err error) {
    return redis.Int(rc.c.Do("LLEN", listName))
}

func (rc *RedisConn) LRangeBytes(listName string, left, right int) (reply [][]byte, err error) {
    return redis.ByteSlices(rc.c.Do("LRANGE", listName, left, right))
}

func (rc *RedisConn) LTrim(listName string, left, right int) (reply int, err error) {
    return redis.Int(rc.c.Do("LTRIM", listName, left, right))
}

func (rc *RedisConn) SendLTrim(listName string, left, right int) error {
    return rc.c.Send("LTRIM", listName, left, right)
}

func (rc *RedisConn) Exists(keyName string) (reply bool, err error) {
    return redis.Bool(rc.c.Do("EXISTS", keyName))
}

func (rc *RedisConn) SAdd(setName string, value interface{}) (int, error) {
    return redis.Int(rc.c.Do("SADD", setName, value))
}

func (rc *RedisConn) SRemove(setName string, value interface{}) (reply interface{}, err error) {
    return rc.c.Do("SREM", setName, value)
}

func (rc *RedisConn) SMembers(setName string) (interface{}, error) {
    return rc.c.Do("SMEMBERS", setName)
}

func (rc *RedisConn) SCard(setName string) (int, error) {
    return redis.Int(rc.c.Do("SCARD", setName))
}

func (rc *RedisConn) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
    return rc.c.Do(commandName, args...)
}

func (rc *RedisConn) Send(commandName string, args ...interface{}) error {
    return rc.c.Send(commandName, args ...)
}

func (rc *RedisConn) Flush() error {
    return rc.c.Flush()
}

func (rc *RedisConn) Close() error {
    return rc.c.Close()
}

func (rc *RedisConn) Receive() (reply interface{}, err error) {
    return rc.c.Receive()
}
