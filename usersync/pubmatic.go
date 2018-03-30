package usersync

import (
	"fmt"
	"net/url"
)

func NewPubmaticSyncer(externalURL string) Usersyncer {
	redirectUri := fmt.Sprintf("%s/setuid?bidder=pubmatic&uid=", externalURL)
	usersyncURL := "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect="

	return &syncer{
		familyName: "pubmatic",
		syncInfo: &UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectUri)),
			Type:        "iframe",
			SupportCORS: false,
		},
	}
}
