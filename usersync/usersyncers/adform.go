package usersyncers

import (
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/usersync"
)

func NewAdformSyncer(usersyncURL string, externalURL string) *syncer {
	redirectUri := fmt.Sprintf("%s/setuid?bidder=adform&uid=$UID", externalURL)

	info := &usersync.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectUri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &syncer{
		familyName:   "adform",
		gdprVendorID: 50,
		syncInfo:     info,
	}
}
