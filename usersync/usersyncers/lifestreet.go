package usersyncers

import (
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/usersync"
)

func NewLifestreetSyncer(externalURL string) *syncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=lifestreet&uid=$$visitor_cookie$$", externalURL)
	usersyncURL := "//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl="

	return &syncer{
		familyName:   "lifestreet",
		gdprVendorID: 67,
		syncInfo: &usersync.UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
