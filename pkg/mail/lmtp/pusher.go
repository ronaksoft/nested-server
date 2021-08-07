package lmtp

import (
	"fmt"
	"net/http"
	"strings"
)

/*
   Creation Time: 2021 - Aug - 07
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/
type pusherClient struct {
	url         string
	apiKey      string
	insecureTls bool
	c           http.Client
}

func newPusherClient(url, apiKey string, insecure bool) (*pusherClient, error) {
	c := &pusherClient{
		url:         strings.TrimRight(url, "/"),
		apiKey:      apiKey,
		insecureTls: insecure,
	}
	return c, nil
}

func (pc *pusherClient) PlaceActivity(placeID string, action int) {
	_, _ = pc.c.Get(fmt.Sprintf("%s/system/pusher/place_activity/%s/%s/%d", pc.url, pc.apiKey, placeID, action))
}
