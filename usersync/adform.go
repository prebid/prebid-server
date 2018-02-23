package usersync

import (
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/pbs"
)

func NewAdformSyncer(usersyncURL string, externalURL string) Usersyncer {
	redirectUri := fmt.Sprintf("%s/setuid?bidder=adform&uid=$UID", externalURL)

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectUri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &syncer{
		familyName: "adform",
		syncInfo:   info,
	}
}
