package nested

import (
    "bytes"
    "encoding/gob"
    "fmt"
    "github.com/gomodule/redigo/redis"
    "github.com/globalsign/mgo/bson"
    "log"
    "time"
)

const (
    TOKEN_TYPE_FILE  string = "file"
    TOKEN_TYPE_APP   string = "app"
)

type FileToken struct {
    ID            string      `bson:"_id" json:"_id"`
    Type          string      `json:"type" bson:"type"`
    FileID        UniversalID `bson:"universal_id" json:"universal_id"`
    Issuer        string      `json:"account_id" bson:"account_id"`
    Receiver      string      `json:"email" bson:"email"`
    AccessCounter int         `json:"access_counter"`
}
type LoginToken struct {
    ID        string `bson:"_id" json:"_id"`
    AccountID string `bson:"account_id" json:"account_id"`
    ExpireOn  uint64 `bson:"expire_time" json:"expire_time"`
    Expired   bool   `bson:"expired" json:"expired"`
}
type AppToken struct {
    ID        string `bson:"_id" json:"_id"`
    AccountID string `bson:"account_id" json:"account_id"`
    AppID     string `bson:"app_id" json:"app_id"`
    Expired   bool   `bson:"expired" json:"-"`
    Favorite  bool   `bson:"favorite" json:"-"`
}

type TokenManager struct{}

func NewTokenManager() *TokenManager {
    return new(TokenManager)
}

func (tm *TokenManager) readFromCache(tokenType, tokenID string) interface{} {
    switch tokenType {
    case TOKEN_TYPE_APP:
        appToken := new(AppToken)
        c := _Cache.Pool.Get()
        defer c.Close()
        keyID := fmt.Sprintf("appToken:gob:%s", tokenID)
        if gobToken, err := redis.Bytes(c.Do("GET", keyID)); err != nil {
            if err := _MongoDB.C(COLLECTION_PLACES).FindId(tokenID).One(appToken); err != nil {
                log.Println("Model::TokenManager::readFromCache::Error 1::", err.Error(), tokenID)
                return nil
            }
            gobToken := new(bytes.Buffer)
            if err := gob.NewEncoder(gobToken).Encode(appToken); err == nil {
                c.Do("SETEX", keyID, CACHE_LIFETIME, gobToken.Bytes())
            }
            return appToken
        } else if err := gob.NewDecoder(bytes.NewBuffer(gobToken)).Decode(appToken); err == nil {
            return appToken
        }
        return nil
    default:
        // Error should not be called
    }
    return nil
}

// CreateFileToken creates a token for a file, and returns Token as a string object
// uniID : UniversalID of the file
// issuer : The accountID who creates this token
// receiver : The email address this file has been sent to
func (tm *TokenManager) CreateFileToken(uniID UniversalID, issuerID, receiverEmail string) (string, error) {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ft := FileToken{
        ID:       fmt.Sprintf("FT%s", RandomID(128)),
        Type:     TOKEN_TYPE_FILE,
        FileID:   uniID,
        Issuer:   issuerID,
        Receiver: receiverEmail,
    }
    if err := db.C(COLLECTION_TOKENS_FILES).Insert(ft); err != nil {
        _Log.Warn(err.Error())
        return "", err
    }
    return ft.ID, nil
}

// CreateLoginToken creates a token object in "tokens.logins" to let user login and change his/her password
// with no need of password set.
func (tm *TokenManager) CreateLoginToken(uid string) string {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    token := LoginToken{
        ID:        RandomID(12),
        AccountID: uid,
        ExpireOn:  uint64(time.Now().AddDate(0, 1, 0).UnixNano() / 1000000),
    }
    if err := db.C(COLLECTION_TOKENS_LOGINS).Insert(token); err != nil {
        _Log.Warn(err.Error())
        return ""
    }
    return token.ID
}

// CreateAppToken creates a token object in "tokens.apps" to let apps interact with server on behalf of users
func (tm *TokenManager) CreateAppToken(accountID, appID string) string {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    token := AppToken{
        ID:        RandomID(36),
        AccountID: accountID,
        AppID:     appID,
        Expired:   false,
        Favorite:  false,
    }
    if err := db.C(COLLECTION_TOKENS_APPS).Find(bson.M{
        "account_id": accountID,
        "app_id":     appID,
    }).One(&token); err != nil {
        if err := db.C(COLLECTION_TOKENS_APPS).Insert(token); err != nil {
            _Log.Warn(err.Error())
            return ""
        }
    }

    return token.ID
}

