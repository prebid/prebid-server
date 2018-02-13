package usersync

import (
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/pbs"
)

func NewAppnexusSyncer(externalURL string) Usersyncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=adnxs&uid=$UID", externalURL)
	usersyncURL := "//ib.adnxs.com/getuid?"

	return &syncer{
		familyName: "adnxs",
		syncInfo: &pbs.UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
