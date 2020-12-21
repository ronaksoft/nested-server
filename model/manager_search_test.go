package nested_test

import (
	nested "git.ronaksoft.com/nested/server/model"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	. "github.com/smartystreets/goconvey/convey"
	"strings"
	"testing"
)

/*
   Creation Time: 2020 - Dec - 21
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

func TestSearchManager_Posts(t *testing.T) {
	Convey("Search/Posts", t, func(c C) {
		accountID := strings.ToLower(tools.RandomID(10))
		b := _Manager.Account.CreateUser(
			accountID, tools.RandomID(10), tools.RandomDigit(10), "IR", "Firstname", "LN",
			"", "", "male",
		)
		c.So(b, ShouldBeTrue)
		_ = _Manager.Place.CreatePersonalPlace(nested.PlaceCreateRequest{
			ID:            accountID,
			AccountID:     accountID,
			Name:          accountID,
			Description:   "",
			GrandParentID: "",
		})

		for i := 0; i < 10; i++ {
			post := _Manager.Post.AddPost(nested.PostCreateRequest{
				PlaceIDs:    []string{accountID},
				Recipients:  []string{},
				ContentType: "text/plain",
				SenderID:    accountID,
				SystemData: nested.PostSystemData{
					NoComment: true,
				},
				Body: "Something",
				Subject: "Some Subject",
			})
			c.So(post.Body, ShouldEqual, "Something")
		}

		_Manager.Search.Posts("hello", accountID, nil, nil, nil, false, nested.Pagination{
			After:  10000000,
			Before: 20000000,
		})
	})

}
