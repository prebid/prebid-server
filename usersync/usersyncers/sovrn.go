package usersyncers

import (
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/usersync"
)

func NewSovrnSyncer(externalURL string, usersyncURL string) *syncer {
	redirectURI := fmt.Sprintf("%s/setuid?bidder=sovrn&uid=$UID", externalURL)

	return &syncer{
		familyName:   "sovrn",
		gdprVendorID: 13,
		syncInfo: &usersync.UsersyncInfo{
			URL:         fmt.Sprintf("%sredir=%s", usersyncURL, url.QueryEscape(redirectURI)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
