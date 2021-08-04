package nested

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/log"
	"regexp"
	"strings"

	"github.com/globalsign/mgo/bson"
	"github.com/gomodule/redigo/redis"
)

const (
	PLACE_ACCESS_WRITE_POST     string = "WR"
	PLACE_ACCESS_ADD_MEMBERS    string = "AM"
	PLACE_ACCESS_ADD_PLACE      string = "AP"
	PLACE_ACCESS_READ_POST      string = "RD"
	PLACE_ACCESS_SEE_MEMBERS    string = "SM"
	PLACE_ACCESS_SEE_PLACE      string = "SP"
	PLACE_ACCESS_REMOVE_MEMBERS string = "RM"
	PLACE_ACCESS_REMOVE_PLACE   string = "RP"
	PLACE_ACCESS_REMOVE_POST    string = "D"
	PLACE_ACCESS_CONTROL        string = "C"
)
const (
	PLACE_POLICY_NOONE    PolicyGroup = "noone"
	PLACE_POLICY_CREATORS PolicyGroup = "creators"
	PLACE_POLICY_EVERYONE PolicyGroup = "everyone"
)
const (
	PLACE_RECEPTIVE_OFF      PrivacyReceptive = "off"
	PLACE_RECEPTIVE_INTERNAL PrivacyReceptive = "internal"
	PLACE_RECEPTIVE_EXTERNAL PrivacyReceptive = "external"
)
const (
	MEMBER_TYPE_ALL        string = "all"
	MEMBER_TYPE_CREATOR    string = "creator"
	MEMBER_TYPE_KEY_HOLDER string = "key_holder"
)
const (
	PLACE_TYPE_PERSONAL string = "personal"
	PLACE_TYPE_SHARED   string = "shared"
)
const (
	PLACE_COUNTER_CREATORS          string = "creators"
	PLACE_COUNTER_KEYHOLDERS        string = "key_holders"
	PLACE_COUNTER_CHILDREN          string = "childs"
	PLACE_COUNTER_UNLOCKED_CHILDREN string = "unlocked_childs"
	PLACE_COUNTER_QUOTA             string = "size"
	PLACE_COUNTER_POSTS             string = "posts"
)

type PrivacyReceptive string
type PolicyGroup string
type PlaceAccess MB

type PlaceCreateRequest struct {
	ID            string
	AccountID     string
	Name          string
	Description   string
	GrandParentID string
	Privacy       PlacePrivacy
	Policy        PlacePolicy
	Picture       Picture
}

type DefaulPlace struct {
	ID      bson.ObjectId `json:"_id" bson:"_id"`
	PlaceID string        `json:"place_id" bson:"place_id"`
}

// Place Manager and Methods
type PlaceManager struct{}

func NewPlaceManager() *PlaceManager {
	return new(PlaceManager)
}

func (pm *PlaceManager) readFromCache(placeID string) *Place {
	place := new(Place)
	c := _Cache.Pool.Get()
	defer c.Close()
	keyID := fmt.Sprintf("place:gob:%s", placeID)
	if gobPlace, err := redis.Bytes(c.Do("GET", keyID)); err != nil {
		if err := _MongoDB.C(global.COLLECTION_PLACES).FindId(placeID).One(place); err != nil {
			log.Warn(err.Error())
			return nil
		}
		gobPlace := new(bytes.Buffer)
		if err := gob.NewEncoder(gobPlace).Encode(place); err == nil {
			c.Do("SETEX", keyID, global.CACHE_LIFETIME, gobPlace.Bytes())
		}
		return place
	} else if err := gob.NewDecoder(bytes.NewBuffer(gobPlace)).Decode(place); err == nil {
		return place
	}
	return nil
}

func (pm *PlaceManager) readMultiFromCache(placeIDs []string) []Place {
	places := make([]Place, 0, len(placeIDs))
	c := _Cache.Pool.Get()
	defer c.Close()
	for _, placeID := range placeIDs {
		keyID := fmt.Sprintf("place:gob:%s", placeID)
		c.Send("GET", keyID)
	}
	c.Flush()
	for _, placeID := range placeIDs {
		if gobPlace, err := redis.Bytes(c.Receive()); err == nil {
			place := new(Place)
			if err := gob.NewDecoder(bytes.NewBuffer(gobPlace)).Decode(place); err == nil {
				places = append(places, *place)
			}
		} else {
			if place := _Manager.Place.readFromCache(placeID); place != nil {
				places = append(places, *place)
			}
		}
	}
	return places
}

func (pm *PlaceManager) removeCache(placeID string) bool {
	c := _Cache.Pool.Get()
	defer c.Close()
	keyID := fmt.Sprintf("place:gob:%s", placeID)
	c.Do("DEL", keyID)
	return true
}

