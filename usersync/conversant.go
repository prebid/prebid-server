package usersync

import (
	"fmt"
	"net/url"
)

func NewConversantSyncer(usersyncURL string, externalURL string) Usersyncer {
	redirectURI := fmt.Sprintf("%s/setuid?bidder=conversant&uid=", externalURL)

	return &syncer{
		familyName: "conversant",
		syncInfo: &UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectURI)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
