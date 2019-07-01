package nested

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/globalsign/mgo"
	"log"
	"strings"
	"time"

	"github.com/globalsign/mgo/bson"
)

/*
   Creation Time: 2018 - May - 13
   Created by:  (ehsan)
   Maintainers:
       1.  (ehsan)
   Auditor: Ehsan N. Moosa
   Copyright Ronak Software Group 2018
*/

// SyncPlaceCounters
// This routine iterate over COLLECTION_PLACES and for each Place:
//  1. Count all the posts
//  2. Count all the children places and if it is a GrandPlace then
//  3. Count all Unlocked children places
// In the end, it updates Place counters: KEY_HOLDERS, CREATORS, CHILDS, POSTS
func SyncPlaceCounters() {
	log.Println("--> Routine:: SyncPlaceCounters")
	defer log.Println("<-- Routine:: SyncPlaceCounters")
	place := new(Place)

	iter := _MongoDB.C(COLLECTION_PLACES).Find(bson.M{}).Iter()
	defer iter.Close()
	for iter.Next(place) {
		numberOfPosts, _ := _MongoDB.C(COLLECTION_POSTS).Find(bson.M{"places": place.ID}).Count()
		numberOfChildren, _ := _MongoDB.C(COLLECTION_PLACES).Find(bson.M{
			"_id": bson.M{
				"$regex":   fmt.Sprintf("^%s\\.[^\\.]*$", strings.Replace(place.ID, ".", "\\.", -1)),
				"$options": "i",
			},
		}).Count()
		if place.IsGrandPlace() {
			numberOfUnlockedChildren, _ := _MongoDB.C(COLLECTION_PLACES).Find(bson.M{
				"_id": bson.M{
					"$regex":   fmt.Sprintf("^%s\\.[^\\.]*$", strings.Replace(place.ID, ".", "\\.", -1)),
					"$options": "i",
				},
				"privacy.locked": false,
			}).Count()
			_MongoDB.C(COLLECTION_PLACES).UpdateId(
				place.ID,
				bson.M{
					"$set": bson.M{
						"counters.creators":        len(place.CreatorIDs),
						"counters.key_holders":     len(place.KeyholderIDs),
						"counters.posts":           numberOfPosts,
						"counters.childs":          numberOfChildren,
						"counters.unlocked_childs": numberOfUnlockedChildren,
					},
				},
			)
		} else {
			_MongoDB.C(COLLECTION_PLACES).UpdateId(
				place.ID,
				bson.M{
					"$set": bson.M{
						"counters.creators":    len(place.CreatorIDs),
						"counters.key_holders": len(place.KeyholderIDs),
						"counters.posts":       numberOfPosts,
						"counters.childs":      numberOfChildren,
					},
				},
			)
		}

	}

}

// SyncPostCounters
// This routine iterate over COLLECTION_POSTS and for each Post:
//  1. Count all the comments
func SyncPostCounters() {
	log.Println("--> Routine:: SyncPostCounters")
	defer log.Println("<-- Routine:: SyncPostCounters")
	post := new(Post)
	iter := _MongoDB.C(COLLECTION_POSTS).Find(bson.M{}).Iter()
	defer iter.Close()
	for iter.Next(post) {
		n, _ := _MongoDB.C(COLLECTION_POSTS_COMMENTS).Find(bson.M{
			"post_id":  post.ID,
			"_removed": false,
		}).Count()
		_MongoDB.C(COLLECTION_POSTS).UpdateId(
			post.ID,
			bson.M{
				"$set": bson.M{
					"counters.comments": n,
					"counters.attaches": len(post.AttachmentIDs),
					"counters.labels":   len(post.LabelIDs),
				},
			},
		)
	}
}

