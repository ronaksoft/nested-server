package nested

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"go.uber.org/zap"
)

func StartupCheckups() {
	// ACCOUNTS, ACCOUNTS.POSTS, ACCOUNTS.PLACES, ACCOUNTS_RECIPIENTS, ACCOUNTS_DEVICES, ACCOUNTS_LABELS
	_ = _MongoDB.C(global.CollectionAccounts).EnsureIndex(mgo.Index{Key: []string{"email"}})
	_ = _MongoDB.C(global.CollectionAccounts).EnsureIndex(mgo.Index{Key: []string{"phone"}})
	_ = _MongoDB.C(global.CollectionAccounts).EnsureIndex(mgo.Index{Key: []string{"full_name"}})
	_ = _MongoDB.C(global.CollectionAccounts).EnsureIndex(mgo.Index{Key: []string{"username"}, Unique: true, Background: false})
	_ = _MongoDB.C(global.CollectionAccounts).EnsureIndex(mgo.Index{Key: []string{"access_places"}, Background: true})
	_ = _MongoDB.C(global.CollectionAccountsPosts).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pin_time"}, Background: true})
	_ = _MongoDB.C(global.CollectionAccountsPlaces).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pts"}, Background: true})
	_ = _MongoDB.C(global.CollectionAccountsAccounts).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pts"}, Background: true})
	_ = _MongoDB.C(global.CollectionAccountsRecipients).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pts"}, Background: true})
	_ = _MongoDB.C(global.CollectionAccountsRecipients).EnsureIndex(mgo.Index{Key: []string{"account_id", "recipient"}, Background: true})
	_ = _MongoDB.C(global.CollectionAccountsDevices).EnsureIndex(mgo.Index{Key: []string{"uid"}, Background: true})
	_ = _MongoDB.C(global.CollectionAccountsLabels).EnsureIndex(mgo.Index{Key: []string{"labels"}, Background: true})

	// PLACES & PLACES.INVITATIONS
	_ = _MongoDB.C(global.CollectionPlaces).EnsureIndex(mgo.Index{Key: []string{"grand_parent_id"}, Background: true})
	_ = _MongoDB.C(global.CollectionPlaces).EnsureIndex(mgo.Index{Key: []string{"$text:name", "$text:description"}, Background: true})
	_ = _MongoDB.C(global.CollectionPlacesActivities).EnsureIndex(mgo.Index{Key: []string{"place_id", "-timestamp"}, Background: true})
	_ = _MongoDB.C(global.CollectionPlacesActivities).EnsureIndex(mgo.Index{Key: []string{"place_id", "action", "-timestamp"}, Background: true})

	// POSTS & POSTS.READS & POSTS.READS.COUNTERS & POSTS.COMMENTS
	_ = _MongoDB.C(global.CollectionPosts).EnsureIndex(mgo.Index{Key: []string{"places", "-timestamp"}, Background: true})
	_ = _MongoDB.C(global.CollectionPosts).EnsureIndex(mgo.Index{Key: []string{"places", "-last_update"}, Background: true})
	_ = _MongoDB.C(global.CollectionPosts).EnsureIndex(mgo.Index{Key: []string{"recipients", "-timestamp"}, Background: true})
	_ = _MongoDB.C(global.CollectionPosts).EnsureIndex(mgo.Index{Key: []string{"sender", "-timestamp"}, Background: true})

	_ = _MongoDB.C(global.CollectionPosts).EnsureIndex(mgo.Index{
		Key: []string{"$text:content", "$text:subject"},
		Weights: map[string]int{
			"subject": 5,
			"content": 1,
		},
		Background: true,
	})
	_ = _MongoDB.C(global.CollectionPosts).EnsureIndex(mgo.Index{Key: []string{"labels"}, Background: true})
	_ = _MongoDB.C(global.CollectionPostsReads).EnsureIndex(mgo.Index{
		Key:        []string{"account_id", "place_id", "-timestamp"},
		Unique:     true,
		Background: true,
	})
	_ = _MongoDB.C(global.CollectionPostsReads).EnsureIndex(mgo.Index{Key: []string{"place_id", "-timestamp"}, Background: true})
	_ = _MongoDB.C(global.CollectionPostsReads).EnsureIndex(mgo.Index{Key: []string{"post_id"}, Background: true})
	_ = _MongoDB.C(global.CollectionPostsReadsCounters).EnsureIndex(mgo.Index{Key: []string{"account_id", "place_id"}, Background: true})
	_ = _MongoDB.C(global.CollectionPostsReadsAccounts).EnsureIndex(mgo.Index{Key: []string{"post_id", "account_id"}, Background: true, Unique: true})
	_ = _MongoDB.C(global.CollectionPostsComments).EnsureIndex(mgo.Index{Key: []string{"post_id", "-timestamp"}, Background: true})
	_ = _MongoDB.C(global.CollectionPostsComments).EnsureIndex(mgo.Index{Key: []string{"$text:text"}, Background: true})
	_ = _MongoDB.C(global.CollectionPostsFiles).EnsureIndex(mgo.Index{Key: []string{"post_id", "universal_id"}, Background: true, Unique: true})

	// Tasks
	_ = _MongoDB.C(global.CollectionTasks).EnsureIndex(mgo.Index{Key: []string{"members"}, Background: true})
	_ = _MongoDB.C(global.CollectionTasks).EnsureIndex(mgo.Index{Key: []string{"due_date"}, Background: true})
	_ = _MongoDB.C(global.CollectionTasks).EnsureIndex(mgo.Index{Key: []string{"timestamp"}, Background: true})
	_ = _MongoDB.C(global.CollectionTasks).EnsureIndex(mgo.Index{
		Key: []string{"$text:title", "$text:description", "$text:todos"},
		Weights: map[string]int{
			"title":       5,
			"description": 1,
			"todos":       1,
		},
		Background: true,
	})
	_ = _MongoDB.C(global.CollectionTasksFiles).EnsureIndex(mgo.Index{Key: []string{"task_id", "universal_id"}, Background: true, Unique: true})

	// Labels
	_ = _MongoDB.C(global.CollectionLabels).EnsureIndex(mgo.Index{Key: []string{"members"}, Background: true})
	_ = _MongoDB.C(global.CollectionLabels).EnsureIndex(mgo.Index{Key: []string{"title"}, Unique: true})
	_ = _MongoDB.C(global.CollectionLabels).EnsureIndex(mgo.Index{Key: []string{"lower_title"}, Unique: true})
	_ = _MongoDB.C(global.CollectionLabelsRequests).EnsureIndex(mgo.Index{Key: []string{"requester_id"}, Background: true})

	// Notifications
	_ = _MongoDB.C(global.CollectionNotifications).EnsureIndex(mgo.Index{Key: []string{"account_id", "type"}, Background: true})
	_ = _MongoDB.C(global.CollectionNotifications).EnsureIndex(mgo.Index{Key: []string{"account_id", "post_id"}, Background: true})

	// Files
	_ = _MongoDB.C(global.CollectionFiles).EnsureIndex(mgo.Index{Key: []string{"owners", "-upload_time"}, Background: true})
	_ = _MongoDB.C(global.CollectionFiles).EnsureIndex(mgo.Index{Key: []string{"owners", "filename"}, Background: true})

	// Search
	_ = _MongoDB.C(global.CollectionSearchIndexPlaces).EnsureIndex(mgo.Index{Key: []string{"name"}, Background: true})

	// Session
	_ = _MongoDB.C(global.CollectionSessions).EnsureIndex(mgo.Index{Key: []string{"uid"}, Background: true})

	// Phones
	_ = _MongoDB.C(global.CollectionPhones).EnsureIndex(mgo.Index{Key: []string{"owner_id"}, Sparse: true, Background: true})

	// Tokens
	_ = _MongoDB.C(global.CollectionTokensFiles).EnsureIndex(mgo.Index{Key: []string{"universal_id"}, Background: true})
	_ = _MongoDB.C(global.CollectionTokensApps).EnsureIndex(mgo.Index{Key: []string{"account_id", "app_id"}, Background: true})

	// Reports
	_ = _MongoDB.C(global.CollectionReportsCounters).EnsureIndex(mgo.Index{Key: []string{"key", "-date"}, Background: true})

	// Hooks
	_ = _MongoDB.C(global.CollectionHooks).EnsureIndex(mgo.Index{Key: []string{"anchor_id"}, Background: true})
	_ = _MongoDB.C(global.CollectionHooks).EnsureIndex(mgo.Index{Key: []string{"set_by", "event_type"}, Background: true})

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

		if err := _MongoDB.C(global.CollectionSystemInternal).Insert(
			bson.M{
				"_id": "constants",
				fmt.Sprintf("strings.%s", global.SystemConstantsCompanyName):             global.DefaultCompanyName,
				fmt.Sprintf("strings.%s", global.SystemConstantsCompanyDesc):             global.DefaultCompanyDesc,
				fmt.Sprintf("strings.%s", global.SystemConstantsCompanyLogo):             global.DefaultCompanyLogo,
				fmt.Sprintf("integers.%s", global.SystemConstantsRegisterMode):           global.RegisterMode,
				fmt.Sprintf("integers.%s", global.SystemConstantsLabelMaxMembers):        global.DefaultLabelMaxMembers,
				fmt.Sprintf("integers.%s", global.SystemConstantsAccountGrandPlaceLimit): global.DefaultAccountGrandPlaces,
				fmt.Sprintf("integers.%s", global.SystemConstantsPlaceMaxLevel):          global.DefaultPlaceMaxLevel,
				fmt.Sprintf("integers.%s", global.SystemConstantsPlaceMaxKeyHolders):     global.DefaultPlaceMaxKeyHolders,
				fmt.Sprintf("integers.%s", global.SystemConstantsPlaceMaxChildren):       global.DefaultPlaceMaxChildren,
			},
		); err != nil {
			log.Warn("StartupChecks::", zap.Error(err))
		}
	}

	// Run the appropriate migration process based on model version
	for migrate(_Manager.System.getDataModelVersion()) {
	}
}
func migrate(currentModelVersion int) bool {
	switch currentModelVersion {
	case 22:
		iter := _MongoDB.C(global.CollectionTasks).Find(bson.M{}).Iter()
		task := new(Task)
		for iter.Next(task) {
			memberIDs := make([]string, 0, len(task.WatcherIDs)+len(task.CandidateIDs)+len(task.EditorIDs)+2)
			memberIDs = append(task.WatcherIDs, task.CandidateIDs...)
			memberIDs = append(memberIDs, task.EditorIDs...)
			memberIDs = append(memberIDs, task.AssignorID)
			if task.AssigneeID != "" {
				memberIDs = append(memberIDs, task.AssigneeID)
			}
			_MongoDB.C(global.CollectionTasks).Update(
				bson.M{"_id": task.ID},
				bson.M{"$addToSet": bson.M{"members": bson.M{"$each": memberIDs}}},
			)
		}
		iter.Close()
		_MongoDB.C(global.CollectionTasks).DropIndex("candidates")
		_MongoDB.C(global.CollectionTasks).DropIndex("watchers")
		_MongoDB.C(global.CollectionTasks).DropIndex("assignor")
		_MongoDB.C(global.CollectionTasks).DropIndex("assignee")
	default:
		return false
	}
	currentModelVersion++
	_Manager.System.setDataModelVersion(currentModelVersion)
	return true

}
