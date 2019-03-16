package nested

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

func StartupCheckups() {
	// ACCOUNTS, ACCOUNTS.POSTS, ACCOUNTS.PLACES, ACCOUNTS_RECIPIENTS, ACCOUNTS_DEVICES, ACCOUNTS_LABELS
	_ = _MongoDB.C(COLLECTION_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"email"}})
	_ = _MongoDB.C(COLLECTION_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"phone"}})
	_ = _MongoDB.C(COLLECTION_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"full_name"}})
	_ = _MongoDB.C(COLLECTION_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"username"}, Unique: true, Background: false})
	_ = _MongoDB.C(COLLECTION_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"access_places"}, Background: true})
	_ = _MongoDB.C(COLLECTION_ACCOUNTS_POSTS).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pin_time"}, Background: true})
	_ = _MongoDB.C(COLLECTION_ACCOUNTS_PLACES).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pts"}, Background: true})
	_ = _MongoDB.C(COLLECTION_ACCOUNTS_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pts"}, Background: true})
	_ = _MongoDB.C(COLLECTION_ACCOUNTS_RECIPIENTS).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pts"}, Background: true})
	_ = _MongoDB.C(COLLECTION_ACCOUNTS_RECIPIENTS).EnsureIndex(mgo.Index{Key: []string{"account_id", "recipient"}, Background: true})
	_ = _MongoDB.C(COLLECTION_ACCOUNTS_DEVICES).EnsureIndex(mgo.Index{Key: []string{"uid"}, Background: true})
	_ = _MongoDB.C(COLLECTION_ACCOUNTS_LABELS).EnsureIndex(mgo.Index{Key: []string{"labels"}, Background: true})

	// PLACES & PLACES.INVITATIONS
	_ = _MongoDB.C(COLLECTION_PLACES).EnsureIndex(mgo.Index{Key: []string{"grand_parent_id"}, Background: true})
	_ = _MongoDB.C(COLLECTION_PLACES).EnsureIndex(mgo.Index{Key: []string{"$text:name", "$text:description"}, Background: true})
	_ = _MongoDB.C(COLLECTION_PLACES_ACTIVITIES).EnsureIndex(mgo.Index{Key: []string{"place_id", "-timestamp"}, Background: true})
	_ = _MongoDB.C(COLLECTION_PLACES_ACTIVITIES).EnsureIndex(mgo.Index{Key: []string{"place_id", "action", "-timestamp"}, Background: true})

	// POSTS & POSTS.READS & POSTS.READS.COUNTERS & POSTS.COMMENTS
	_ = _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{Key: []string{"places", "-timestamp"}, Background: true})
	_ = _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{Key: []string{"places", "-last_update"}, Background: true})
	_ = _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{Key: []string{"recipients", "-timestamp"}, Background: true})
	_ = _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{Key: []string{"sender", "-timestamp"}, Background: true})
	_ = _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{
		Key: []string{"$text:body", "$text:subject"},
		Weights: map[string]int{
			"subject": 5,
			"body":    1,
		},
		Background: true,
	})
	_ = _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{Key: []string{"labels"}, Background: true})
	_ = _MongoDB.C(COLLECTION_POSTS_READS).EnsureIndex(mgo.Index{
		Key:        []string{"account_id", "place_id", "-timestamp"},
		Unique:     true,
		Background: true,
	})
	_ = _MongoDB.C(COLLECTION_POSTS_READS).EnsureIndex(mgo.Index{Key: []string{"place_id", "-timestamp"}, Background: true})
	_ = _MongoDB.C(COLLECTION_POSTS_READS).EnsureIndex(mgo.Index{Key: []string{"post_id"}, Background: true})
	_ = _MongoDB.C(COLLECTION_POSTS_READS_COUNTERS).EnsureIndex(mgo.Index{Key: []string{"account_id", "place_id"}, Background: true})
	_ = _MongoDB.C(COLLECTION_POSTS_READS_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"post_id", "account_id"}, Background: true, Unique: true})
	_ = _MongoDB.C(COLLECTION_POSTS_COMMENTS).EnsureIndex(mgo.Index{Key: []string{"post_id", "-timestamp"}, Background: true})
	_ = _MongoDB.C(COLLECTION_POSTS_COMMENTS).EnsureIndex(mgo.Index{Key: []string{"$text:text"}, Background: true})
	_ = _MongoDB.C(COLLECTION_POSTS_FILES).EnsureIndex(mgo.Index{Key: []string{"post_id", "universal_id"}, Background: true, Unique: true})

	// Tasks
	_ = _MongoDB.C(COLLECTION_TASKS).EnsureIndex(mgo.Index{Key: []string{"members"}, Background: true})
	_ = _MongoDB.C(COLLECTION_TASKS).EnsureIndex(mgo.Index{Key: []string{"due_date"}, Background: true})
	_ = _MongoDB.C(COLLECTION_TASKS).EnsureIndex(mgo.Index{Key: []string{"timestamp"}, Background: true})
	_ = _MongoDB.C(COLLECTION_TASKS).EnsureIndex(mgo.Index{
		Key: []string{"$text:title", "$text:description", "$text:todos"},
		Weights: map[string]int{
			"title":       5,
			"description": 1,
			"todos":       1,
		},
		Background: true,
	})
	_ = _MongoDB.C(COLLECTION_TASKS_FILES).EnsureIndex(mgo.Index{Key: []string{"task_id", "universal_id"}, Background: true, Unique: true})

	// Labels
	_ = _MongoDB.C(COLLECTION_LABELS).EnsureIndex(mgo.Index{Key: []string{"members"}, Background: true})
	_ = _MongoDB.C(COLLECTION_LABELS).EnsureIndex(mgo.Index{Key: []string{"title"}, Unique: true})
	_ = _MongoDB.C(COLLECTION_LABELS).EnsureIndex(mgo.Index{Key: []string{"lower_title"}, Unique: true})
	_ = _MongoDB.C(COLLECTION_LABELS_REQUESTS).EnsureIndex(mgo.Index{Key: []string{"requester_id"}, Background: true})

	// Notifications
	_ = _MongoDB.C(COLLECTION_NOTIFICATIONS).EnsureIndex(mgo.Index{Key: []string{"account_id", "type"}, Background: true})
	_ = _MongoDB.C(COLLECTION_NOTIFICATIONS).EnsureIndex(mgo.Index{Key: []string{"account_id", "post_id"}, Background: true})

	// Files
	_ = _MongoDB.C(COLLECTION_FILES).EnsureIndex(mgo.Index{Key: []string{"owners", "-upload_time"}, Background: true})
	_ = _MongoDB.C(COLLECTION_FILES).EnsureIndex(mgo.Index{Key: []string{"owners", "filename"}, Background: true})

	// Search
	_ = _MongoDB.C(COLLECTION_SEARCH_INDEX_PLACES).EnsureIndex(mgo.Index{Key: []string{"name"}, Background: true})

	// Session
	_ = _MongoDB.C(COLLECTION_SESSIONS).EnsureIndex(mgo.Index{Key: []string{"uid"}, Background: true})

	// Phones
	_ = _MongoDB.C(COLLECTION_PHONES).EnsureIndex(mgo.Index{Key: []string{"owner_id"}, Sparse: true, Background: true})

	// Tokens
	_ = _MongoDB.C(COLLECTION_TOKENS_FILES).EnsureIndex(mgo.Index{Key: []string{"universal_id"}, Background: true})
	_ = _MongoDB.C(COLLECTION_TOKENS_APPS).EnsureIndex(mgo.Index{Key: []string{"account_id", "app_id"}, Background: true})

	// Reports
	_ = _MongoDB.C(COLLECTION_REPORTS_COUNTERS).EnsureIndex(mgo.Index{Key: []string{"key", "-date"}, Background: true})

	// Hooks
	_ = _MongoDB.C(COLLECTION_HOOKS).EnsureIndex(mgo.Index{Key: []string{"anchor_id"}, Background: true})
	_ = _MongoDB.C(COLLECTION_HOOKS).EnsureIndex(mgo.Index{Key: []string{"set_by", "event_type"}, Background: true})

	if !_Manager.Account.Exists("nested") {
		md5Hash := md5.New()
		md5Hash.Write([]byte("nested1234"))
		_Manager.Account.CreateUser(
			"nested",
			hex.EncodeToString(md5Hash.Sum(nil)),
			"48222195888",
			"IR",
			"Nested",
			"Mail",
			"nested@nested.me",
			"20160922",
			"o",
		)
		_Manager.Account.SetAdmin("nested", true)
		p := PlaceCreateRequest{
			ID:            "nested",
			GrandParentID: "nested",
			AccountID:     "nested",
			Name:          "Nested",
			Description:   "I am the first member of nested.",
		}
		_Manager.Place.CreatePersonalPlace(p)

		// add the new user to his/her new personal place
		_Manager.Place.AddKeyholder("nested", "nested")
		_Manager.Place.Promote("nested", "nested")

		if err := _MongoDB.C(COLLECTION_SYSTEM_INTERNAL).Insert(
			bson.M{
				"_id": "constants",
				"$set": bson.M{
					fmt.Sprintf("strings.%s", SYSTEM_CONSTANTS_COMPANY_NAME):              DEFAULT_COMPANY_NAME,
					fmt.Sprintf("strings.%s", SYSTEM_CONSTANTS_COMPANY_DESC):              DEFAULT_COMPANY_DESC,
					fmt.Sprintf("strings.%s", SYSTEM_CONSTANTS_COMPANY_LOGO):              DEFAULT_COMPANY_LOGO,
					fmt.Sprintf("integers.%s", SYSTEM_CONSTANTS_REGISTER_MODE):            REGISTER_MODE,
					fmt.Sprintf("integers.%s", SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS):        DEFAULT_LABEL_MAX_MEMBERS,
					fmt.Sprintf("integers.%s", SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT): DEFAULT_ACCOUNT_GRAND_PLACES,
					fmt.Sprintf("integers.%s", SYSTEM_CONSTANTS_PLACE_MAX_LEVEL):          DEFAULT_PLACE_MAX_LEVEL,
					fmt.Sprintf("integers.%s", SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS):     DEFAULT_PLACE_MAX_KEYHOLDERS,
					fmt.Sprintf("integers.%s", SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN):       DEFAULT_PLACE_MAX_CHILDREN,
				},
			},
		); err != nil {
			log.Println("StartupChecks::", err.Error())
		}
	}

	// Run the appropriate migration process based on model version
	for migrate(_Manager.System.getDataModelVersion()) {
	}
}
func migrate(currentModelVersion int) bool {
	switch currentModelVersion {
	case 22:
		iter := _MongoDB.C(COLLECTION_TASKS).Find(bson.M{}).Iter()
		task := new(Task)
		for iter.Next(task) {
			memberIDs := make([]string, 0, len(task.WatcherIDs)+len(task.CandidateIDs)+len(task.EditorIDs)+2)
			memberIDs = append(task.WatcherIDs, task.CandidateIDs...)
			memberIDs = append(memberIDs, task.EditorIDs...)
			memberIDs = append(memberIDs, task.AssignorID)
			if task.AssigneeID != "" {
				memberIDs = append(memberIDs, task.AssigneeID)
			}
			_MongoDB.C(COLLECTION_TASKS).Update(
				bson.M{"_id": task.ID},
				bson.M{"$addToSet": bson.M{"members": bson.M{"$each": memberIDs}}},
			)
		}
		iter.Close()
		_MongoDB.C(COLLECTION_TASKS).DropIndex("candidates")
		_MongoDB.C(COLLECTION_TASKS).DropIndex("watchers")
		_MongoDB.C(COLLECTION_TASKS).DropIndex("assignor")
		_MongoDB.C(COLLECTION_TASKS).DropIndex("assignee")
	//case 23:
	//    iter := _MongoDB.C(COLLECTION_ACCOUNTS).Find(bson.M{}).Iter()
	//    account := new(Account)
	//    for iter.Next(account) {
	//        _ = _MongoDB.C(COLLECTION_ACCOUNTS).UpdateId(account.ID, bson.M{"$set": bson.M{"username": account.ID}})
	//    }
	//case 24:
	//    if err := _MongoDB.C(COLLECTION_SEARCH_INDEX_PLACES).DropCollection(); err != nil {
	//        _Log.Warn(err.Error())
	//    }
	//    iter := _MongoDB.C(COLLECTION_PLACES).Find(bson.M{}).Iter()
	//    place := new(Place)
	//    for iter.Next(place) {
	//        if place.Privacy.Search {
	//           if err := _MongoDB.C(COLLECTION_SEARCH_INDEX_PLACES).Insert(bson.M{"_id":place.ID, "name": place.Name, "picture": place.Picture}); err != nil {
	//               _Log.Warn(err.Error())
	//           }
	//        }
	//    }
	//    iter.Close()

	default:
		return false
	}
	currentModelVersion++
	_Manager.System.setDataModelVersion(currentModelVersion)
	return true

}