// AddKeyHolder add the accountID to the list of placeID key holders, if he/she was not
// a member of that place before (i.e. he/she is not creator or key holder of the placeID)
func (pm *PlaceManager) AddKeyholder(placeID, accountID string) *PlaceManager {
	defer _Manager.Place.removeCache(placeID)
	defer _Manager.Account.removeCache(accountID)

	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	place := _Manager.Place.GetByID(placeID, nil)
	account := _Manager.Account.GetByID(accountID, nil)

	// Update PLACES collection
	if err := db.C(global.COLLECTION_PLACES).Update(
		bson.M{
			"_id":         placeID,
			"creators":    bson.M{"$ne": accountID},
			"key_holders": bson.M{"$ne": accountID},
		},
		bson.M{
			"$addToSet": bson.M{"key_holders": accountID},
			"$inc":      bson.M{"counters.key_holders": 1},
		},
	); err != nil {
		log.Warn(err.Error())
	}

	// Update ACCOUNTS collection
	if err := db.C(global.COLLECTION_ACCOUNTS).Update(
		bson.M{"_id": accountID},
		bson.M{"$addToSet": bson.M{"access_places": placeID}},
	); err != nil {
		log.Warn(err.Error())
	}

	// Update POSTS.READS.COUNTERS collection
	no_unreads, _ := db.C(global.COLLECTION_POSTS_READS).Find(
		bson.M{
			"account_id": accountID,
			"place_id":   placeID,
			"read":       false,
		},
	).Count()
	db.C(global.COLLECTION_POSTS_READS_COUNTERS).Upsert(
		bson.M{"account_id": accountID, "place_id": placeID},
		bson.M{"$set": bson.M{"no_unreads": no_unreads}},
	)
	if place.IsGrandPlace() {
		for _, unlockedPlaceID := range place.UnlockedChildrenIDs {
			no_unreads, _ := db.C(global.COLLECTION_POSTS_READS).Find(
				bson.M{
					"account_id": accountID,
					"place_id":   unlockedPlaceID,
					"read":       false,
				},
			).Count()
			db.C(global.COLLECTION_POSTS_READS_COUNTERS).Upsert(
				bson.M{"account_id": accountID, "place_id": unlockedPlaceID},
				bson.M{"$set": bson.M{"no_unreads": no_unreads}},
			)
		}
	}

	// Increment PlaceConnection counter of the accountID and placeID by one
	_Manager.Account.UpdatePlaceConnection(accountID, []string{placeID}, 1)

	// Increment AccountConnection between accountID and other members of placeID by one
	_Manager.Account.UpdateAccountConnection(accountID, place.GetMemberIDs(), 1)

	// Updates the place activity
	_Manager.PlaceActivity.MemberJoin(accountID, placeID, "")

	// Send the hook event
	_Manager.Hook.chEvents <- NewMemberEvent{
		PlaceID:         place.ID,
		MemberID:        account.ID,
		MemberName:      account.FullName,
		ProfilePicSmall: string(account.Picture.X32),
		ProfilePicLarge: string(account.Picture.X128),
	}

	return pm
}

//	Available returns true if the placeID is available to be created. It means that this placeID
//	is not reserved or not already taken.
func (pm *PlaceManager) Available(placeID string) bool {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	dotPosition := strings.LastIndex(placeID, ".")
	if dotPosition == -1 {
		if matched, err := regexp.MatchString(global.DEFAULT_REGEX_GRANDPLACE_ID, placeID); err != nil || !matched {
			return false
		}
	} else {
		localPlaceID := string(placeID[dotPosition+1:])
		if matched, err := regexp.MatchString(global.DEFAULT_REGEX_PLACE_ID, localPlaceID); err != nil || !matched {
			return false
		}
	}

	if n, _ := db.C(global.COLLECTION_PLACES).FindId(placeID).Count(); n > 0 {
		return false
	}

	if n, _ := db.C(global.COLLECTION_SYS_RESERVED_WORDS).Find(bson.M{"word": placeID}).Count(); n > 0 {
		return false
	}

	return true
}

//	CountUnreadPosts counts all the unread posts for accountID in all placeIDs
func (pm *PlaceManager) CountUnreadPosts(placeIDs []string, accountID string) int {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	c := 0
	r := struct {
		Count int `json:"no_unreads" bson:"no_unreads"`
	}{}

	iter := db.C(global.COLLECTION_POSTS_READS_COUNTERS).Find(
		bson.M{"account_id": accountID, "place_id": bson.M{"$in": placeIDs}},
	).Iter()
	defer iter.Close()
	for iter.Next(&r) {
		c += r.Count
	}
	return c
}

//	CreatePersonalPlace creates personal grand place and sub places.  The difference between this function and
func (pm *PlaceManager) CreatePersonalPlace(pcr PlaceCreateRequest) *Place {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	p := Place{
		ID:          pcr.ID,
		Type:        PLACE_TYPE_PERSONAL,
		Name:        pcr.Name,
		Description: pcr.Description,
		CreatedOn:   Timestamp(),
		Privacy:     pcr.Privacy,
	}

	// Initialize Place Policy and Privacy
	p.Policy.AddPost = PLACE_POLICY_CREATORS
	p.Policy.AddMember = PLACE_POLICY_NOONE
	p.Policy.AddPlace = PLACE_POLICY_CREATORS
	p.Privacy.Locked = true
	p.Privacy.Receptive = PLACE_RECEPTIVE_EXTERNAL

	if pcr.ID == pcr.GrandParentID {
		p.GrandParentID = p.ID
		p.Level = 0
		// Initialize Place Limits
		p.Limit.Creators = 1
		p.Limit.Keyholders = 0
		p.Limit.Children = DEFAULT_PLACE_MAX_CHILDREN
	} else if pcr.GrandParentID != "" {
		grandPlace := _Manager.Place.GetByID(pcr.GrandParentID, nil)
		p.Limit = grandPlace.Limit
		p.GrandParentID = pcr.GrandParentID
	}
	p.MainCreatorID = pcr.AccountID
	p.Picture = pcr.Picture

	if err := db.C(global.COLLECTION_PLACES).Insert(p); err != nil {
		log.Warn(err.Error())
		return nil
	} else if err = db.C(global.COLLECTION_PLACES).FindId(p.ID).One(&p); err != nil {
		log.Warn(err.Error())
		return nil
	}

	// Update System.Internal Counter
	if p.Level == 0 {
		_Manager.System.incrementCounter(MI{SYSTEM_COUNTERS_GRAND_PLACES: 1})
	} else {
		_Manager.System.incrementCounter(MI{SYSTEM_COUNTERS_LOCKED_PLACES: 1})
	}

	// add the timeline event
	_Manager.PlaceActivity.PlaceAdd(pcr.AccountID, pcr.ID)

	// create a group for members want notification for this place
	_Manager.Group.CreatePlaceGroup(pcr.ID, NOTIFICATION_GROUP)

	return &p
}

