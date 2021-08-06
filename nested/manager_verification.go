package nested

import (
	"crypto/md5"
	"encoding/base64"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"go.uber.org/zap"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

const (
	TestPhoneNumber       string = "98123456789"
	TestEmail             string = "test@nested.me"
	MagicVerificationCode string = "VER20170209justForTe$T####PQRTS"
)

type Verification struct {
	ID        string              `json:"_id" bson:"_id"`
	Phone     string              `json:"phone" bson:"phone"`
	Email     string              `json:"email" bson:"email"`
	Counters  VerificationCounter `json:"counters" bson:"counters"`
	ShortCode string              `json:"short_code" bson:"short_code"`
	LongCode  string              `json:"long_code" bson:"long_code"`
	Verified  bool                `json:"verified" bson:"verified"`
	Timestamp int64               `json:"timestamp" bson:"timestamp"`
	Expired   bool                `json:"expired" bson:"expired"`
}
type VerificationCounter struct {
	Attempts int `json:"attempts" bson:"attempts"`
	Sms      int `json:"sms" bson:"sms"`
	Email    int `json:"email" bson:"email"`
	Call     int `json:"call" bson:"call"`
}
type VerificationManager struct{}

func newVerificationManager() *VerificationManager {
	return new(VerificationManager)
}

// CreateByPhone
// Creates a verification object for 'phone' and return the verification object to caller
// if verification object is nil then something has been wrong
func (vm *VerificationManager) CreateByPhone(phone string) *Verification {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := new(Verification)
	v.Phone = phone
	v.ID = RandomID(32)
	v.Timestamp = time.Now().Unix()
	if v.Phone == TestPhoneNumber {
		v.ShortCode = "123456"
		v.LongCode = "TEST_LONG_CODE_KEY"
	} else {
		v.ShortCode = RandomDigit(6)
		v.LongCode = base64.URLEncoding.EncodeToString(md5.New().Sum([]byte(RandomID(10))))
	}
	if err := db.C(global.CollectionVerifications).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return v
}

// CreateByEmail
// Creates a verification object for 'email' and return the verification object to caller
// if verification object is nil then something has been wrong
func (vm *VerificationManager) CreateByEmail(email string) *Verification {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := new(Verification)
	v.Email = email
	v.ID = RandomID(32)
	v.Timestamp = time.Now().Unix()
	if v.Email == TestEmail {
		v.ShortCode = "123456"
		v.LongCode = "TEST_LONG_CODE_KEY"
	} else {
		v.ShortCode = RandomDigit(6)
		v.LongCode = base64.URLEncoding.EncodeToString(md5.New().Sum([]byte(RandomID(10))))
	}
	if err := db.C(global.CollectionVerifications).Insert(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return v
}

// GetByID
// Returns the Verification object identified by ID, this function does not check any
// extra parameter. It returns the Verification object even if it was expired or verified ...
func (vm *VerificationManager) GetByID(verifyID string) *Verification {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := new(Verification)
	if err := db.C(global.CollectionVerifications).FindId(verifyID).One(v); err != nil {
		log.Warn("Got error", zap.Error(err))
		return nil
	}
	return v
}

// Verify
// Returns true if the code matches verification code otherwise if attempts are exceeded the limit
// or expire time has been passed the verification object is expired.
func (vm *VerificationManager) Verify(verifyID, code string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := new(Verification)
	ch := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"counters.attempts": 1}},
		ReturnNew: true,
	}
	if _, err := db.C(global.CollectionVerifications).FindId(verifyID).Apply(ch, v); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	// expire the verification if too many wrong attempts or too long
	if v.Expired {
		return false
	}
	// increment the counter
	v.Counters.Attempts += 1

	// if this verification has been expired then return false
	if time.Unix(v.Timestamp, 0).Add(24 * time.Hour).Before(time.Now()) {
		vm.Expire(verifyID)
		return false
	}

	// if attempts are more than permitted value then return false and expire the verification object
	if v.Counters.Attempts > global.DefaultMaxVerificationAttempts {
		vm.Expire(verifyID)
		return false
	}

	if v.ShortCode == code || v.LongCode == code {
		v.Verified = true
		db.C(global.CollectionVerifications).UpdateId(
			v.ID,
			bson.M{"$set": bson.M{"verified": true}},
		)
	}

	return v.Verified
}

// Verified
// Returns true if verification identified by verifyID is verified otherwise returns false
func (vm *VerificationManager) Verified(verifyID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := new(Verification)
	if err := db.C(global.CollectionVerifications).FindId(verifyID).One(v); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return v.Verified
}

// IncrementSmsCounter Increments the counter for number SMS messages have been sent for this Verification object.
func (vm *VerificationManager) IncrementSmsCounter(verifyID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := new(Verification)
	if err := db.C(global.CollectionVerifications).FindId(verifyID).One(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	if err := db.C(global.CollectionVerifications).UpdateId(verifyID, bson.M{"$inc": bson.M{"counters.sms": 1}}); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
}

// IncrementCallCounter Increments the counter for number of calls have been made for this Verification object.
func (vm *VerificationManager) IncrementCallCounter(verifyID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := new(Verification)
	if err := db.C(global.CollectionVerifications).FindId(verifyID).One(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	if err := db.C(global.CollectionVerifications).UpdateId(verifyID, bson.M{"$inc": bson.M{"counters.call": 1}}); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
}

// IncrementEmailCounter Increments the counter for the number of emails have been sent for this Verification object.
func (vm *VerificationManager) IncrementEmailCounter(verifyID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := new(Verification)
	if err := db.C(global.CollectionVerifications).FindId(verifyID).One(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	if err := db.C(global.CollectionVerifications).UpdateId(verifyID, bson.M{"$inc": bson.M{"counters.email": 1}}); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
}

// Expire expires the verification identified by "verifyID" and that Verification object cannot be verified anymore.
func (vm *VerificationManager) Expire(verifyID string) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	v := new(Verification)
	if err := db.C(global.CollectionVerifications).FindId(verifyID).One(v); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	if err := db.C(global.CollectionVerifications).UpdateId(verifyID, bson.M{"$set": bson.M{"expired": true}}); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
}
