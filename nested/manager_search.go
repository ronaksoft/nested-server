package nested

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"github.com/globalsign/mgo/bson"
	"go.uber.org/zap"
	"strings"
)

const (
	PlaceSearchFilterGrandPlace      string = "grand_places"
	PlaceSearchFilterLockedPlaces    string = "locked_places"
	PlaceSearchFilterUnlockedPlaces  string = "unlocked_places"
	PlaceSearchFilterPersonal        string = "personal_places"
	PlaceSearchFilterShared          string = "shared_places"
	PlaceSearchFilterAll             string = "all"
	AccountSearchFilterUsersEnabled  string = "users_enabled"
	AccountSearchFilterUsersDisabled string = "users_disabled"
	AccountSearchFilterUsers         string = "users"
	AccountSearchFilterDevices       string = "devices"
	AccountSearchFilterAll           string = "all"
)

type SearchManager struct{}

func newSearchManager() *SearchManager {
	return new(SearchManager)
}

// 	Places searches through PLACE collection and apply grand_parent_id, keyword and filter on its query
// 	filter :	PlaceSearchFilterGrandPlace
// 				PlaceSearchFilterLockedPlaces
// 				PlaceSearchFilterUnlockedPlaces
// 				PlaceSearchFilterPersonal
// 				PlaceSearchFilterAll
func (sm *SearchManager) Places(keyword, filter, sort, grandParentID string, pg Pagination) []Place {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	places := make([]Place, 0, pg.GetLimit())
	q := bson.M{
		"$or": []bson.M{
			{"_id": bson.M{"$regex": fmt.Sprintf("^%s", keyword), "$options": "i"}},
			{"name": bson.M{"$regex": fmt.Sprintf("^%s", keyword), "$options": "i"}},
		},
	}
	if grandParentID != "" {
		q["grand_parent_id"] = grandParentID
	}
	switch filter {
	case PlaceSearchFilterGrandPlace:
		q["level"] = 0
		q["type"] = PlaceTypeShared
	case PlaceSearchFilterLockedPlaces:
		q["level"] = bson.M{"$ne": 0}
		q["privacy.locked"] = true
	case PlaceSearchFilterUnlockedPlaces:
		q["level"] = bson.M{"$ne": 0}
		q["privacy.locked"] = false
	case PlaceSearchFilterPersonal:
		q["type"] = PlaceTypePersonal
	case PlaceSearchFilterShared:
		q["type"] = PlaceTypeShared
	case PlaceSearchFilterAll:
	default:

	}

	Q := db.C(global.CollectionPlaces).Find(q)
	if len(sort) != 0 {
		Q = Q.Sort(sort)
	}
	Q = Q.Skip(pg.GetSkip()).Limit(pg.GetLimit())
	// Log Explain Query

	if err := Q.All(&places); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return places
}