//	CreateGrandPlace, CreateLockedPlace and CreateUnlockedPlace are only in the privacy and policy settings
//	overrides. We used separate functions for creating different place for more code clarity and better
//	maintainability.
func (pm *PlaceManager) CreateGrandPlace(pcr PlaceCreateRequest) *Place {
	defer _Manager.Account.removeCache(pcr.AccountID)

	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	place := Place{
		ID:          pcr.ID,
		Type:        PLACE_TYPE_SHARED,
		Name:        pcr.Name,
		Description: pcr.Description,
		Policy:      pcr.Policy,
		Privacy:     pcr.Privacy,
		CreatedOn:   Timestamp(),
	}
	// Initialize Place Limits
	place.Limit.Creators = DEFAULT_PLACE_MAX_CREATORS
	place.Limit.Keyholders = DEFAULT_PLACE_MAX_KEYHOLDERS
	place.Limit.Children = DEFAULT_PLACE_MAX_CHILDREN
	// Initialize Place Policy and Privacy
	place.Privacy.Locked = true
	place.GrandParentID = pcr.ID
	place.Level = 0
	place.MainCreatorID = pcr.AccountID
	place.Picture = pcr.Picture

	if err := db.C(global.COLLECTION_PLACES).Insert(place); err != nil {
		log.Warn(err.Error())
		return nil
	} else if err = db.C(global.COLLECTION_PLACES).FindId(place.ID).One(&place); err != nil {
		log.Warn(err.Error())
		return nil
	}

	// Update System.Internal Counter
	_Manager.System.incrementCounter(MI{SYSTEM_COUNTERS_GRAND_PLACES: 1})

	// add timeline event
	_Manager.PlaceActivity.PlaceAdd(pcr.AccountID, pcr.ID)

	// create notification group for members who want to get notification
	_Manager.Group.CreatePlaceGroup(pcr.ID, NOTIFICATION_GROUP)
	return &place
}

//	CreateGrandPlace, CreateLockedPlace and CreateUnlockedPlace are only in the privacy and policy settings
//	overrides. We used separate functions for creating different place for more code clarity and better
//	maintainability.
func (pm *PlaceManager) CreateLockedPlace(pcr PlaceCreateRequest) *Place {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	p := Place{
		ID:          pcr.ID,
		Type:        PLACE_TYPE_SHARED,
		Name:        pcr.Name,
		Description: pcr.Description,
		Policy:      pcr.Policy,
		Privacy:     pcr.Privacy,
		CreatedOn:   Timestamp(),
	}
	defer _Manager.Place.removeCache(p.GetParentID())

	grandParentPlace := _Manager.Place.GetByID(pcr.GrandParentID, nil)
	parentPlace := _Manager.Place.GetByID(p.GetParentID(), nil)
	// Initialize Place Limits
	p.Limit = grandParentPlace.Limit

	// Initialize Place Policy and Privacy
	p.Privacy.Locked = true
	p.GrandParentID = pcr.GrandParentID
	p.MainCreatorID = pcr.AccountID
	p.Picture = pcr.Picture
	p.Level = parentPlace.Level + 1

	if err := db.C(global.COLLECTION_PLACES).Insert(p); err != nil {
		log.Warn(err.Error())
		return nil
	} else if db.C(global.COLLECTION_PLACES).FindId(p.ID).One(&p); err != nil {
		log.Warn(err.Error())
		return nil
	}

	// update parent counters
	if err := db.C(global.COLLECTION_PLACES).UpdateId(
		p.GetParentID(),
		bson.M{"$inc": bson.M{"counters.childs": 1}},
	); err != nil {
		log.Warn(err.Error())
	}

	// Update System.Internal Counter
	_Manager.System.incrementCounter(MI{SYSTEM_COUNTERS_LOCKED_PLACES: 1})

	// add timeline event
	_Manager.PlaceActivity.PlaceAdd(pcr.AccountID, pcr.ID)

	// create a group to hold accounts who want notification for this place
	_Manager.Group.CreatePlaceGroup(pcr.ID, NOTIFICATION_GROUP)
	return &p
}

