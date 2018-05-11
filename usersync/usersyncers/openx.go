package usersyncers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/usersync"
)

func NewOpenxSyncer(externalURL string) *syncer {
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURL := fmt.Sprintf("%s/setuid?bidder=openx&uid=${UID}", externalURL)

	return &syncer{
		familyName:   "openx",
		gdprVendorID: 69,
		syncInfo: &usersync.UsersyncInfo{
			URL:         fmt.Sprintf("https://rtb.openx.net/sync/prebid?r=%s", url.QueryEscape(redirectURL)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
