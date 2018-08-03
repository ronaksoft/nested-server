package nested

import (
    "fmt"
    "github.com/gomodule/redigo/redis"
    "strings"
)

type Websocket struct {
    WebsocketID string
    BundleID    string
    UID         string
    DeviceID    string
}

type WebsocketManager struct{}

func NewWebsocketManager() *WebsocketManager { return new(WebsocketManager) }

// Register
// save websockets in the cache.
// Following keys will be set in the cache
//	1. HASH::	ws:account:[accountID]
// 		[bundleID]:[websocketID] ==> [deviceID]
//
//	2. SET::	ws:bundle:[bundleID]
//		==> [accountID]
//
//	3. KEY-VALUE::	bundle-ws:[bundleID]:[websocketID]
//		==> [accountID]
//
//	HashKey is used to find websockets and deviceIDs by [accountID]
//	SetKey is used to find accountIDs by [bundleID]
//	DHKey is used to find accountID but [bundleID],[websocketID]
func (wm *WebsocketManager) Register(websocketID, bundleID, deviceID, accountID string) bool {
    _funcName := "WebsocketManager::Register"
    _Log.FunctionStarted(_funcName, websocketID, bundleID, deviceID, accountID)
    defer _Log.FunctionFinished(_funcName)

    c := _Cache.Pool.Get()
    defer c.Close()
    if c == nil {
        _Log.Error(_funcName, "Cannot get redis connection")
        return false
    }

    // Creates an new field bundleID:websocketID to be used by GetWebsocketsByAccountID
    hashKeyID := fmt.Sprintf("ws:account:%s", accountID)
    fieldName := fmt.Sprintf("%s:%s", bundleID, websocketID)
    c.Do("HSET", hashKeyID, fieldName, deviceID)

    // Creates a new field accountID to be used by RemoveWebsocketsByBundleID
    setKeyID := fmt.Sprintf("ws:bundle:%s", bundleID)
    c.Do("SADD", setKeyID, accountID)

    // Create a new key-value per bundleID / websocketID
    keyID := fmt.Sprintf("bundle-ws:%s:%s", bundleID, websocketID)
    c.Do("SET", keyID, accountID)

    return true
}

// GetWebsocketsByAccountID
// Returns an array of Websocket, if bundleID != "" then it only returns websockets
// which are in the bundleID
func (wm *WebsocketManager) GetWebsocketsByAccountID(accountID, bundleID string) []Websocket {
    _funcName := "WebsocketManager::GetWebsocketsByAccountID"
    _Log.FunctionStarted(_funcName, accountID)
    defer _Log.FunctionFinished(_funcName)

    c := _Cache.Pool.Get()
    defer c.Close()
    if c == nil {
        _Log.Error(_funcName, "Cannot get redis connection")
        return []Websocket{}
    }
    websockets := make([]Websocket, 0)
    hashKeyID := fmt.Sprintf("ws:account:%s", accountID)
    if m, err := redis.StringMap(c.Do("HGETALL", hashKeyID)); err != nil {
        _Log.Error(_funcName, err.Error())
        return []Websocket{}
    } else {
        for key, deviceID := range m {
            // fieldKey :: bundleID:websocketID
            fieldKey := strings.SplitN(key, ":", 2)
            if len(bundleID) > 0 && bundleID != fieldKey[0] {
                continue
            }
            websockets = append(websockets, Websocket{
                UID:         accountID,
                BundleID:    fieldKey[0],
                WebsocketID: fieldKey[1],
                DeviceID:    deviceID,
            })
        }
    }
    return websockets
}

// GetAccountsByBundleID
func (wm *WebsocketManager) GetAccountsByBundleID(bundleID string) []string {
    _funcName := "WebsocketManager::GetWebsocketsByBundleID"
    _Log.FunctionStarted(_funcName)
    defer _Log.FunctionFinished(_funcName)

    c := _Cache.Pool.Get()
    defer c.Close()
    if c == nil {
        _Log.Error(_funcName, "Cannot get redis connection")
        return []string{}
    }
    setKeyID := fmt.Sprintf("ws:bundle:%s", bundleID)
    if accountIDs, err := redis.Strings(c.Do("SMEMBERS", setKeyID)); err != nil {
        _Log.Error(_funcName, err.Error())
    } else {
        return accountIDs
    }
    return []string{}
}