//	CreateGrandPlace, CreateLockedPlace and CreateUnlockedPlace are only in the privacy and policy settings
//	overrides. We used separate functions for creating different place for more code clarity and better
//	maintainability.
func (pm *PlaceManager) CreateUnlockedPlace(pcr PlaceCreateRequest) *Place {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	p := Place{
		ID:          pcr.ID,
		Type:        PLACE_TYPE_SHARED,
		Name:        pcr.Name,
		Description: pcr.Description,
		CreatedOn:   Timestamp(),
	}

	if p.GetParentID() != pcr.GrandParentID {
		return nil
	}
	defer _Manager.Place.removeCache(p.GetParentID())

	grandPlace := _Manager.Place.GetByID(pcr.GrandParentID, nil)

	// Initialize Place Limits
	p.Limit.Creators = grandPlace.Limit.Creators
	p.Limit.Keyholders = grandPlace.Limit.Keyholders
	p.Limit.Children = 0

	// Initialize Place Policy and Privacy
	p.Privacy.Locked = false
	p.Privacy.Receptive = PLACE_RECEPTIVE_OFF
	p.Policy.AddMember = PLACE_POLICY_CREATORS
	p.Policy.AddPlace = PLACE_POLICY_NOONE
	p.Policy.AddPost = PLACE_POLICY_EVERYONE
	p.GrandParentID = pcr.GrandParentID
	p.MainCreatorID = pcr.AccountID
	p.Picture = pcr.Picture
	p.Level = 1

	if err := db.C(global.COLLECTION_PLACES).Insert(p); err != nil {
		log.Warn(err.Error())
		return nil
	} else if db.C(global.COLLECTION_PLACES).FindId(p.ID).One(&p); err != nil {
		log.Warn(err.Error())
		return nil
	}

	// Update System.Internal Counter
	_Manager.System.incrementCounter(MI{SYSTEM_COUNTERS_UNLOCKED_PLACES: 1})

	// update parent counters
	db.C(global.COLLECTION_PLACES).UpdateId(
		p.GrandParentID,
		bson.M{
			"$inc": bson.M{
				"counters.childs":          1,
				"counters.unlocked_childs": 1,
			},
			"$addToSet": bson.M{"unlocked_childs": pcr.ID},
		},
	)
	// add timeline event
	_Manager.PlaceActivity.PlaceAdd(pcr.AccountID, pcr.ID)

	// create a group to hold accounts who want notification for this place
	_Manager.Group.CreatePlaceGroup(pcr.ID, NOTIFICATION_GROUP)
	return &p

}

//	Demote change user level from creator to key holder
func (pm *PlaceManager) Demote(placeID, accountID string) *PlaceManager {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	defer _Manager.Place.removeCache(placeID)
	defer _Manager.Account.removeCache(accountID)
	// Update PLACES collection
	if err := db.C(global.COLLECTION_PLACES).Update(
		bson.M{
			"_id":         placeID,
			"key_holders": bson.M{"$ne": accountID},
			"creators":    accountID,
		},
		bson.M{
			"$addToSet": bson.M{"key_holders": accountID},
			"$pull":     bson.M{"creators": accountID},
			"$inc": bson.M{
				"counters.key_holders": 1,
				"counters.creators":    -1,
			},
		},
	); err != nil {
		log.Warn(err.Error())
	}
	return pm
}

//	Exists returns true if place is already exists, this function is opposite of Available
func (pm *PlaceManager) Exists(placeID string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	n, _ := db.C(global.COLLECTION_PLACES).FindId(placeID).Count()

	return n > 0
}

//	GetByID returns a pointer to a place identified by placeID.
func (pm *PlaceManager) GetByID(placeID string, pj M) *Place {
	return _Manager.Place.readFromCache(placeID)
}

//	GetPlacesByIDs returns an array of places identified by placeIDs. Only found places will be returned
//	and the rest will be silently ignored
func (pm *PlaceManager) GetPlacesByIDs(placeIDs []string) []Place {
	return _Manager.Place.readMultiFromCache(placeIDs)
}

// GetGrandParentIDs accepts an array of placeIDs and returns an array of their grand place ids.
func (pm *PlaceManager) GetGrandParentIDs(placeIDs []string) []string {
	var res []string
	for _, v := range placeIDs {
		res = append(res, strings.Split(v, ".")[0])
	}

	return res
}

// GetParentID returns the parent's id of the placeID
func (pm *PlaceManager) GetParentID(placeID string) string {
	return string(placeID[:strings.LastIndex(placeID, ".")])
}

//	IncrementCounter increase/decrease place counters supported counters are
//	1. PLACE_COUNTER_CHILDREN
//	2. PLACE_COUNTER_UNLOCKED_CHILDREN
//	3. PLACE_COUNTER_CREATORS
//	4. PLACE_COUNTER_KEYHOLDERS
//	5. PLACE_COUNTER_POSTS
//	6. PLACE_COUNTER_QUOTA
func (pm *PlaceManager) IncrementCounter(placeIDs []string, counterName string, c int) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	switch counterName {
	case PLACE_COUNTER_CHILDREN, PLACE_COUNTER_UNLOCKED_CHILDREN,
		PLACE_COUNTER_CREATORS, PLACE_COUNTER_KEYHOLDERS,
		PLACE_COUNTER_POSTS, PLACE_COUNTER_QUOTA:
		keyName := fmt.Sprintf("counters.%s", counterName)
		if err := db.C(global.COLLECTION_PLACES).Update(
			bson.M{"_id": bson.M{"$in": placeIDs}},
			bson.M{"$inc": bson.M{keyName: c}},
		); err != nil {
			log.Warn(err.Error())
			return false
		}
	}
	return true
}

//	IsSubPlace returns TRUE if subPlaceID is a sub-place of placeID. It will returns TRUE even if
//	subPlaceID is not a direct child of the placeID.
func (pm *PlaceManager) IsSubPlace(placeID, subPlaceID string) bool {
	di := strings.Index(subPlaceID, ".")
	pi := strings.Index(subPlaceID, placeID+".")
	if placeID != subPlaceID && di != -1 && pi == 0 {
		return true
	}
	return false
}

// Promote promotes the accountID in the placeID from keyholder to creator
func (pm *PlaceManager) Promote(placeID, accountID string) *PlaceManager {
	defer _Manager.Place.removeCache(placeID)
	defer _Manager.Account.removeCache(accountID)

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	// Update PLACES collection
	if err := db.C(global.COLLECTION_PLACES).Update(
		bson.M{
			"_id":         placeID,
			"creators":    bson.M{"$ne": accountID},
			"key_holders": accountID,
		},
		bson.M{
			"$addToSet": bson.M{"creators": accountID},
			"$pull":     bson.M{"key_holders": accountID},
			"$inc": bson.M{
				"counters.creators":    1,
				"counters.key_holders": -1,
			},
		},
	); err != nil {
		log.Warn(err.Error())
		return nil
	}
	_Manager.Account.UpdatePlaceConnection(accountID, []string{placeID}, 1)
	return pm
}

