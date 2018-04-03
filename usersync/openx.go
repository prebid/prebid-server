package usersync

import (
	"fmt"
	"net/url"
	"strings"
)

func NewOpenxSyncer(externalURL string) Usersyncer {
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURL := fmt.Sprintf("%s/setuid?bidder=openx&uid=${UID}", externalURL)

	return &syncer{
		familyName: "openx",
		syncInfo: &UsersyncInfo{
			URL:         fmt.Sprintf("https://rtb.openx.net/sync/prebid?r=%s", url.QueryEscape(redirectURL)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
