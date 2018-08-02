package ronak

import "github.com/gomodule/redigo/redis"

var (
    LUA_POP_ALL = `
local result = {}
local length = tonumber(redis.call('LLEN', KEYS[1]))
for i = 1 , length do
    local val = redis.call('LPOP',KEYS[1])
    if val then
        table.insert(result,val)
    end
end
return result
`
)

type RedisQueueManager struct {
    redisCache *RedisCache
}

func NewRedisQueueManager(redisCache *RedisCache) *RedisQueueManager {
    m := new(RedisQueueManager)
    m.redisCache = redisCache
    return m
}

func (m *RedisQueueManager) Exists(queueName string) bool {
    c := m.redisCache.GetConn()
    defer c.Close()

    if b, err := c.Exists(queueName); err == nil && b {
        return true
    }
    return false
}

func (m *RedisQueueManager) Push(queueName string, item interface{}) (size int, err error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    size, err = c.LPush(queueName, item)

    return
}

func (m *RedisQueueManager) PushWithLimit(queueName string, item interface{}, limit int) (size int, err error) {
    // FIXME: use LUA script to improve performance
    if l, err := m.Length(queueName); err != nil || l >= limit {
        return l, err
    }
    return m.Push(queueName, item)
}

func (m *RedisQueueManager) Pop(queueName string) (b []byte, err error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    b, err = redis.Bytes(c.RPop(queueName))

    return
}

func (m *RedisQueueManager) PopAll(queueName string) (b [][]byte, err error) {
    // FIXME:: use LUA script to improve performance
    c := m.redisCache.GetConn()
    defer c.Close()

    qLength, err := m.Length(queueName)
    if err != nil {
        return nil, err
    }

    if qLength >= 1 {
        b, err = c.LRangeBytes(queueName, 0, int(qLength-1))
        if err != nil {
            return nil, err
        }
        c.LTrim(queueName, int(qLength), -1)
    }

    return b, nil
}

func (m *RedisQueueManager) Length(queueName string) (l int, err error) {
    c := m.redisCache.GetConn()
    defer c.Close()

    l, err = c.LLen(queueName)

    return
}
