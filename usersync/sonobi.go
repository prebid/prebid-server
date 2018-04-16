package usersync

import (
	"fmt"
	"net/url"
)

const USER_SYNC_URL = "http://sync.go.sonobi.com/us.gif?loc=%s"

func NewSonobiSyncer(externalURL string) Usersyncer {
	return &syncer{
		familyName: "sonobi",
		syncInfo: &UsersyncInfo{
			URL:         fmt.Sprintf(USER_SYNC_URL, url.QueryEscape(externalURL)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
