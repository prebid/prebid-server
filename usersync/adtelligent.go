package usersync

import (
	"fmt"
	"net/url"
)

func NewAdtelligentSyncer(externalURL string) Usersyncer {

	redirectURI := fmt.Sprintf("%s/setuid?bidder=adtelligent&uid={uid}", externalURL)
	usersyncURL := "//sync.adtelligent.com/csync?t=p&ep=0&redir="

	return &syncer{
		familyName: "adtelligent",
		syncInfo: &UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectURI)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
