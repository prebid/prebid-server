package usersync

import (
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/pbs"
)

func NewSovrnSyncer(externalURL string, usersyncURL string) Usersyncer {
	redirectURI := fmt.Sprintf("%s/setuid?bidder=sovrn&uid=$UID", externalURL)

	return &syncer{
		familyName: "sovrn",
		syncInfo: &pbs.UsersyncInfo{
			URL:         fmt.Sprintf("%sredir=%s", usersyncURL, url.QueryEscape(redirectURI)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
