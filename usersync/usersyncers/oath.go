package usersyncers

import (
	"fmt"
	"github.com/prebid/prebid-server/usersync"
	"net/url"
	"strings"
)

func NewOathSyncer(usersyncURL string, externalURL string) *syncer {
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURL := fmt.Sprintf("%s/setuid?bidder=oath&uid=${UID}", externalURL)

	return &syncer{
		familyName:   "oath",
		gdprVendorID: 25,
		syncInfo: &usersync.UsersyncInfo{
			URL:         fmt.Sprintf(usersyncURL, url.QueryEscape(redirectURL)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}

}
