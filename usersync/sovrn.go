package usersync

import (
	"fmt"
	"net/url"
)

func NewSovrnSyncer(externalURL string, usersyncURL string) Usersyncer {
	redirectURI := fmt.Sprintf("%s/setuid?bidder=sovrn&uid=$UID", externalURL)

	return &syncer{
		familyName:   "sovrn",
		gdprVendorID: 13,
		syncInfo: &UsersyncInfo{
			URL:         fmt.Sprintf("%sredir=%s", usersyncURL, url.QueryEscape(redirectURI)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
