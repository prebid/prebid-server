package usersyncers

import (
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/usersync"
)

func NewEPlanningSyncer(usersyncURL string, externalURL string) *syncer {
	redirectUri := fmt.Sprintf("%s/setuid?bidder=eplanning&uid=$UID", externalURL)

	info := &usersync.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectUri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &syncer{
		familyName: "eplanning",
		syncInfo:   info,
	}
}
