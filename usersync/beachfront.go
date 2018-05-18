package usersync

import (
	"fmt"
	"net/url"
)

func NewBeachfrontSyncer(usersyncURL string, external string) Usersyncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=beachfront&uid=$UID", external)
	url := fmt.Sprintf("%s?redirect=%s", usersyncURL, url.QueryEscape(redirect_uri))

	return &syncer{
		familyName: "beachfront",
		syncInfo: &UsersyncInfo{
			URL:         url,
			Type:        "redirect",
			SupportCORS: true,
		},
	}
}
