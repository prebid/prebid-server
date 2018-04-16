package usersync

import (
	"fmt"
	"net/url"
)

const USER_SYNC_URL = "http://sync.go.sonobi.com/us.gif?loc=%s"

func NewSonobiSyncer(externalURL string) Usersyncer {
	redirectURI := fmt.Sprintf("%s/setuid?bidder=sonobi&uid=$UID}", externalURL)
	return &syncer{
		familyName: "sonobi",
		syncInfo: &UsersyncInfo{
			URL:         fmt.Sprintf(USER_SYNC_URL, url.QueryEscape(redirectURI)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
