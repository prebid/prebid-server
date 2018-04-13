package usersync

import (
	"fmt"
	"net/url"
	"strings"
)

const USER_SYNC_URL = "http://apex.go.sonobi.com"

func NewSonobiSyncer(externalURL string) Usersyncer {
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURL := fmt.Sprintf("%s/setuid?bidder=sonobi&uid=${UID}", externalURL)

	return &syncer{
		familyName: "sonobi",
		syncInfo: &UsersyncInfo{
			URL:         fmt.Sprintf(USER_SYNC_URL, url.QueryEscape(redirectURL)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