// Remove
// Removes the websocketID in the bundleID
func (wm *WebsocketManager) Remove(websocketID, bundleID string) *Websocket {
    _funcName := "WebsocketManager::Remove"
    _Log.FunctionStarted(_funcName, websocketID, bundleID)
    defer _Log.FunctionFinished(_funcName)

    c := _Cache.Pool.Get()
    defer c.Close()
    if c == nil {
        _Log.Error(_funcName, "Cannot get redis connection")
        return nil
    }

    ws := new(Websocket)
    keyID := fmt.Sprintf("bundle-ws:%s:%s", bundleID, websocketID)
    if accountID, err := redis.String(c.Do("GET", keyID)); err != nil {
        _Log.Error(_funcName, err.Error(), bundleID, websocketID)
        return nil
    } else {
        fieldName := fmt.Sprintf("%s:%s", bundleID, websocketID)
        hashKeyID := fmt.Sprintf("ws:account:%s", accountID)
        deviceID, _ := redis.String(c.Do("HGET", hashKeyID, fieldName))
        c.Send("HDEL", hashKeyID, fieldName)
        c.Send("DEL", keyID)
        c.Flush()
        m, _ := redis.StringMap(c.Do("HGETALL", hashKeyID))
        hasMoreConnection := false
        for k := range m {
            if strings.HasPrefix(k, bundleID) {
                hasMoreConnection = true
                break
            }
        }
        if !hasMoreConnection {
            setKeyID := fmt.Sprintf("ws:bundle:%s", bundleID)
            c.Do("SREM", setKeyID, accountID)
        }
        ws.BundleID = bundleID
        ws.WebsocketID = websocketID
        ws.UID = accountID
        ws.DeviceID = deviceID
    }

    return ws
}

// RemoveByBundleID
func (wm *WebsocketManager) RemoveByBundleID(bundleID string) {
    _funcName := "WebsocketManager::RemoveByBundleID"
    _Log.FunctionStarted(_funcName, bundleID)
    defer _Log.FunctionFinished(_funcName)

    c := _Cache.Pool.Get()
    defer c.Close()
    if c == nil {
        _Log.Error(_funcName, "Cannot get redis connection")
        return
    }

    setKeyID := fmt.Sprintf("ws:bundle:%s", bundleID)
    if accountIDs, err := redis.Strings(c.Do("SMEMBERS", setKeyID)); err != nil {
        _Log.Error(_funcName, err.Error())
    } else {
        for _, accountID := range accountIDs {
            hashKeyID := fmt.Sprintf("ws:account:%s", accountID)
            m, _ := redis.StringMap(c.Do("HGETALL", hashKeyID))
            for k := range m {
                if strings.HasPrefix(k, bundleID) {
                    c.Do("HDEL", hashKeyID, k)
                }
            }
        }
    }
}

// IsConnected
// Returns a map of accountIDs with TRUE value for each accountID which has at least one open socket
func (wm *WebsocketManager) IsConnected(accountIDs []string) MB {
    _funcName := "WebsocketManager::IsConnected"
    _Log.FunctionStarted(_funcName, accountIDs)
    defer _Log.FunctionFinished(_funcName)
    res := MB{}
    c := _Cache.Pool.Get()
    defer c.Close()
    if c == nil {
        _Log.Error(_funcName, "cannot get redis connection")
        return nil
    }
    for _, accountID := range accountIDs {
        keyID := fmt.Sprintf("ws:account:%s", accountID)
        res[keyID] = false
        n, _ := redis.Int(c.Do("HLEN", keyID))
        if n > 0 {
            res[keyID] = true
        }
    }
    return res
}
