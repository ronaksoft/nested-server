package main

import (
	"bytes"
	"fmt"
	nested "git.ronaksoftware.com/nested/server/model"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"strings"
)

const (
	dbName = "nested-20191000001"
)

func main() {
	s, err := mgo.Dial("***REMOVED***/?ssl=true&authSource=nested-20191000001")
	if err != nil {
		panic(err)
	}
	s.SetBatch(5000)
	s.SetPrefetch(0.25)

	// replacePosts(s)
	replaceApps(s)

}

func replacePosts(s *mgo.Session) {
	iter := s.DB(dbName).C(nested.COLLECTION_POSTS).Find(bson.M{"_id": bson.ObjectIdHex("5f0f134209a90a0001c2fb31")}).Select(bson.M{"body": "1"}).Iter()
	var post nested.Post
	var cnt int64
	for iter.Next(&post) {
		for idx := 0; idx < len(post.Body)-20; idx += 20 {
			fmt.Println(post.Body[idx:idx+20])
		}
		fmt.Println(bytes.Contains([]byte(post.Body), []byte("core.tablighgram.com")))
		if idx := strings.Index(post.Body, "core.tablighgram.com"); idx > 0 {
			fmt.Println(post.Body, post.ID.Hex())
			s2 := s.Copy()
			s2.DB(dbName).C(nested.COLLECTION_POSTS).
				UpdateId(
					post.ID,
					bson.M{"$set": bson.M{
						"body": strings.Replace(post.Body, "core.tablighgram.com", "core.tablighdrive.com", -1),
					}},
				)
			s2.Close()
			cnt++
			if cnt%1000 == 0 {
				fmt.Println("Scanned ", cnt)
			}
		}
	}
	_ = iter.Close()
}
func replaceApps(s *mgo.Session) {
	iter := s.DB(dbName).C(nested.COLLECTION_APPS).Find(bson.M{"_id": "_inbox"}).Iter()
	var app nested.App
	for iter.Next(&app) {
		app.Homepage = strings.Replace(app.Homepage, "core.tablighgram.com", "core.tablighdrive.com", 1)
		app.CallbackURL = strings.Replace(app.CallbackURL, "core.tablighgram.com", "core.tablighdrive.com", 1)
		app.IconLargeURL = "https://core.tablighdrive.com/wp-content/uploads/2019/04/inbox.jpg"
		app.IconSmallURL = "https://core.tablighdrive.com/wp-content/uploads/2019/04/inbox.jpg"
		fmt.Println(app)
		s2 := s.Copy()
		s2.DB(dbName).C(nested.COLLECTION_APPS).
			UpsertId(app.ID, app)

		s2.Close()
	}

	_ = iter.Close()
}