// bookmarkPost pins postID to one of the pinned posts of placeID
func (pm *PlaceManager) PinPost(placeID string, postID bson.ObjectId) bool {
	defer _Manager.Place.removeCache(placeID)

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_PLACES).Update(
		bson.M{"_id": placeID},
		bson.M{
			"$push": bson.M{
				"pinned_posts": bson.M{
					"$each":  []bson.ObjectId{postID},
					"$slice": 1,
				},
			},
		},
	); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

// UnpinPost unpins postID from the placeID
func (pm *PlaceManager) UnpinPost(placeID string, postID bson.ObjectId) bool {
	defer _Manager.Place.removeCache(placeID)

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_PLACES).Update(
		bson.M{"_id": placeID},
		bson.M{"$pull": bson.M{"pinned_posts": postID}},
	); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

// Remove deletes the place forever and all the posts and activities of that place will be gone.
// also all the members will be removed from the place
func (pm *PlaceManager) Remove(placeID string, accountID string) bool {
	defer _Manager.Place.removeCache(placeID)

	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	place := pm.GetByID(placeID, nil)

	// Places with children cannot be removed
	if place.HasChild() {
		return false
	}

	// Update parent place if it is not a grand place
	if !place.IsGrandPlace() {
		defer _Manager.Place.removeCache(place.GrandParentID)
		if place.Level > 1 {
			defer _Manager.Place.removeCache(place.GetParentID())
		}
		if err := db.C(global.COLLECTION_PLACES).UpdateId(
			place.GetParentID(),
			bson.M{"$inc": bson.M{"counters.childs": -1}},
		); err != nil {
			log.Warn(err.Error())
		}
	}

	// Update grand place if place is OPEN
	if !place.Privacy.Locked {
		if err := db.C(global.COLLECTION_PLACES).UpdateId(
			place.GrandParentID,
			bson.M{
				"$pull": bson.M{"unlocked_childs": placeID},
				"$inc":  bson.M{"counters.unlocked_childs": -1},
			},
		); err != nil {
			log.Warn(err.Error())
		}
	}

	// Remove All Members of the place
	pm.RemoveAllMembers(placeID)

	// Remove all posts
	iter := db.C(global.COLLECTION_POSTS).Find(bson.M{"places": placeID}).Select(bson.M{"_id": 1}).Iter()
	defer iter.Close()
	post := new(Post)
	for iter.Next(post) {
		_Manager.Post.Remove(accountID, post.ID, placeID)
		_Manager.PlaceActivity.PostRemove(accountID, placeID, post.ID)
	}

	// Remove the place from PLACES collection
	if err := db.C(global.COLLECTION_PLACES).RemoveId(placeID); err != nil {
		log.Warn(err.Error())
		return false
	}

	// Remove all the timeline activities related to the placeID
	_Manager.PlaceActivity.PlaceRemove(placeID)

	// Update System.Internal Counter
	if place.Level == 0 {
		_Manager.System.incrementCounter(MI{SYSTEM_COUNTERS_GRAND_PLACES: -1})
	} else if place.Privacy.Locked {
		_Manager.System.incrementCounter(MI{SYSTEM_COUNTERS_LOCKED_PLACES: -1})
	} else {
		_Manager.System.incrementCounter(MI{SYSTEM_COUNTERS_UNLOCKED_PLACES: -1})
	}

	return true
}

func (pm *PlaceManager) RemoveAllMembers(placeID string) {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	place := _Manager.Place.GetByID(placeID, nil)
	memberIDs := place.GetMemberIDs()
	_Manager.Account.removeMultiFromCache(memberIDs)
	if _, err := db.C(global.COLLECTION_ACCOUNTS).UpdateAll(
		bson.M{"_id": bson.M{"$in": memberIDs}},
		bson.M{"$pull": bson.M{"access_places": placeID}},
	); err != nil {
		log.Warn(err.Error())
	}

	if _, err := db.C(global.COLLECTION_ACCOUNTS).UpdateAll(
		bson.M{"bookmarked_places": placeID},
		bson.M{"$pull": bson.M{"bookmarked_places": placeID}},
	); err != nil {
		log.Warn(err.Error())
	}

}

func (pm *PlaceManager) RemoveKeyHolder(placeID, accountID, actorID string) *PlaceManager {
	defer _Manager.Place.removeCache(placeID)
	defer _Manager.Account.removeCache(accountID)

	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	place := _Manager.Place.GetByID(placeID, nil)
	// update PLACES collection
	if err := db.C(global.COLLECTION_PLACES).Update(
		bson.M{
			"_id":         placeID,
			"key_holders": accountID,
			"creators":    bson.M{"$ne": accountID},
		},
		bson.M{
			"$pull": bson.M{"key_holders": accountID},
			"$inc":  bson.M{"counters.key_holders": -1},
		},
	); err != nil {
		log.Warn(err.Error())
	}

	// Update ACCOUNTS collection
	// remove the place from account's document
	if err := db.C(global.COLLECTION_ACCOUNTS).Update(
		bson.M{"_id": accountID},
		bson.M{"$pull": bson.M{
			"access_places":     placeID,
			"bookmarked_places": placeID,
			"recently_visited":  placeID,
		}},
	); err != nil {
		log.Warn(err.Error())
	}
	if _, err := db.C(global.COLLECTION_POSTS_READS).UpdateAll(
		bson.M{
			"account_id": accountID,
			"place_id":   placeID,
			"read":       false,
		},
		bson.M{"$set": bson.M{"read": true}},
	); err != nil {
		log.Warn(err.Error())
	}

	// Update POSTS.READS.COUNTERS collection
	if err := db.C(global.COLLECTION_POSTS_READS_COUNTERS).Remove(
		bson.M{
			"account_id": accountID,
			"place_id":   placeID,
		},
	); err != nil {
		log.Warn(err.Error())
	}

	if place.IsGrandPlace() {
		for _, unlockedPlaceID := range place.UnlockedChildrenIDs {
			if err := db.C(global.COLLECTION_POSTS_READS_COUNTERS).Remove(
				bson.M{
					"account_id": accountID,
					"place_id":   unlockedPlaceID,
				},
			); err != nil {
				log.Warn(err.Error())
			}
		}
	}

	// Remove the accountID from Notification Group of the place
	_Manager.Group.RemoveItems(place.Groups[NOTIFICATION_GROUP], []string{accountID})

	// Remove the
	_Manager.PlaceActivity.MemberRemove(actorID, placeID, accountID, "")

	return pm
}