// 	PlacesForCompose return an array of Place objects filtered by keyword
// 	It searches through two rounds:
// 		1. ACCOUNTS.PLACES collection and sorted by the connection strength
// 		2. SEARCH.PLACES collection which contains all the places which are searchable
func (sm *SearchManager) PlacesForCompose(keyword, accountID string, pg Pagination) []Place {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	limit := pg.GetLimit()
	foundPlaces := make([]Place, 0, limit)
	q := []bson.M{
		{"$match": bson.M{"account_id": accountID}},
		{"$lookup": bson.M{
			"from":         global.CollectionPlaces,
			"localField":   "place_id",
			"foreignField": "_id",
			"as":           "place",
		}},
		{"$match": bson.M{
			"place": bson.M{
				"$elemMatch": bson.M{
					"$or": []bson.M{
						{"name": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
						{"_id": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
					},
				},
			},
		}},
		{"$limit": limit},
	}
	Q := db.C(global.CollectionAccountsPlaces).Pipe(q)
	iter := Q.Iter()
	defer iter.Close()
	fetchedDoc := struct {
		AccountID string  `bson:"account_id"`
		Places    []Place `bson:"place"`
	}{}
	for iter.Next(&fetchedDoc) {
		foundPlaces = append(foundPlaces, fetchedDoc.Places[0])
	}
	limit = limit - len(foundPlaces)
	if limit > 0 {
		iter = db.C(global.CollectionSearchIndexPlaces).Find(bson.M{
			"$or": []bson.M{
				{"_id": bson.M{"$regex": fmt.Sprintf("^%s", keyword), "$options": "i"}},
				{"name": bson.M{"$regex": fmt.Sprintf("^%s", keyword), "$options": "i"}},
			},
		}).Limit(limit).Iter()
	}
	place := Place{}
	for iter.Next(&place) {
		foundPlaces = append(foundPlaces, place)
	}
	return foundPlaces
}

// 	RecipientsForCompose returns an array of Recipients filtered by keyword
func (sm *SearchManager) RecipientsForCompose(keyword, accountID string, pg Pagination) []string {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	limit := pg.GetLimit()
	recipients := make([]string, 0, limit)
	m := tools.M{}
	iter := db.C(global.CollectionAccountsRecipients).Find(bson.M{
		"account_id": accountID,
		"recipient":  bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"},
	}).Sort("-pts").Limit(limit).Iter()
	defer iter.Close()

	for iter.Next(m) {
		recipients = append(recipients, m["recipient"].(string))
	}
	return recipients
}

// 	PlacesForSearch returns an array of Place objects filtered by keyword
// 	It searches through all the places that accountID is member of
func (sm *SearchManager) PlacesForSearch(keyword, accountID string, pg Pagination) []Place {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	limit := pg.GetLimit()
	foundPlaces := make([]Place, 0, limit)
	q := []bson.M{
		{"$match": bson.M{"account_id": accountID}},
		{"$lookup": bson.M{
			"from":         global.CollectionPlaces,
			"localField":   "place_id",
			"foreignField": "_id",
			"as":           "place",
		}},
		{"$match": bson.M{
			"place": bson.M{
				"$elemMatch": bson.M{
					"$or": []bson.M{
						{"name": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
						{"_id": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
					},
				},
			},
		}},
		{"$limit": limit},
	}
	Q := db.C(global.CollectionAccountsPlaces).Pipe(q)

	iter := Q.Iter()
	defer iter.Close()
	fetchedDoc := struct {
		AccountID string  `bson:"account_id"`
		Places    []Place `bson:"place"`
	}{}
	for iter.Next(&fetchedDoc) {
		foundPlaces = append(foundPlaces, fetchedDoc.Places[0])
	}
	return foundPlaces
}

// 	Accounts searches through  ACCOUNTS collection and apply keyword, filter on it query
// 	filter:			AccountSearchFilterUsersEnabled
// 					AccountSearchFilterUsersDisabled
// 					AccountSearchFilterUsers
// 					AccountSearchFilterDevices
// 					AccountSearchFilterAll
func (sm *SearchManager) Accounts(keyword, filter, sort string, pg Pagination) []Account {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	accounts := make([]Account, 0, pg.GetLimit())
	q := bson.M{
		"$or": []bson.M{
			{"_id": bson.M{"$regex": fmt.Sprintf("^%s", keyword), "$options": "i"}},
			{"full_name": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
		},
	}
	switch filter {
	case AccountSearchFilterUsersEnabled:
		q["acc_type"] = ACCOUNT_TYPE_USER
		q["disabled"] = false
	case AccountSearchFilterUsersDisabled:
		q["acc_type"] = ACCOUNT_TYPE_USER
		q["disabled"] = true
	case AccountSearchFilterUsers:
		q["acc_type"] = ACCOUNT_TYPE_USER
	case AccountSearchFilterDevices:
		q["acc_type"] = ACCOUNT_TYPE_DEVICE
	case AccountSearchFilterAll:
	default:

	}
	Q := db.C(global.CollectionAccounts).Find(q)
	if len(sort) != 0 {
		Q = Q.Sort(sort)
	}

	if err := Q.Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&accounts); err != nil {
		log.Warn("Got error", zap.Error(err))
	}

	return accounts
}

// 	AccountsForAddToPlace search through the members of grand place of placeID and filter by keyword
func (sm *SearchManager) AccountsForAddToPlace(accountID, placeID string, keywords []string, pg Pagination) []Account {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	grandPlaceID := placeID
	if strings.Index(placeID, ".") != -1 {
		grandPlaceID = strings.Split(placeID, ".")[0]
	}
	accounts := make([]Account, 0, pg.GetLimit())
	q := bson.M{
		"$and": []bson.M{
			{"acc_type": ACCOUNT_TYPE_USER},
			{"disabled": false},
			{"access_places": grandPlaceID},
			{"access_places": bson.M{"$ne": placeID}},
			{"_id": bson.M{"$ne": accountID}},
		},
	}

	conds := make([]bson.M, 0, len(keywords)*3)
	for _, k := range keywords {
		conds = append(conds, bson.M{"_id": bson.M{"$regex": fmt.Sprintf("^%s", k), "$options": "i"}})
		conds = append(conds, bson.M{"full_name": bson.M{"$regex": fmt.Sprintf("^%s", k), "$options": "i"}})
	}
	q["$or"] = conds
	db.C(global.CollectionAccounts).Find(q).Select(
		bson.M{"fname": 1, "lname": 1, "picture": 1},
	).Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&accounts)
	return accounts
}

// 	AccountsForAddToGrandPlace search through all the members of nested who are searchable and they are not already member of
// 	the placeID
func (sm *SearchManager) AccountsForAddToGrandPlace(inviterID, placeID string, keyword string, pg Pagination) []Account {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	limit := pg.GetLimit()
	foundAccounts := make([]Account, 0, limit)
	q := []bson.M{
		{"$match": bson.M{"account_id": inviterID}},
		{"$lookup": bson.M{
			"from":         global.CollectionAccounts,
			"localField":   "other_account_id",
			"foreignField": "_id",
			"as":           "account",
		}},
		{"$match": bson.M{
			"account": bson.M{
				"$elemMatch": bson.M{
					"$or": []bson.M{
						{"fname": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
						{"lname": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
						{"_id": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
					},
					"access_places": bson.M{"$ne": placeID},
					"disabled":      false,
				},
			},
		}},
		{"$limit": limit},
	}
	Q := db.C(global.CollectionAccountsAccounts).Pipe(q)
	iter := Q.Iter()
	defer iter.Close()
	fetchedDoc := struct {
		AccountID string    `bson:"account_id"`
		Accounts  []Account `bson:"account"`
	}{}
	for iter.Next(&fetchedDoc) {
		foundAccounts = append(foundAccounts, fetchedDoc.Accounts[0])
	}
	limit = limit - len(foundAccounts)

	// if limit > 0 {
	//     iter = _MongoDB.C(global.CollectionSearchIndexPlaces).Find(bson.M{
	//         "$and": []bson.M{
	//             {"acc_type": ACCOUNT_TYPE_USER},
	//             {"disabled": false},
	//             {"privacy.searchable": true},
	//             {"access_places": bson.M{"$ne": placeID}},
	//             {"_id": bson.M{"$ne": inviterID}},
	//         },
	//         "$or": []bson.M{
	//             {"_id": bson.M{"$regex": fmt.Sprintf("^%s", keyword), "$options": "i"}},
	//             {"full_name": bson.M{"$regex": fmt.Sprintf("^%s", keyword), "$options": "i"}},
	//         },
	//     }).Limit(limit).Iter()
	// }

	// account := Account{}
	// for iter.Next(&account) {
	//     foundAccounts = append(foundAccounts, account)
	// }
	return foundAccounts
}

// 	AccountsForSearch search through all the members of placeIDs and all the users who are searchable.
func (sm *SearchManager) AccountsForSearch(accountID string, keyword string, pg Pagination) []Account {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	limit := pg.GetLimit()
	foundAccounts := make([]Account, 0, pg.GetLimit())
	q := []bson.M{
		{"$match": bson.M{"account_id": accountID}},
		{"$lookup": bson.M{
			"from":         global.CollectionAccounts,
			"localField":   "other_account_id",
			"foreignField": "_id",
			"as":           "account",
		}},
		{"$match": bson.M{
			"account": bson.M{
				"$elemMatch": bson.M{
					"$or": []bson.M{
						{"full_name": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
						{"_id": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
					},
					"disabled": false,
				},
			},
		}},
		{"$limit": limit},
	}
	Q := db.C(global.CollectionAccountsAccounts).Pipe(q)
	iter1 := Q.Iter()
	fetchedDoc := struct {
		AccountID string    `bson:"account_id"`
		Accounts  []Account `bson:"account"`
	}{}
	for iter1.Next(&fetchedDoc) {
		foundAccounts = append(foundAccounts, fetchedDoc.Accounts[0])
	}
	iter1.Close()

	account := new(Account)
	if err := db.C(global.CollectionAccounts).Find(bson.M{"_id": keyword}).One(account); err == nil {
		foundAccounts = append(foundAccounts, *account)
	}
	return foundAccounts
}

// 	AccountsForTaskMention searches through members of placeIDs and filter by keyword
func (sm *SearchManager) AccountsForTaskMention(task *Task, keyword string) []Account {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	accounts := make([]Account, 0, global.DefaultMaxResultLimit)
	accountsDomain := append(task.WatcherIDs, task.AssignorID, task.AssigneeID)
	accountsDomain = append(accountsDomain, task.CandidateIDs...)
	accountsDomain = append(accountsDomain, task.EditorIDs...)
	q := bson.M{
		"_id": bson.M{"$in": accountsDomain},
		"$or": []bson.M{
			{"full_name": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
			{"_id": bson.M{
				"$regex": fmt.Sprintf("%s", keyword), "$options": "i",
			}},
		},
	}

	db.C(global.CollectionAccounts).Find(q).All(&accounts)
	return accounts
}

// 	AccountsForPostMention searches through members of placeIDs and filter by keyword
func (sm *SearchManager) AccountsForPostMention(placeIDs, keywords []string, pg Pagination) []Account {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	accounts := make([]Account, 0, global.DefaultMaxResultLimit)
	targetPlacesMap := make(map[string]bool, len(placeIDs))
	for _, placeID := range placeIDs {
		place := _Manager.Place.GetByID(placeID, nil)
		if place == nil {
			continue
		}
		// Add the place to search list
		targetPlacesMap[placeID] = true

		// If place is not locked then add the grand-place to list
		if !place.Privacy.Locked {
			targetPlacesMap[place.GrandParentID] = true
		}
	}
	targetPlaceIDs := make([]string, 0, len(targetPlacesMap))
	for placeID := range targetPlacesMap {
		targetPlaceIDs = append(targetPlaceIDs, placeID)
	}
	q := bson.M{
		"$and": []bson.M{
			{"acc_type": ACCOUNT_TYPE_USER},
			{"disabled": false},
			{"access_places": bson.M{"$in": targetPlaceIDs}},
		},
	}
	conds := make([]bson.M, 0, len(keywords)*3)
	for _, k := range keywords {
		conds = append(conds, bson.M{"_id": bson.M{"$regex": fmt.Sprintf("^%s", k), "$options": "i"}})
		conds = append(conds, bson.M{"full_name": bson.M{"$regex": fmt.Sprintf("%s", k), "$options": "i"}})
	}
	q["$or"] = conds
	db.C(global.CollectionAccounts).Find(q).Select(
		bson.M{"fname": 1, "lname": 1, "picture": 1},
	).Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&accounts)
	return accounts
}

// Apps searches through registered apps
func (sm *SearchManager) Apps(keyword string, pg Pagination) []App {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	apps := make([]App, 0, pg.GetLimit())
	if err := db.C(global.CollectionApps).Find(
		bson.M{"app_name": bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}},
	).Limit(pg.GetLimit()).All(&apps); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return apps
}

// 	Labels returns an array of all the labels ids filtered by keyword and filter
func (sm *SearchManager) Labels(accountID, keyword, filter string, pg Pagination) []Label {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	labels := make([]Label, 0, pg.GetLimit())
	q := bson.M{}
	if len(keyword) > 0 {
		q["title"] = bson.M{"$regex": fmt.Sprintf("%s", keyword), "$options": "i"}
	}
	switch filter {
	case LabelFilterMyLabels:
		q["$or"] = []bson.M{
			{"$and": []bson.M{{"members": accountID}, {"public": false}}},
			{"public": true},
		}
	case LabelFilterMyPrivates:
		q["members"] = accountID
		q["public"] = false
	case LabelFilterPrivates:
		q["public"] = false
	case LabelFilterPublic:
		q["public"] = true
	case LabelFilterAll:
		fallthrough
	default:
	}
	if err := db.C(global.CollectionLabels).Find(q).Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&labels); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return labels
}

// 	Posts searches through posts in "placeIDs" and filter them by keywords
func (sm *SearchManager) Posts(keyword, accountID string, placeIDs, senderIDs, labelIDs []string, hasAttachments bool, pg Pagination) []Post {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	sortItem := PostSortTimestamp
	sortDir := fmt.Sprintf("-%s", sortItem)
	q := bson.M{}
	switch len(labelIDs) {
	case 0:
	case 1:
		q["labels"] = labelIDs[0]
	default:
		v := make([]bson.M, 0)
		for _, labelID := range labelIDs {
			v = append(v, bson.M{"labels": labelID})
		}
		q["$and"] = v
	}
	if len(placeIDs) > 0 {
		q["places"] = bson.M{"$in": placeIDs}
	}
	if len(senderIDs) > 0 {
		q["sender"] = bson.M{"$in": senderIDs}
	}
	if len(keyword) > 0 {
		q["$text"] = bson.M{
			"$search":             keyword,
			"$caseSensitive":      false,
			"$diacriticSensitive": false,
		}
	}

	q, sortDir = pg.FillQuery(q, sortItem, sortDir)

	if hasAttachments {
		q["counters.attaches"] = bson.M{"$gt": 0}
	}

	post := new(Post)
	posts := make([]Post, 0, pg.GetLimit())
	iter := db.C(global.CollectionPosts).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit()).Iter()
	defer iter.Close()
	for iter.Next(post) && len(posts) < cap(posts) {
		if post.HasAccess(accountID) {
			posts = append(posts, *post)
		}
	}
	return posts
}

// 	PostsConversations returns posts between two accounts: accountID1 and accountID2
func (sm *SearchManager) PostsConversations(peerID1, peerID2, keywords string, pg Pagination) []Post {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	sortItem := PostSortTimestamp
	sortDir := fmt.Sprintf("-%s", sortItem)
	q := bson.M{
		"$or": []bson.M{
			{"$and": []bson.M{
				{"sender": peerID1},
				{"$or": []bson.M{
					{"places": bson.M{"$regex": fmt.Sprintf("^%s\\b", peerID2), "$options": "i"}},
					{"recipients": peerID2},
				}},
			}},
			{"$and": []bson.M{
				{"sender": peerID2},
				{"$or": []bson.M{
					{"places": bson.M{"$regex": fmt.Sprintf("^%s\\b", peerID1), "$options": "i"}},
					{"recipients": peerID1},
				}},
			}},
		},
	}

	if len(keywords) > 0 {
		q["$text"] = bson.M{
			"$search":             keywords,
			"$caseSensitive":      false,
			"$diacriticSensitive": true,
		}
	}
	q, sortDir = pg.FillQuery(q, sortItem, sortDir)

	posts := make([]Post, 0, pg.GetLimit())
	if err := db.C(global.CollectionPosts).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Limit(pg.GetLimit()).All(&posts); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	return posts
}

// 	Tasks searches through tasks
func (sm *SearchManager) Tasks(keyword, accountID string, assignorIDs, assigneeIDs, labelIDs []string, hasAttachments bool, pg Pagination) []Task {
	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	sortItem := PostSortTimestamp
	sortDir := fmt.Sprintf("-%s", sortItem)
	q := bson.M{
		"_removed": false,
	}
	switch len(labelIDs) {
	case 0:
	case 1:
		q["labels"] = labelIDs[0]
	default:
		v := make([]bson.M, 0, len(labelIDs))
		for _, labelID := range labelIDs {
			v = append(v, bson.M{"labels": labelID})
		}
		q["$and"] = v
	}
	if len(assigneeIDs) > 0 {
		q["assignee"] = bson.M{"$in": assigneeIDs}
	}
	if len(assignorIDs) > 0 {
		q["assignor"] = bson.M{"$in": assignorIDs}
	}
	if len(keyword) > 0 {
		q["$text"] = bson.M{
			"$search":             keyword,
			"$caseSensitive":      false,
			"$diacriticSensitive": false,
		}
	}
	if pg.After > 0 && pg.Before > 0 {
		switch x := q["$and"].(type) {
		case []bson.M:
			q["$and"] = append(x, bson.M{"$gt": pg.After}, bson.M{"$lt": pg.Before})
		default:
			q["$and"] = []bson.M{
				{"$gt": pg.After}, {"$lt": pg.Before},
			}
		}
	} else if pg.After > 0 {
		sortDir = sortItem
		q[sortItem] = bson.M{"$gt": pg.After}
	} else if pg.Before > 0 {
		q[sortItem] = bson.M{"$lt": pg.Before}
	}

	if hasAttachments {
		q["counters.attachments"] = bson.M{"$gt": 0}
	}

	task := new(Task)
	tasks := make([]Task, 0, pg.GetLimit())
	iter := db.C(global.CollectionTasks).Find(q).Sort(sortDir).Skip(pg.GetSkip()).Iter()
	defer iter.Close()
	for iter.Next(task) && len(tasks) < cap(tasks) {
		if task.HasAccess(accountID, TaskAccessRead) {
			tasks = append(tasks, *task)
		}
	}
	return tasks
}

// 	AddPlaceToSearchIndex adds placeID and placeName into the search index, then all the users can find the place
// 	info in the search result
func (sm *SearchManager) AddPlaceToSearchIndex(placeID, placeName string, p Picture) {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if _, err := db.C(global.CollectionSearchIndexPlaces).UpsertId(placeID,
		bson.M{"$set": bson.M{"name": placeName, "picture": p}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
}

// RemovePlaceFromSearchIndex removes placeID and placeName from the search index collection
func (sm *SearchManager) RemovePlaceFromSearchIndex(placeID string) {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	db.C(global.CollectionSearchIndexPlaces).RemoveId(placeID)
}

// 	AddSearchHistory adds searched terms of users in an object with an array inside it.
func (sm *SearchManager) AddSearchHistory(accountID, keyword string) bool {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	if len(keyword) == 0 {
		return false
	}
	if _, err := db.C(global.CollectionAccountsSearchHistory).UpsertId(
		accountID,
		bson.M{"$push": bson.M{
			"history": bson.M{
				"$each":  []string{keyword},
				"$slice": -500, // TODO:: use Constant,
			},
		}},
	); err != nil {
		log.Warn("Got error", zap.Error(err))
		return false
	}
	return true
}

// GetSearchHistory returns an array of searched queries of user accountID
func (sm *SearchManager) GetSearchHistory(accountID, keyword string) []string {
	//

	dbSession := _MongoSession.Clone()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	searchHistory := struct {
		AccountID string   `bson:"_id"`
		History   []string `bson:"history"`
	}{}
	db.C(global.CollectionAccountsSearchHistory).FindId(
		accountID,
	).Select(bson.M{"history": bson.M{
		"$elemMatch": bson.M{
			"$regex": fmt.Sprintf("%s", keyword), "$options": "i",
		},
	}}).One(&searchHistory)
	return searchHistory.History
}

// 	AccountIDs searches through  ACCOUNTS collection and apply keyword, filter on it query
// 	filter:			AccountSearchFilterUsersEnabled
// 					AccountSearchFilterUsersDisabled
// 					AccountSearchFilterUsers
// 					AccountSearchFilterDevices
// 					AccountSearchFilterAll
func (sm *SearchManager) AccountIDs(filter string) []string {
	//

	dbSession := _MongoSession.Copy()
	db := dbSession.DB(global.DbName)
	defer dbSession.Close()

	q := bson.M{}

	switch filter {
	case AccountSearchFilterUsersEnabled:
		q["acc_type"] = ACCOUNT_TYPE_USER
		q["disabled"] = false
	case AccountSearchFilterUsersDisabled:
		q["acc_type"] = ACCOUNT_TYPE_USER
		q["disabled"] = true
	case AccountSearchFilterUsers:
		q["acc_type"] = ACCOUNT_TYPE_USER
	case AccountSearchFilterDevices:
		q["acc_type"] = ACCOUNT_TYPE_DEVICE
	case AccountSearchFilterAll:
	default:

	}
	Q := db.C(global.CollectionAccounts).Find(q)

	var accountIDs []string
	if err := Q.Select(bson.M{"_id": 1}).All(&accountIDs); err != nil {
		log.Warn("Got error", zap.Error(err))
	}

	return accountIDs
}
