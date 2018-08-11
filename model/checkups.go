package nested

import (
    "crypto/md5"
    "encoding/hex"
    "fmt"
    "github.com/globalsign/mgo"
    "github.com/globalsign/mgo/bson"
    "log"
)

func StartupCheckups() {
    // ACCOUNTS, ACCOUNTS.POSTS, ACCOUNTS.PLACES, ACCOUNTS_RECIPIENTS, ACCOUNTS_DEVICES, ACCOUNTS_LABELS
    _MongoDB.C(COLLECTION_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"email"}})
    _MongoDB.C(COLLECTION_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"phone"}})
    _MongoDB.C(COLLECTION_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"full_name"}})
    _MongoDB.C(COLLECTION_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"access_places"}, Background: true})
    _MongoDB.C(COLLECTION_ACCOUNTS_POSTS).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pin_time"}, Background: true})
    _MongoDB.C(COLLECTION_ACCOUNTS_PLACES).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pts"}, Background: true})
    _MongoDB.C(COLLECTION_ACCOUNTS_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pts"}, Background: true})
    _MongoDB.C(COLLECTION_ACCOUNTS_RECIPIENTS).EnsureIndex(mgo.Index{Key: []string{"account_id", "-pts"}, Background: true})
    _MongoDB.C(COLLECTION_ACCOUNTS_RECIPIENTS).EnsureIndex(mgo.Index{Key: []string{"account_id", "recipient"}, Background: true})
    _MongoDB.C(COLLECTION_ACCOUNTS_DEVICES).EnsureIndex(mgo.Index{Key: []string{"uid"}, Background: true})
    _MongoDB.C(COLLECTION_ACCOUNTS_LABELS).EnsureIndex(mgo.Index{Key: []string{"labels"}, Background: true})

    // PLACES & PLACES.INVITATIONS
    _MongoDB.C(COLLECTION_PLACES).EnsureIndex(mgo.Index{Key: []string{"grand_parent_id"}, Background: true})
    _MongoDB.C(COLLECTION_PLACES).EnsureIndex(mgo.Index{Key: []string{"$text:name", "$text:description"}, Background: true})
    _MongoDB.C(COLLECTION_PLACES_ACTIVITIES).EnsureIndex(mgo.Index{Key: []string{"place_id", "-timestamp"}, Background: true})
    _MongoDB.C(COLLECTION_PLACES_ACTIVITIES).EnsureIndex(mgo.Index{Key: []string{"place_id", "action", "-timestamp"}, Background: true})

    // POSTS & POSTS.READS & POSTS.READS.COUNTERS & POSTS.COMMENTS
    _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{Key: []string{"places", "-timestamp"}, Background: true})
    _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{Key: []string{"places", "-last_update"}, Background: true})
    _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{Key: []string{"recipients", "-timestamp"}, Background: true})
    _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{Key: []string{"sender", "-timestamp"}, Background: true})
    _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{
        Key: []string{"$text:body", "$text:subject"},
        Weights: map[string]int{
            "subject": 5,
            "body":    1,
        },
        Background: true,
    })
    _MongoDB.C(COLLECTION_POSTS).EnsureIndex(mgo.Index{Key: []string{"labels"}, Background: true})
    _MongoDB.C(COLLECTION_POSTS_READS).EnsureIndex(mgo.Index{
        Key:        []string{"account_id", "place_id", "-timestamp"},
        Unique:     true,
        Background: true,
    })
    _MongoDB.C(COLLECTION_POSTS_READS).EnsureIndex(mgo.Index{Key: []string{"place_id", "-timestamp"}, Background: true})
    _MongoDB.C(COLLECTION_POSTS_READS).EnsureIndex(mgo.Index{Key: []string{"post_id"}, Background: true})
    _MongoDB.C(COLLECTION_POSTS_READS_COUNTERS).EnsureIndex(mgo.Index{Key: []string{"account_id", "place_id"}, Background: true})
    _MongoDB.C(COLLECTION_POSTS_READS_ACCOUNTS).EnsureIndex(mgo.Index{Key: []string{"post_id", "account_id"}, Background: true, Unique: true})
    _MongoDB.C(COLLECTION_POSTS_COMMENTS).EnsureIndex(mgo.Index{Key: []string{"post_id", "-timestamp"}, Background: true})
    _MongoDB.C(COLLECTION_POSTS_COMMENTS).EnsureIndex(mgo.Index{Key: []string{"$text:text"}, Background: true})
    _MongoDB.C(COLLECTION_POSTS_FILES).EnsureIndex(mgo.Index{Key: []string{"post_id", "universal_id"}, Background: true, Unique: true})

    // Tasks
    _MongoDB.C(COLLECTION_TASKS).EnsureIndex(mgo.Index{Key: []string{"members"}, Background: true})
    _MongoDB.C(COLLECTION_TASKS).EnsureIndex(mgo.Index{Key: []string{"due_date"}, Background: true})
    _MongoDB.C(COLLECTION_TASKS).EnsureIndex(mgo.Index{
        Key: []string{"$text:title", "$text:description", "$text:todos"},
        Weights: map[string]int{
            "title":       5,
            "description": 1,
            "todos":       1,
        },
        Background: true,
    })
    _MongoDB.C(COLLECTION_TASKS_FILES).EnsureIndex(mgo.Index{Key: []string{"task_id", "universal_id"}, Background: true, Unique: true})

    // Labels
    _MongoDB.C(COLLECTION_LABELS).EnsureIndex(mgo.Index{Key: []string{"members"}, Background: true})
    _MongoDB.C(COLLECTION_LABELS).EnsureIndex(mgo.Index{Key: []string{"title"}, Unique: true})
    _MongoDB.C(COLLECTION_LABELS).EnsureIndex(mgo.Index{Key: []string{"lower_title"}, Unique: true})
    _MongoDB.C(COLLECTION_LABELS_REQUESTS).EnsureIndex(mgo.Index{Key: []string{"requester_id"}, Background: true})

    // Notifications
    _MongoDB.C(COLLECTION_NOTIFICATIONS).EnsureIndex(mgo.Index{Key: []string{"account_id", "type"}, Background: true})
    _MongoDB.C(COLLECTION_NOTIFICATIONS).EnsureIndex(mgo.Index{Key: []string{"account_id", "post_id"}, Background: true})

    // Files
    _MongoDB.C(COLLECTION_FILES).EnsureIndex(mgo.Index{Key: []string{"owners", "-upload_time"}, Background: true})
    _MongoDB.C(COLLECTION_FILES).EnsureIndex(mgo.Index{Key: []string{"owners", "filename"}, Background: true})

    // Search
    _MongoDB.C(COLLECTION_SEARCH_INDEX_PLACES).EnsureIndex(mgo.Index{Key: []string{"name"}, Background: true})

    // Session
    _MongoDB.C(COLLECTION_SESSIONS).EnsureIndex(mgo.Index{Key: []string{"uid"}, Background: true})

    // Phones
    _MongoDB.C(COLLECTION_PHONES).EnsureIndex(mgo.Index{Key: []string{"owner_id"}, Sparse: true, Background: true})

    // Tokens
    _MongoDB.C(COLLECTION_TOKENS_FILES).EnsureIndex(mgo.Index{Key: []string{"universal_id"}, Background: true})
    _MongoDB.C(COLLECTION_TOKENS_APPS).EnsureIndex(mgo.Index{Key: []string{"account_id", "app_id"}, Background: true})

    // Reports
    _MongoDB.C(COLLECTION_REPORTS_COUNTERS).EnsureIndex(mgo.Index{Key: []string{"key", "-date"}, Background: true})

    // Hooks
    _MongoDB.C(COLLECTION_HOOKS).EnsureIndex(mgo.Index{Key: []string{"anchor_id"}, Background: true})
    _MongoDB.C(COLLECTION_HOOKS).EnsureIndex(mgo.Index{Key: []string{"set_by", "event_type"}, Background: true})

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

    case 15:
        tasks := make([]Task, 0)
        _MongoDB.C(COLLECTION_TASKS).Find(
            bson.M{
                "due_date": bson.M{"$gt": Timestamp()},
                "status": bson.M{"$nin": []TaskStatus{
                    TASK_STATUS_COMPLETED, TASK_STATUS_HOLD, TASK_STATUS_OVERDUE, TASK_STATUS_FAILED,
                }},
            },
        ).All(&tasks)
        for _, task := range tasks {
            _Manager.TimeBucket.AddOverdueTask(task.DueDate, task.ID)
        }
    case 16:
        _MongoDB.C("hooks.places").DropCollection()
        _MongoDB.C("hooks.accounts").DropCollection()
    case 17:
        _MongoDB.C(COLLECTION_TOKENS_APPS).DropIndex("account_id")
    case 18:
        _MongoDB.C(COLLECTION_PLACES_ACTIVITIES).RemoveAll(
            bson.M{
                "action": bson.M{"$nin": []int{
                    PLACE_ACTIVITY_ACTION_MEMBER_REMOVE,
                    PLACE_ACTIVITY_ACTION_MEMBER_JOIN,
                    PLACE_ACTIVITY_ACTION_PLACE_ADD,
                    PLACE_ACTIVITY_ACTION_POST_ADD,
                    PLACE_ACTIVITY_ACTION_POST_REMOVE,
                    PLACE_ACTIVITY_ACTION_POST_MOVE_FROM,
                    PLACE_ACTIVITY_ACTION_POST_MOVE_TO,
                }},
            },
        )
    case 19:
        _MongoDB.C("places.invitations").DropCollection()
        _MongoDB.C("hooks.accounts").DropCollection()
        _MongoDB.C("hooks.places").DropCollection()
    case 20:
    case 21:
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

    default:
        return false
    }
    currentModelVersion++
    _Manager.System.setDataModelVersion(currentModelVersion)
    return true

}
