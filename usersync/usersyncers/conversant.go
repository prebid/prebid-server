package usersyncers

import (
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/usersync"
)

func NewConversantSyncer(usersyncURL string, externalURL string) *syncer {
	redirectURI := fmt.Sprintf("%s/setuid?bidder=conversant&uid=", externalURL)

	return &syncer{
		familyName:   "conversant",
		gdprVendorID: 24,
		syncInfo: &usersync.UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectURI)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
