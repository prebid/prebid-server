package usersync

import (
	"fmt"
	"net/url"
)

func NewLifestreetSyncer(externalURL string) Usersyncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=lifestreet&uid=$$visitor_cookie$$", externalURL)
	usersyncURL := "//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl="

	return &syncer{
		familyName:   "lifestreet",
		gdprVendorID: 67,
		syncInfo: &UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
