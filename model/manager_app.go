package nested

import "github.com/globalsign/mgo/bson"

type AppManager struct{}
type App struct {
	ID           string `bson:"_id" json:"_id"`
	Name         string `bson:"app_name" json:"app_name"`
	Homepage     string `bson:"homepage" json:"homepage"`
	CallbackURL  string `bson:"callback_url" json:"callback_url"`
	IconLargeURL string `bson:"icon_large_url" json:"icon_large_url"`
	IconSmallURL string `bson:"icon_small_url" json:"icon_small_url"`
	Developer    string `bson:"developer" json:"developer"`
}

var (
	_AppStore = App{
		ID:           "_appstore",
		Name:         "Nested Store",
		Homepage:     "https://store.nested.me",
		CallbackURL:  "",
		IconLargeURL: "https://store.nested.me/public/assets/icons/App_Store_32.svg",
		IconSmallURL: "https://store.nested.me/public/assets/icons/App_Store_32.svg",
		Developer:    "Ronak Software Group",
	}
)

func NewAppManager() *AppManager {
	return new(AppManager)
}

// Register register the app info as a verified app to be used by members of the Nested instance
func (m *AppManager) Register(appID, appName, homepage, callbackURL, developer, iconSmall, iconLarge string) bool {
	a := App{
		ID:           appID,
		Name:         appName,
		Homepage:     homepage,
		CallbackURL:  callbackURL,
		Developer:    developer,
		IconLargeURL: iconLarge,
		IconSmallURL: iconSmall,
	}
	if appID == _AppStore.ID {
		return false
	}
	if err := _MongoDB.C(global.COLLECTION_APPS).Insert(a); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

// UnRegister removes the app from the verified apps list
func (m *AppManager) UnRegister(appID string) bool {
	if err := _MongoDB.C(global.COLLECTION_APPS).RemoveId(appID); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

// GetByID returns a pointer to App or nil if it does not found any app in the collection
func (m *AppManager) GetByID(appID string) *App {
	app := new(App)
	if appID == _AppStore.ID {
		return &_AppStore
	}
	if err := _MongoDB.C(global.COLLECTION_APPS).FindId(appID).One(app); err != nil {
		log.Warn(err.Error())
		return nil
	}
	return app
}

// GetManyByIDs returns an array of Apps
func (m *AppManager) GetManyByIDs(appIDs []string) []App {
	apps := make([]App, 0, len(appIDs))
	if err := _MongoDB.C(global.COLLECTION_APPS).Find(
		bson.M{"_id": bson.M{"$in": appIDs}},
	).One(&apps); err != nil {
		log.Warn(err.Error())
	}
	return apps
}

// ExpireTokens remove all the AppTokens assigned to the appID
func (m *AppManager) ExpireTokens(appID string) {
	if _, err := _MongoDB.C(global.COLLECTION_TOKENS_APPS).RemoveAll(
		bson.M{"app_id": appID},
	); err != nil {
		log.Warn(err.Error())
	}
}

// Exists returns TRUE if appID has been registered with the system otherwise returns FALSE
func (m *AppManager) Exists(appID string) bool {
	if appID == _AppStore.ID {
		return true
	}
	if n, err := _MongoDB.C(global.COLLECTION_APPS).FindId(appID).Count(); err != nil {
		log.Warn(err.Error())
		return false
	} else if n > 0 {
		return true
	}
	return false
}
