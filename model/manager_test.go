package nested_test

import (
	nested "git.ronaksoft.com/nested/server/model"
)

/*
   Creation Time: 2020 - Dec - 21
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

var _Manager *nested.Manager
func init() {
	var err error
	_Manager, err = nested.NewManager("TestServer", "mongodb://localhost:27001/nested", "localhost:6379", -1)
	if err != nil {
		panic(err)
	}
}