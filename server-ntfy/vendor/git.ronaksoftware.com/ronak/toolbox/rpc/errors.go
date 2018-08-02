package rpc

import "errors"

/*
    Creation Time: 2018 - Apr - 07
    Created by:  Ehsan N. Moosa (ehsan)
    Maintainers:
        1.  Ehsan N. Moosa (ehsan)
    Auditor: Ehsan N. Moosa
    Copyright Ronak Software Group 2018
*/


var (
    ErrNoHandler = errors.New("no handler submitted")
    ErrTimeout = errors.New("time out")
    ErrIncorrectSize = errors.New("incorrect size")
)