// GetFileByToken returns the universalID of the file which is attached to this token,
// if any error happens it returns the error message as second return argument
func (tm *TokenManager) GetFileByToken(token string) (UniversalID, error) {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ft := new(FileToken)
    if err := db.C(COLLECTION_TOKENS_FILES).FindId(token).One(ft); err != nil {
        _Log.Warn(err.Error())
        return "", err
    }
    return ft.FileID, nil
}

// GetFileToken returns a pointer to FileToken struct and if any error happens it return nil
func (tm *TokenManager) GetFileToken(token string) *FileToken {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    ft := new(FileToken)
    if err := db.C(COLLECTION_TOKENS_FILES).FindId(token).One(ft); err != nil {
        _Log.Warn(err.Error())
        return nil
    }
    return ft
}

// GetLoginToken returns a pointer of LoginToken struct and if any error happens it returns nil
func (tm *TokenManager) GetLoginToken(token string) *LoginToken {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    loginToken := new(LoginToken)
    if err := db.C(COLLECTION_TOKENS_LOGINS).FindId(token).One(loginToken); err != nil {
        _Log.Warn(err.Error())
        return nil
    }
    if loginToken.Expired || loginToken.ExpireOn < Timestamp() {
        if err := db.C(COLLECTION_TOKENS_LOGINS).RemoveId(token); err != nil {
            _Log.Warn(err.Error())
            return nil
        }
    }
    return loginToken
}

// GetAppToken returns a pointer of AppToken struct if any error happens it returns nil
func (tm *TokenManager) GetAppToken(token string) *AppToken {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    appToken := new(AppToken)
    if err := db.C(COLLECTION_TOKENS_APPS).FindId(token).One(appToken); err != nil {
        _Log.Warn(err.Error())
        return nil
    }
    if appToken.Expired {
        if err := db.C(COLLECTION_TOKENS_APPS).RemoveId(token); err != nil {
            _Log.Warn(err.Error())
            return nil
        }
    }
    return appToken
}

// GetAppTokenByAccountID returns an array of AppTokens for the accountID
func (tm *TokenManager) GetAppTokenByAccountID(accountID string, pg Pagination) []AppToken {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    appTokens := make([]AppToken, 0, 10)
    if err := db.C(COLLECTION_TOKENS_APPS).Find(
        bson.M{"account_id": accountID},
    ).Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&appTokens); err != nil {
        _Log.Warn(err.Error())
    }
    return appTokens
}

func (tm *TokenManager) AppTokenExists(accountID, appID string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if n, err := db.C(COLLECTION_TOKENS_APPS).Find(
        bson.M{"account_id": accountID, "app_id": appID},
    ).Count(); err != nil {
        _Log.Warn(err.Error())
        return false
    } else if n > 0 {
        return true
    }
    return false
}

// IncreaseAccessCounter increases the access counter of the token
func (tm *TokenManager) IncreaseAccessCounter(token string) {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TOKENS_FILES).UpdateId(
        token,
        bson.M{"$inc": bson.M{"access_counter": 1}},
    ); err != nil {
        _Log.Warn(err.Error())
    }
}

// RevokeFileToken revokes the token. The file cannot be accessed by this token anymore.
func (tm *TokenManager) RevokeFileToken(token string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TOKENS_FILES).RemoveId(token); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

// RevokeLoginToken revokes the login token. This is token is disposable that means The user cannot login using this token anymore.
func (tm *TokenManager) RevokeLoginToken(token string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TOKENS_LOGINS).RemoveId(token); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

// RevokeAppToken revokes the app token. The app requests will be failed after revoking the token.
func (tm *TokenManager) RevokeAppToken(accountID, token string) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TOKENS_APPS).Remove(bson.M{
        "_id":        token,
        "account_id": accountID,
    }); err != nil {
        _Log.Warn(err.Error())
        return false
    }
    return true
}

// SetAppFavoriteStatus sets favorite status of an app for user
func (tm *TokenManager) SetAppFavoriteStatus(accountID, appID string, state bool) bool {
    // _funcName

    // removed LOG Function

    dbSession := _MongoSession.Clone()
    db := dbSession.DB(DB_NAME)
    defer dbSession.Close()

    if err := db.C(COLLECTION_TOKENS_APPS).Update(
        bson.M{"account_id": accountID, "app_id": appID,},
        bson.M{"$set": bson.M{"favorite": state}},
    ); err != nil {
        log.Println(_funcName, err.Error())
        return false
    }
    return true
}
