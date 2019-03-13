package rpc

import "errors"

/*
   Creation Time: 2019 - Mar - 13
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2018
*/

var (
	ErrNoHandler     = errors.New("no handler submitted")
	ErrTimeout       = errors.New("time out")
	ErrIncorrectSize = errors.New("incorrect size")
)