// SyncLabelCounters
// This routine iterates over COLLECTION_LABELS and for each Label:
//  1. Count Posts and Tasks which has that label
func SyncLabelCounters() {
	log.Println("--> Routine:: SyncLabelCounters")
	defer log.Println("<-- Routine:: SyncLabelCounters")
	label := new(Label)
	iter := _MongoDB.C(COLLECTION_LABELS).Find(bson.M{}).Iter()
	defer iter.Close()
	for iter.Next(label) {
		nPost, _ := _MongoDB.C(COLLECTION_POSTS).Find(bson.M{
			"labels":   label.ID,
			"_removed": false,
		}).Count()
		nTasks, _ := _MongoDB.C(COLLECTION_TASKS).Find(bson.M{
			"labels":   label.ID,
			"_removed": false,
		}).Count()
		_MongoDB.C(COLLECTION_LABELS).UpdateId(
			label.ID,
			bson.M{
				"$set": bson.M{
					"counters.posts":   nPost + nTasks,
					"counters.members": len(label.Members),
				},
			},
		)
	}
}

// SyncTaskCounters
// This routine iterates over COLLECTION_TASKS and for each task:
//  1. Count Comments
func SyncTaskCounters() {
	log.Println("--> Routine:: SyncTaskCounters")
	defer log.Println("<-- Routine:: SyncTaskCounters")
	task := new(Task)
	iter := _MongoDB.C(COLLECTION_TASKS).Find(bson.M{}).Iter()
	defer iter.Close()
	for iter.Next(task) {
		numberOfComments, _ := _MongoDB.C(COLLECTION_TASKS_ACTIVITIES).Find(bson.M{
			"task_id": task.ID,
			"action":  TASK_ACTIVITY_COMMENT,
		}).Count()

		_MongoDB.C(COLLECTION_TASKS).UpdateId(
			task.ID,
			bson.M{
				"$set": bson.M{
					"counters.comments":    numberOfComments,
					"counters.labels":      len(task.LabelIDs),
					"counters.watchers":    len(task.WatcherIDs),
					"counters.editors":     len(task.EditorIDs),
					"counters.candidates":  len(task.CandidateIDs),
					"counters.attachments": len(task.AttachmentIDs),
				},
			},
		)
	}
}

// SyncFileRefCounters
// This routine iterates over COLLECTION_POSTS and COLLECTIONS_TASKS and update the ref_count of the
// files in the COLLECTION_FILES
func SyncFileRefCounters() {
	log.Println("--> Routine:: SyncFileRefCounters")
	defer log.Println("<-- Routine:: SyncFileRefCounters")

	_MongoDB.C(COLLECTION_FILES).UpdateAll(bson.M{}, bson.M{"$set": bson.M{"ref_count": 0}})
	iter1 := _MongoDB.C(COLLECTION_POSTS).Find(bson.M{}).Iter()
	post := new(Post)
	for iter1.Next(post) {
		_MongoDB.C(COLLECTION_FILES).UpdateAll(
			bson.M{"_id": bson.M{"$in": post.AttachmentIDs}},
			bson.M{"$inc": bson.M{"ref_count": 1}},
		)
		for _, attachmentID := range post.AttachmentIDs {
			_MongoDB.C(COLLECTION_POSTS_FILES).Insert(
				bson.M{"universal_id": attachmentID, "post_id": post.ID},
			)
		}

	}
	iter1.Close()
	iter2 := _MongoDB.C(COLLECTION_TASKS).Find(bson.M{}).Iter()
	task := new(Task)
	for iter2.Next(task) {
		_MongoDB.C(COLLECTION_FILES).UpdateAll(
			bson.M{"_id": bson.M{"$in": task.AttachmentIDs}},
			bson.M{"$inc": bson.M{"ref_count": 1}},
		)
		for _, attachmentID := range task.AttachmentIDs {
			_MongoDB.C(COLLECTION_TASKS_FILES).Insert(
				bson.M{"universal_id": attachmentID, "task_id": task.ID},
			)
		}
	}
	iter2.Close()
	iter3 := _MongoDB.C(COLLECTION_POSTS_COMMENTS).Find(bson.M{}).Iter()
	defer iter3.Close()
	comment := new(Comment)
	for iter3.Next(comment) {
		if comment.AttachmentID != "" {
			_MongoDB.C(COLLECTION_POSTS_COMMENTS).Update(
				bson.M{"_id": comment.AttachmentID},
				bson.M{"$inc": bson.M{"ref_count": 1}},
			)
		}
		_MongoDB.C(COLLECTION_POSTS_FILES).Insert(
			bson.M{"universal_id": comment.AttachmentID, "post_id": comment.PostID},
		)
	}
	iter3.Close()
}

