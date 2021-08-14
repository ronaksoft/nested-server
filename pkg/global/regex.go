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
	RegExSpamScore, _    = regexp.Compile(`\sscore=[0-9.]*\s`)
	RegExMention, _      = regexp.Compile(`@([a-zA-Z0-9-]*)(\s|$)`)
	RegExPlaceID, _      = regexp.Compile("^[a-zA-Z][a-zA-Z0-9-_]{0,30}[a-zA-Z0-9]$")
	RegExGrandPlaceID, _ = regexp.Compile("^[a-zA-Z][a-zA-Z0-9-_]{1,30}[a-zA-Z0-9]$")
	RegExAccountID, _    = regexp.Compile("^[a-zA-Z][a-zA-Z0-9-_]{1,30}[a-zA-Z0-9]$")
	RegExEmail, _        = regexp.Compile("^[a-z0-9._%+\\-]+@[a-z0-9.\\-]+\\.[a-z]{2,4}$")
)
