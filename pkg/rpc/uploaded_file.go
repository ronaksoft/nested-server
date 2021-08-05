package rpc

import (
	"git.ronaksoft.com/nested/server/nested"
)

/*
   Creation Time: 2021 - Aug - 05
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type UploadedFile struct {
	Type                string             `json:"type"`
	Name                string             `json:"name"`
	Size                int64              `json:"size"`
	Thumbs              nested.Picture     `json:"thumbs,omitempty"`
	UniversalId         nested.UniversalID `json:"universal_id"`
	ExpirationTimestamp uint64             `json:"expiration_timestamp"`
}
