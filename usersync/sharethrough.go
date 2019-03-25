package usersync

import (
	"fmt"
	"net/url"
)

func NewSharethroughSyncer(externalURL string) Usersyncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=sharethrough&uid=$UID", externalURL)
	usersyncURL := "//sharethrough.adnxs.com/getuid?"
	return &syncer{
		familyName:   "adnxs",
		gdprVendorID: 80,
		syncInfo: &UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
