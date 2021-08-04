package nestedGateway

import (
	"git.ronaksoft.com/nested/server/nested"
)

type UploadedFile struct {
	Type                string             `json:"type"`
	Name                string             `json:"name"`
	Size                int64              `json:"size"`
	Thumbs              nested.Picture     `json:"thumbs,omitempty"`
	UniversalId         nested.UniversalID `json:"universal_id"`
	ExpirationTimestamp uint64             `json:"expiration_timestamp"`
}
type UploadOutput struct {
	Files []UploadedFile `json:"files"`
}
type UploadResponse struct {
	Payload UploadOutput `json:"data"`
}
