package usersync

import (
	"fmt"
	"net/url"
)

func NewEPlanningSyncer(usersyncURL string, externalURL string) Usersyncer {
	redirectUri := fmt.Sprintf("%s/setuid?bidder=eplanning&uid=$UID", externalURL)

	info := &UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectUri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &syncer{
		familyName: "eplanning",
		syncInfo:   info,
	}
}