// SyncSystemCounters
// This routines counts all the accounts (active, de-active), places (grand places, locked places, unlocked places)
func SyncSystemCounters() {
	log.Println("--> Routine:: SyncSystemCounters")
	defer log.Println("<-- Routine:: SyncSystemCounters")

	enabledAccounts, _ := _MongoDB.C(COLLECTION_ACCOUNTS).Find(bson.M{"disabled": false}).Count()
	disabledAccounts, _ := _MongoDB.C(COLLECTION_ACCOUNTS).Find(bson.M{"disabled": true}).Count()
	personalPlaces, _ := _MongoDB.C(COLLECTION_PLACES).Find(bson.M{
		"type": PLACE_TYPE_PERSONAL,
	}).Count()
	grandPlaces, _ := _MongoDB.C(COLLECTION_PLACES).Find(bson.M{
		"level": 0,
		"type":  PLACE_TYPE_SHARED,
	}).Count()
	lockedPlaces, _ := _MongoDB.C(COLLECTION_PLACES).Find(bson.M{
		"privacy.locked": true,
		"level":          bson.M{"$ne": 0},
		"type":           PLACE_TYPE_SHARED,
	}).Count()
	unLockedPlaces, _ := _MongoDB.C(COLLECTION_PLACES).Find(bson.M{
		"privacy.locked": false,
		"level":          bson.M{"$ne": 0},
		"type":           PLACE_TYPE_SHARED,
	}).Count()
	_Manager.System.SetCounter(
		MI{
			SYSTEM_COUNTERS_ENABLED_ACCOUNTS:  enabledAccounts,
			SYSTEM_COUNTERS_DISABLED_ACCOUNTS: disabledAccounts,
			SYSTEM_COUNTERS_PERSONAL_PLACES:   personalPlaces,
			SYSTEM_COUNTERS_GRAND_PLACES:      grandPlaces,
			SYSTEM_COUNTERS_LOCKED_PLACES:     lockedPlaces,
			SYSTEM_COUNTERS_UNLOCKED_PLACES:   unLockedPlaces,
		},
	)
}

// CleanupSessions
func CleanupSessions() {
	log.Println("--> Routine:: CleanupSessions")
	defer log.Println("<-- Routine:: CleanupSessions")
	_MongoDB.C(COLLECTION_SESSIONS).RemoveAll(bson.M{"expired": true})
}

// CleanupTasks
func CleanupTasks() {
	log.Println("--> Routine:: CleanupTasks")
	defer log.Println("<-- Routine:: CleanupTasks")

	iter := _MongoDB.C(COLLECTION_TASKS).Find(bson.M{"_removed": true}).Iter()
	defer iter.Close()
	task := new(Task)
	for iter.Next(task) {
		_MongoDB.C(COLLECTION_TASKS_ACTIVITIES).RemoveAll(
			bson.M{"task_id": task.ID},
		)

	}
	_MongoDB.C(COLLECTION_TASKS).RemoveAll(bson.M{"_removed": true})
}

// CleanupPosts
func CleanupPosts() {
	log.Println("--> Routine:: CleanupPosts")
	defer log.Println("<-- Routine:: CleanupPosts")

	iter := _MongoDB.C(COLLECTION_POSTS).Find(bson.M{"_removed": true}).Iter()
	defer iter.Close()
	task := new(Task)
	for iter.Next(task) {
		_MongoDB.C(COLLECTION_POSTS_ACTIVITIES).RemoveAll(
			bson.M{"post_id": task.ID},
		)
		_MongoDB.C(COLLECTION_POSTS_COMMENTS).RemoveAll(
			bson.M{"post_id": task.ID},
		)
	}
	_MongoDB.C(COLLECTION_POSTS).RemoveAll(bson.M{"_removed": true})
}

