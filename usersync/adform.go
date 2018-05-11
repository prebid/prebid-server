package usersync

import (
	"fmt"
	"net/url"
)

func NewAdformSyncer(usersyncURL string, externalURL string) Usersyncer {
	redirectUri := fmt.Sprintf("%s/setuid?bidder=adform&uid=$UID", externalURL)

	info := &UsersyncInfo{
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
