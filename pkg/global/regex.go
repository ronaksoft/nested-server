package global

import (
	"regexp"
)

/*
   Creation Time: 2021 - Aug - 14
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

var (
	RegEXSpamScore, _ = regexp.Compile(`\sscore=[0-9.]*\s`)
)
