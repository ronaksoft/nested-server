package nested

import (
	"encoding/json"
	"time"

	"github.com/globalsign/mgo/bson"
)

const (
	LICENSE_ENCRYPT_KEY = "ERVx43f9304gu30gjjsofp0-4lf0de%^"
)

type License struct {
	LicenseID         string `json:"license_id"`
	Signature         []byte `json:"signature"`
	OwnerName         string `json:"owner_name"`
	OwnerOrganization string `json:"owner_organization"`
	OwnerEmail        string `json:"owner_email"`
	ExpireDate        uint64 `json:"expire_date"`
	MaxActiveUsers    int    `json:"max_active_users"`
}

type LicenseManager struct {
	license *License
}

func NewLicenceManager() *LicenseManager {
	lm := new(LicenseManager)
	lm.license = new(License)
	return lm
}

// Load reads the appropriate key from SYSTEM_INTERNAL collection and unmarshal it.
func (m *LicenseManager) Load() bool {
	// _funcName

	// removed LOG Function

	r := MS{}
	if err := _MongoDB.C(COLLECTION_SYSTEM_INTERNAL).FindId("license_key").One(r); err != nil {
		_Log.Warn(err.Error())
		return false
	}
	licenseKey := r["value"]
	if len(licenseKey) == 0 {
		return false
	}
	jsonLicense := Decrypt(LICENSE_ENCRYPT_KEY, licenseKey)
	if err := json.Unmarshal([]byte(jsonLicense), m.license); err != nil {
		_Log.Warn(err.Error())
		return false
	}
	return true
}

func (m *LicenseManager) IsExpired() bool {
	currentTime := time.Now()
	expireTime := time.Unix(int64(m.license.ExpireDate/1000), 0)
	if currentTime.After(expireTime) {
		return true
	}
	return false
}

func (m *LicenseManager) Get() *License {
	return m.license
}

func (m *LicenseManager) Set(licenseKey string) {
	// _funcName
	if _, err := _MongoDB.C(COLLECTION_SYSTEM_INTERNAL).UpsertId(
		"license_key",
		bson.M{"$set": bson.M{
			"value": licenseKey,
		}},
	); err != nil {
		_Log.Warn(err.Error())
	}
}
