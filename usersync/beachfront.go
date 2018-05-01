package usersync

import (
	"fmt"
	"net/url"
)

func NewBeachfrontSyncer(external string) Usersyncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=beachfront&uid=$UID", external)
	// usersyncURL := "//sync.bfmio.com?url="
	usersyncURL := "http://10.0.0.181/fakesync.html?nothing="

	url := fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri))

	return &syncer{
		familyName: "beachfront",
		syncInfo: &UsersyncInfo{
			URL:         url,
			Type:        "redirect",
			SupportCORS: true,
		},
	}
}
