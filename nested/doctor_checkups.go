package nested

import (
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"github.com/PuerkitoBio/goquery"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"go.uber.org/zap"
	"strings"
	"time"
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
// This routine iterate over global.CollectionPlaces and for each Place:
//  1. Count all the posts
//  2. Count all the children places and if it is a GrandPlace then
//  3. Count all Unlocked children places
// In the end, it updates Place counters: KEY_HOLDERS, CREATORS, CHILDS, POSTS
func SyncPlaceCounters() {
	place := new(Place)
	iter := _MongoDB.C(global.CollectionPlaces).Find(bson.M{}).Iter()
	defer iter.Close()
	for iter.Next(place) {
		numberOfPosts, _ := _MongoDB.C(global.CollectionPosts).Find(bson.M{"places": place.ID}).Count()
		numberOfChildren, _ := _MongoDB.C(global.CollectionPlaces).Find(bson.M{
			"_id": bson.M{
				"$regex":   fmt.Sprintf("^%s\\.[^\\.]*$", strings.Replace(place.ID, ".", "\\.", -1)),
				"$options": "i",
			},
		}).Count()
		if place.IsGrandPlace() {
			numberOfUnlockedChildren, _ := _MongoDB.C(global.CollectionPlaces).Find(bson.M{
				"_id": bson.M{
					"$regex":   fmt.Sprintf("^%s\\.[^\\.]*$", strings.Replace(place.ID, ".", "\\.", -1)),
					"$options": "i",
				},
				"privacy.locked": false,
			}).Count()
			_MongoDB.C(global.CollectionPlaces).UpdateId(
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
			_MongoDB.C(global.CollectionPlaces).UpdateId(
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
// This routine iterate over global.CollectionPosts and for each Post:
//  1. Count all the comments
func SyncPostCounters() {
	post := new(Post)
	iter := _MongoDB.C(global.CollectionPosts).Find(bson.M{}).Iter()
	defer iter.Close()
	for iter.Next(post) {
		n, _ := _MongoDB.C(global.CollectionPostsComments).Find(bson.M{
			"post_id":  post.ID,
			"_removed": false,
		}).Count()
		_MongoDB.C(global.CollectionPosts).UpdateId(
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
// This routine iterates over global.CollectionLabels and for each Label:
//  1. Count Posts and Tasks which has that label
func SyncLabelCounters() {
	label := new(Label)
	iter := _MongoDB.C(global.CollectionLabels).Find(bson.M{}).Iter()
	defer iter.Close()
	for iter.Next(label) {
		nPost, _ := _MongoDB.C(global.CollectionPosts).Find(bson.M{
			"labels":   label.ID,
			"_removed": false,
		}).Count()
		nTasks, _ := _MongoDB.C(global.CollectionTasks).Find(bson.M{
			"labels":   label.ID,
			"_removed": false,
		}).Count()
		_MongoDB.C(global.CollectionLabels).UpdateId(
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
// This routine iterates over global.CollectionTasks and for each task:
//  1. Count Comments
func SyncTaskCounters() {
	task := new(Task)
	iter := _MongoDB.C(global.CollectionTasks).Find(bson.M{}).Iter()
	defer iter.Close()
	for iter.Next(task) {
		numberOfComments, _ := _MongoDB.C(global.CollectionTasksActivities).Find(bson.M{
			"task_id": task.ID,
			"action":  global.TaskActivityComment,
		}).Count()

		_MongoDB.C(global.CollectionTasks).UpdateId(
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

// SyncFileRefCounters iterates over global.CollectionPosts and COLLECTIONS_TASKS and update the ref_count of the
// files in the global.CollectionFiles
func SyncFileRefCounters() {
	_MongoDB.C(global.CollectionFiles).UpdateAll(bson.M{}, bson.M{"$set": bson.M{"ref_count": 0}})
	iter1 := _MongoDB.C(global.CollectionPosts).Find(bson.M{}).Iter()
	post := new(Post)
	for iter1.Next(post) {
		_MongoDB.C(global.CollectionFiles).UpdateAll(
			bson.M{"_id": bson.M{"$in": post.AttachmentIDs}},
			bson.M{"$inc": bson.M{"ref_count": 1}},
		)
		for _, attachmentID := range post.AttachmentIDs {
			_MongoDB.C(global.CollectionPostsFiles).Insert(
				bson.M{"universal_id": attachmentID, "post_id": post.ID},
			)
		}

	}
	iter1.Close()
	iter2 := _MongoDB.C(global.CollectionTasks).Find(bson.M{}).Iter()
	task := new(Task)
	for iter2.Next(task) {
		_MongoDB.C(global.CollectionFiles).UpdateAll(
			bson.M{"_id": bson.M{"$in": task.AttachmentIDs}},
			bson.M{"$inc": bson.M{"ref_count": 1}},
		)
		for _, attachmentID := range task.AttachmentIDs {
			_MongoDB.C(global.CollectionTasksFiles).Insert(
				bson.M{"universal_id": attachmentID, "task_id": task.ID},
			)
		}
	}
	iter2.Close()
	iter3 := _MongoDB.C(global.CollectionPostsComments).Find(bson.M{}).Iter()
	defer iter3.Close()
	comment := new(Comment)
	for iter3.Next(comment) {
		if comment.AttachmentID != "" {
			_MongoDB.C(global.CollectionPostsComments).Update(
				bson.M{"_id": comment.AttachmentID},
				bson.M{"$inc": bson.M{"ref_count": 1}},
			)
		}
		_MongoDB.C(global.CollectionPostsFiles).Insert(
			bson.M{"universal_id": comment.AttachmentID, "post_id": comment.PostID},
		)
	}
	iter3.Close()
}

// SyncSystemCounters counts all the accounts (active, de-active), places (grand places, locked places, unlocked places)
func SyncSystemCounters() {
	enabledAccounts, _ := _MongoDB.C(global.CollectionAccounts).Find(bson.M{"disabled": false}).Count()
	disabledAccounts, _ := _MongoDB.C(global.CollectionAccounts).Find(bson.M{"disabled": true}).Count()
	personalPlaces, _ := _MongoDB.C(global.CollectionPlaces).Find(bson.M{
		"type": PlaceTypePersonal,
	}).Count()
	grandPlaces, _ := _MongoDB.C(global.CollectionPlaces).Find(bson.M{
		"level": 0,
		"type":  PlaceTypeShared,
	}).Count()
	lockedPlaces, _ := _MongoDB.C(global.CollectionPlaces).Find(bson.M{
		"privacy.locked": true,
		"level":          bson.M{"$ne": 0},
		"type":           PlaceTypeShared,
	}).Count()
	unLockedPlaces, _ := _MongoDB.C(global.CollectionPlaces).Find(bson.M{
		"privacy.locked": false,
		"level":          bson.M{"$ne": 0},
		"type":           PlaceTypeShared,
	}).Count()
	_Manager.System.SetCounter(
		MI{
			global.SystemCountersEnabledAccounts:  enabledAccounts,
			global.SystemCountersDisabledAccounts: disabledAccounts,
			global.SystemCountersPersonalPlaces:   personalPlaces,
			global.SystemCountersGrandPlaces:      grandPlaces,
			global.SystemCountersLockedPlaces:     lockedPlaces,
			global.SystemCountersUnlockedPlaces:   unLockedPlaces,
		},
	)
}

func CleanupSessions() {
	_MongoDB.C(global.CollectionSessions).RemoveAll(bson.M{"expired": true})
}

func CleanupTasks() {
	iter := _MongoDB.C(global.CollectionTasks).Find(bson.M{"_removed": true}).Iter()
	defer iter.Close()
	task := new(Task)
	for iter.Next(task) {
		_MongoDB.C(global.CollectionTasksActivities).RemoveAll(
			bson.M{"task_id": task.ID},
		)

	}
	_MongoDB.C(global.CollectionTasks).RemoveAll(bson.M{"_removed": true})
}

func CleanupPosts() {
	iter := _MongoDB.C(global.CollectionPosts).Find(bson.M{"_removed": true}).Iter()
	defer iter.Close()
	task := new(Task)
	for iter.Next(task) {
		_MongoDB.C(global.CollectionPostsActivities).RemoveAll(
			bson.M{"post_id": task.ID},
		)
		_MongoDB.C(global.CollectionPostsComments).RemoveAll(
			bson.M{"post_id": task.ID},
		)
	}
	_MongoDB.C(global.CollectionPosts).RemoveAll(bson.M{"_removed": true})
}

// CleanupTempFiles
// Cleanup Temporary Files
func CleanupTempFiles() {
	iter := _MongoDB.C(global.CollectionFiles).Find(bson.M{}).Iter()
	defer iter.Close()
	file := new(FileInfo)
	for iter.Next(file) {
		uploadTime := time.Unix(int64(file.UploadTimestamp)/1000, 0)
		if file.Status == FileStatusTemp {
			if time.Now().Sub(uploadTime).Hours() > 24 {
				log.Sugar().Info("File Removed:", file.ID, file.Filename, uploadTime.String())
				_MongoStore.RemoveId(file.ID)
				_MongoDB.C(global.CollectionFiles).RemoveId(file.ID)
			}
		}
	}
}

// FixReferredTmpFiles Fix file status of task attached files
func FixReferredTmpFiles() {
	iter := _MongoDB.C(global.CollectionFiles).Find(bson.M{"ref_count": bson.M{"$gt": 0}}).Iter()
	defer iter.Close()
	file := new(FileInfo)
	for iter.Next(file) {
		if file.Status == FileStatusTemp {
			_MongoDB.C(global.CollectionFiles).Update(
				bson.M{"_id": file.ID},
				bson.M{"$set": bson.M{"status": FileStatusAttached}},
			)
		}
	}
}

// FixSearchIndexPlacesCollection Fix file status of task attached files
func FixSearchIndexPlacesCollection() {
	if err := _MongoDB.C(global.CollectionSearchIndexPlaces).DropCollection(); err != nil {
		log.Warn("Got error", zap.Error(err))
	}
	_ = _MongoDB.C(global.CollectionSearchIndexPlaces).EnsureIndex(mgo.Index{Key: []string{"name"}, Background: true})
	iter := _MongoDB.C(global.CollectionPlaces).Find(bson.M{}).Iter()
	place := new(Place)
	for iter.Next(place) {
		if place.Privacy.Search {
			if err := _MongoDB.C(global.CollectionSearchIndexPlaces).Insert(bson.M{"_id": place.ID, "name": place.Name, "picture": place.Picture}); err != nil {
				log.Warn("Got error", zap.Error(err))
			}
		}
	}
	iter.Close()
}

func AddContentToPost() {
	err := _MongoDB.C(global.CollectionTasks).DropIndexName("title_text_description_text_todos_text")
	if err != nil {
		log.Warn("Got error", zap.Error(err))
		fmt.Println("DropIndexName", err)
	}
	_ = _MongoDB.C(global.CollectionTasks).EnsureIndex(mgo.Index{
		Key: []string{"$text:title", "$text:description", "$text:todos.txt"},
		Weights: map[string]int{
			"title":       5,
			"description": 1,
			"todos.txt":   1,
		},
		Background: true,
	})

	err = _MongoDB.C(global.CollectionPosts).DropIndexName("body_text_subject_text")
	if err != nil {
		log.Warn("Got error", zap.Error(err))
		fmt.Println("DropIndexName", err)
	}
	_ = _MongoDB.C(global.CollectionPosts).EnsureIndex(mgo.Index{
		Key: []string{"$text:content", "$text:subject"},
		Weights: map[string]int{
			"subject": 5,
			"content": 1,
		},
		Background: true,
	})

	iter := _MongoDB.C(global.CollectionPosts).Find(bson.M{}).Iter()
	defer iter.Close()
	p := new(Post)

	for iter.Next(p) {
		var postContent string
		switch p.ContentType {
		case ContentTypeTextPlain:
			postContent = p.Body
		case ContentTypeTextHtml:
			reader := strings.NewReader(p.Body)
			doc, _ := goquery.NewDocumentFromReader(reader)
			doc.Find("").Each(func(i int, el *goquery.Selection) {
				el.Remove()
			})
			postContent = doc.Text()
		default:
			continue
		}
		err := _MongoDB.C(global.CollectionPosts).UpdateId(p.ID, bson.M{"$set": bson.M{"content": postContent}})
		if err != nil {
			log.Warn("Got error", zap.Error(err))
		}
	}
}