func (pm *PlaceManager) RemoveCreator(placeID, accountID, actorID string) *PlaceManager {
	pm.Demote(placeID, accountID)
	pm.RemoveKeyHolder(placeID, accountID, actorID)
	return pm
}

func (pm *PlaceManager) SetPicture(placeID string, pic Picture) {
	defer _Manager.Place.removeCache(placeID)

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	if err := db.C(global.COLLECTION_PLACES).UpdateId(
		placeID,
		bson.M{"$set": bson.M{"picture": pic}},
	); err != nil {
		log.Warn(err.Error())
	}
}

func (pm *PlaceManager) Update(placeID string, placeUpdateRequest M) bool {
	defer _Manager.Place.removeCache(placeID)

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	for k := range placeUpdateRequest {
		switch k {
		case "name", "description", "privacy.search", "privacy.receptive",
			"policy.add_post", "policy.add_member", "policy.add_place":
		default:
			delete(placeUpdateRequest, k)
		}
	}
	if len(placeUpdateRequest) > 0 {
		if err := db.C(global.COLLECTION_PLACES).UpdateId(placeID, bson.M{"$set": placeUpdateRequest}); err != nil {
			log.Warn(err.Error())
			return false
		}
	}
	return true
}

func (pm *PlaceManager) UpdateLimits(placeID string, limits MI) bool {
	defer _Manager.Place.removeCache(placeID)

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	m := MI{}
	for limitKey, limitValue := range limits {
		switch limitKey {
		case "limits.key_holders":
			m[limitKey] = ClampInteger(limitValue, global.SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_LL, global.SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_UL)
		case "limits.creators":
			m[limitKey] = ClampInteger(limitValue, global.SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_LL, global.SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_UL)
		case "limits.childs":
			m[limitKey] = ClampInteger(limitValue, global.SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_LL, global.SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_UL)
		case "limits.size":
			m[limitKey] = limitValue
		}
	}
	if len(m) == 0 {
		return false
	}
	if _, err := db.C(global.COLLECTION_PLACES).UpdateAll(
		bson.M{"grand_parent_id": placeID},
		bson.M{"$set": m},
	); err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

func (pm *PlaceManager) GetPlaceBlockedAddresses(placeID string) []string {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()
	blockedAddresses := BlockedAddresses{}
	if err := db.C(global.COLLECTION_PLACES_BLOCKED_ADDRESSES).FindId(placeID).One(&blockedAddresses); err != nil {
		log.Warn(err.Error())
		return nil
	}
	return blockedAddresses.Addresses
}

func (pm *PlaceManager) AddToBlacklist(placeID string, addresses []string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	_, err := db.C(global.COLLECTION_PLACES_BLOCKED_ADDRESSES).UpsertId(
		placeID,
		bson.M{"$addToSet": bson.M{"addresses": bson.M{"$each": addresses}}},
	)
	if err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

func (pm *PlaceManager) RemoveFromBlacklist(placeID string, addresses []string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	err := db.C(global.COLLECTION_PLACES_BLOCKED_ADDRESSES).UpdateId(
		placeID,
		bson.M{"$pull": bson.M{"addresses": bson.M{"$in": addresses}}},
	)
	if err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

func (pm *PlaceManager) IsBlocked(placeID, address string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	n, err := db.C(global.COLLECTION_PLACES_BLOCKED_ADDRESSES).FindId(placeID).Select(
		bson.M{"addresses": address},
	).Count()
	if err != nil {
		log.Warn(err.Error())
		return false
	}
	return n > 0
}

//	AddDefaultPlaces adds placeIDs to the initial place list
func (pm *PlaceManager) AddDefaultPlaces(placeIDs []string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()

	bulk := db.C(global.COLLECTION_PLACES_DEFAULT).Bulk()
	bulk.Unordered()
	for _, id := range placeIDs {
		d := DefaulPlace{
			ID:      bson.NewObjectId(),
			PlaceID: id,
		}
		bulk.Upsert(bson.M{"place_id": id}, d)
	}
	_, err := bulk.Run()
	if err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

//	GetDefaultPlacesWithPagination gets initial placeIDs
func (pm *PlaceManager) GetDefaultPlacesWithPagination(pg Pagination) ([]string, int) {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()
	defaultPlaces := make([]DefaulPlace, 0, pg.GetLimit())
	ids := make([]string, 0, pg.GetLimit())
	err := db.C(global.COLLECTION_PLACES_DEFAULT).Find(nil).Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&defaultPlaces)
	if err != nil {
		log.Warn(err.Error())
		return nil, 0
	}
	for _, placeID := range defaultPlaces {
		ids = append(ids, placeID.PlaceID)
	}
	n, err := db.C(global.COLLECTION_PLACES_DEFAULT).Find(nil).Count()
	if err != nil {
		log.Warn(err.Error())
		return nil, 0
	}
	return ids, n
}

//	GetDefaultPlaces gets default placeIDs
func (pm *PlaceManager) GetDefaultPlaces() []string {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()
	var defaultPlaces []DefaulPlace
	err := db.C(global.COLLECTION_PLACES_DEFAULT).Find(nil).All(&defaultPlaces)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}
	ids := make([]string, 0, len(defaultPlaces))
	for _, placeID := range defaultPlaces {
		ids = append(ids, placeID.PlaceID)
	}
	return ids
}

//	RemoveDefaultPlaces removes default placeIDs
func (pm *PlaceManager) RemoveDefaultPlaces(placeIDs []string) bool {
	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DB_NAME)
	defer dbSession.Close()
	err := db.C(global.COLLECTION_PLACES_DEFAULT).Remove(bson.M{"place_id": bson.M{"$in": placeIDs}})
	if err != nil {
		log.Warn(err.Error())
		return false
	}
	return true
}

type Place struct {
	ID                  string          `json:"_id" bson:"_id"`
	Type                string          `json:"type" bson:"type"`
	Name                string          `json:"name,omitempty" bson:"name"`
	Description         string          `json:"description" bson:"description"`
	GrandParentID       string          `json:"grand_parent_id" bson:"grand_parent_id"`
	Privacy             PlacePrivacy    `json:"privacy" bson:"privacy"`
	Policy              PlacePolicy     `json:"policy" bson:"policy"`
	Level               int             `json:"level" bson:"level"`
	CreatedOn           uint64          `json:"created_on" bson:"created_on"`
	MainCreatorID       string          `json:"created_by" bson:"created_by"`
	CreatorIDs          []string        `json:"creators" bson:"creators"`
	KeyholderIDs        []string        `json:"key_holders" bson:"key_holders"`
	UnlockedChildrenIDs []string        `json:"unlocked_childs" bson:"unlocked_childs"`
	Limit               PlaceLimit      `json:"limits" bson:"limits"`
	Counter             PlaceCounter    `json:"counters" bson:"counters"`
	Picture             Picture         `json:"picture" bson:"picture"`
	Groups              MS              `json:"groups" bson:"groups"`
	PinnedPosts         []bson.ObjectId `json:"pinned_posts" bson:"pinned_posts"`
}
type PlacePrivacy struct {
	Locked    bool             `json:"locked" bson:"locked"`
	Search    bool             `json:"search" bson:"search"`
	Receptive PrivacyReceptive `json:"receptive" bson:"receptive"`
}
type PlacePolicy struct {
	AddPost   PolicyGroup `json:"add_post" bson:"add_post"`
	AddPlace  PolicyGroup `json:"add_place" bson:"add_place"`
	AddMember PolicyGroup `json:"add_member" bson:"add_member"`
}
type PlaceCounter struct {
	Creators         int `json:"creators" bson:"creators"`
	Keyholders       int `json:"key_holders" bson:"key_holders"`
	Children         int `json:"childs" bson:"childs"`
	Quota            int `json:"size" bson:"size"`
	Posts            int `json:"posts" bson:"posts"`
	UnlockedChildren int `json:"unlocked_childs" bson:"unlocked_childs"`
}
type PlaceLimit struct {
	Creators   int `json:"creators" bson:"creators"`
	Keyholders int `json:"key_holders" bson:"key_holders"`
	Children   int `json:"childs" bson:"childs"`
	Quota      int `json:"size" bson:"size"`
}

type BlockedAddresses struct {
	PlaceID   string   `json:"_id" bson:"_id"`
	Addresses []string `json:"addresses" bson:"addresses"`
}

func (p *Place) GetPrivacy() PlacePrivacy {
	return p.Privacy
}

func (p *Place) GetPolicy() PlacePolicy {
	return p.Policy
}

func (p *Place) GetParentID() string {
	ldi := strings.LastIndex(p.ID, ".")
	if ldi == -1 {
		p.GrandParentID = p.ID
		return ""
	} else {
		return string(p.ID[:ldi])
	}
}

func (p *Place) GetGrandParent() *Place {
	grandParent := _Manager.Place.GetByID(p.GrandParentID, nil)
	return grandParent
}

func (p *Place) GetMemberIDs() []string {
	return append(p.KeyholderIDs, p.CreatorIDs...)
}

func (p *Place) IsGrandPlace() bool {
	if p.GrandParentID == p.ID {
		return true
	}
	return false
}

func (p *Place) IsPersonal() bool {
	if p.Type == PLACE_TYPE_PERSONAL {
		return true
	}
	return false
}

func (p *Place) HasChild() bool {
	noc := p.Counter.Children + p.Counter.UnlockedChildren
	if noc > 0 {
		return true
	}
	return false
}

func (p *Place) HasChildLimit() bool {
	if p.Counter.Children < p.Limit.Children {
		return false
	}
	return true
}

func (p *Place) HasKeyholderLimit() bool {
	if p.Counter.Keyholders < p.Limit.Keyholders {
		return false
	}
	return true
}

func (p *Place) HasCreatorLimit() bool {
	if p.Counter.Creators < p.Limit.Creators {
		return false
	}
	return true
}

func (p *Place) IsCreator(accountID string) bool {


	for _, creatorID := range p.CreatorIDs {
		if creatorID == accountID {
			return true
		}
	}
	return false

}

func (p *Place) IsKeyholder(accountID string) bool {


	for _, keyholderID := range p.KeyholderIDs {
		if keyholderID == accountID {
			return true
		}
	}
	return false
}

func (p *Place) IsMember(accountID string) bool {


	for _, creatorID := range p.CreatorIDs {
		if creatorID == accountID {
			return true
		}
	}
	for _, keyholderID := range p.KeyholderIDs {
		if keyholderID == accountID {
			return true
		}
	}
	return false
}

func (p *Place) HasReadAccess(accountID string) bool {


	if p.IsMember(accountID) {
		return true
	} else if !p.Privacy.Locked && !p.IsGrandPlace() {
		gp := _Manager.Place.GetByID(p.GrandParentID, nil)
		if gp.IsMember(accountID) {
			return true
		}
	}
	return false
}

func (p *Place) HasWriteAccess(accountID string) bool {


	if p.IsMember(accountID) && p.Policy.AddPost == PLACE_POLICY_EVERYONE {
		return true
	} else if p.IsCreator(accountID) {
		return true
	}
	grandParent := p.GetGrandParent()
	igpm := grandParent.IsMember(accountID)
	switch p.Privacy.Receptive {
	case PLACE_RECEPTIVE_INTERNAL:
		if igpm {
			return true
		}
	case PLACE_RECEPTIVE_EXTERNAL:
		return true

	}
	return false
}

func (p *Place) GetAccess(accountID string) MB {


	acl := MB{}
	acl[PLACE_ACCESS_READ_POST] = false
	acl[PLACE_ACCESS_WRITE_POST] = false
	acl[PLACE_ACCESS_REMOVE_POST] = false
	acl[PLACE_ACCESS_ADD_PLACE] = false
	acl[PLACE_ACCESS_SEE_PLACE] = false
	acl[PLACE_ACCESS_REMOVE_PLACE] = false
	acl[PLACE_ACCESS_ADD_MEMBERS] = false
	acl[PLACE_ACCESS_REMOVE_MEMBERS] = false
	acl[PLACE_ACCESS_SEE_MEMBERS] = false
	acl[PLACE_ACCESS_CONTROL] = false

	if p.IsCreator(accountID) {
		if p.Type == PLACE_TYPE_PERSONAL {
			acl[PLACE_ACCESS_READ_POST] = true
			acl[PLACE_ACCESS_WRITE_POST] = true
			acl[PLACE_ACCESS_CONTROL] = true
			acl[PLACE_ACCESS_REMOVE_POST] = true
			acl[PLACE_ACCESS_REMOVE_MEMBERS] = true
			acl[PLACE_ACCESS_SEE_PLACE] = true
			acl[PLACE_ACCESS_ADD_PLACE] = true
			acl[PLACE_ACCESS_SEE_MEMBERS] = false

			if p.GrandParentID != p.ID {
				acl[PLACE_ACCESS_REMOVE_PLACE] = true
			} else {
				acl[PLACE_ACCESS_REMOVE_PLACE] = false
			}
		} else {
			acl[PLACE_ACCESS_READ_POST] = true
			acl[PLACE_ACCESS_WRITE_POST] = true
			acl[PLACE_ACCESS_REMOVE_POST] = true
			acl[PLACE_ACCESS_CONTROL] = true
			acl[PLACE_ACCESS_REMOVE_PLACE] = true
			acl[PLACE_ACCESS_SEE_PLACE] = true
			acl[PLACE_ACCESS_ADD_MEMBERS] = true
			acl[PLACE_ACCESS_SEE_MEMBERS] = true
			acl[PLACE_ACCESS_REMOVE_MEMBERS] = true
			if p.Privacy.Locked {
				acl[PLACE_ACCESS_ADD_PLACE] = true
			} else {
				acl[PLACE_ACCESS_ADD_PLACE] = false
			}
		}
	} else if p.IsKeyholder(accountID) {
		acl[PLACE_ACCESS_READ_POST] = true
		acl[PLACE_ACCESS_SEE_MEMBERS] = true
		if p.Policy.AddMember == PLACE_POLICY_EVERYONE {
			acl[PLACE_ACCESS_ADD_MEMBERS] = true
		} else {
			acl[PLACE_ACCESS_ADD_MEMBERS] = false
		}
		if p.Privacy.Locked && p.Policy.AddPlace == PLACE_POLICY_EVERYONE {
			acl[PLACE_ACCESS_ADD_PLACE] = true
		} else {
			acl[PLACE_ACCESS_ADD_PLACE] = false
		}
		if p.Policy.AddPost == PLACE_POLICY_EVERYONE {
			acl[PLACE_ACCESS_WRITE_POST] = true
		} else {
			acl[PLACE_ACCESS_WRITE_POST] = false
		}
	} else {
		grandParent := _Manager.Place.GetByID(p.GrandParentID, nil)
		igpm := grandParent.IsMember(accountID)
		if !p.Privacy.Locked && igpm {
			acl[PLACE_ACCESS_READ_POST] = true
			acl[PLACE_ACCESS_SEE_MEMBERS] = true
		}
		switch p.Privacy.Receptive {
		case PLACE_RECEPTIVE_INTERNAL:
			if igpm {
				acl[PLACE_ACCESS_WRITE_POST] = true
			}
		case PLACE_RECEPTIVE_EXTERNAL:
			acl[PLACE_ACCESS_WRITE_POST] = true

		}
	}
	return acl

}

func (p *Place) GetAccessArray(accountID string) []string {
	access := p.GetAccess(accountID)
	array := make([]string, 0)
	for k, v := range access {
		if v {
			array = append(array, k)
		}
	}
	return array
}