// CleanupTempFiles
// Cleanup Temporary Files
func CleanupTempFiles() {
	log.Println("--> Routine:: CleanupTempFiles")
	defer log.Println("<-- Routine:: CleanupTempFiles")

	iter := _MongoDB.C(COLLECTION_FILES).Find(bson.M{}).Iter()
	defer iter.Close()
	file := new(FileInfo)
	for iter.Next(file) {
		uploadTime := time.Unix(int64(file.UploadTimestamp)/1000, 0)
		if file.Status == FILE_STATUS_TEMP {
			if time.Now().Sub(uploadTime).Hours() > 24 {
				log.Println("File Removed:", file.ID, file.Filename, uploadTime.String())
				_MongoStore.RemoveId(file.ID)
				_MongoDB.C(COLLECTION_FILES).RemoveId(file.ID)
			}
		}
	}
}

// Fix file status of task attached files
func FixReferredTmpFiles() {
	log.Println("--> Routine:: FixReferedTmpFiles")
	defer log.Println("<-- Routine:: FixReferedTmpFiles")
	iter := _MongoDB.C(COLLECTION_FILES).Find(bson.M{"ref_count": bson.M{"$gt": 0}}).Iter()
	defer iter.Close()
	file := new(FileInfo)
	for iter.Next(file) {
		if file.Status == FILE_STATUS_TEMP {
			_MongoDB.C(COLLECTION_FILES).Update(
				bson.M{"_id": file.ID},
				bson.M{"$set": bson.M{"status": FILE_STATUS_ATTACHED}},
			)
		}
	}
}

// Fix file status of task attached files
func FixSearchIndexPlacesCollection() {
	log.Println("--> Routine:: FixSearchIndexPlacesCollection")
	defer log.Println("<-- Routine:: FixSearchIndexPlacesCollection")
	if err := _MongoDB.C(COLLECTION_SEARCH_INDEX_PLACES).DropCollection(); err != nil {
		_Log.Warn(err.Error())
	}
	_ = _MongoDB.C(COLLECTION_SEARCH_INDEX_PLACES).EnsureIndex(mgo.Index{Key: []string{"name"}, Background: true})
	iter := _MongoDB.C(COLLECTION_PLACES).Find(bson.M{}).Iter()
	place := new(Place)
	for iter.Next(place) {
		if place.Privacy.Search {
			if err := _MongoDB.C(COLLECTION_SEARCH_INDEX_PLACES).Insert(bson.M{"_id": place.ID, "name": place.Name, "picture": place.Picture}); err != nil {
				_Log.Warn(err.Error())
			}
		}
	}
	iter.Close()
}

func AddContentToPost() {
	log.Println("--> Routine:: AddContentToPost")
	defer log.Println("<-- Routine:: AddContentToPost")
	err := _MongoDB.C(COLLECTION_POSTS).DropIndexName("body")
	if err != nil {
		_Log.Warn(err.Error())
	}
	iter := _MongoDB.C(COLLECTION_POSTS).Find(bson.M{}).Iter()
	defer iter.Close()
	p := new(Post)

	for iter.Next(p) {
		var postContent string
		switch p.ContentType {
		case CONTENT_TYPE_TEXT_PLAIN:
			postContent = p.Body
		case CONTENT_TYPE_TEXT_HTML:
			reader := strings.NewReader(p.Body)
			doc, _ := goquery.NewDocumentFromReader(reader)
			doc.Find("").Each(func(i int, el *goquery.Selection) {
				el.Remove()
			})
			postContent = doc.Text()
		default:
			continue
		}
		err := _MongoDB.C(COLLECTION_POSTS).UpdateId(p.ID, bson.M{"$set": bson.M{"content": postContent}})
		if err != nil {
			_Log.Warn(err.Error())
		}
	}
}
