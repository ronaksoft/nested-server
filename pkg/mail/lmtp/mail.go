package lmtp

import (
	"git.ronaksoft.com/nested/server/nested"
)

/*
   Creation Time: 2021 - Aug - 07
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type NestedMail struct {
	SenderID         string
	SenderName       string
	SenderPic        nested.Picture
	ReplyTo          string
	RawUniversalID   nested.UniversalID
	NonBlindPlaceIDs []string
	NonBlindTargets  []string
	BlindPlaceIDs    []string
	AttachOwners     []string
}
