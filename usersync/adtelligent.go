package usersync

import (
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/pbs"
)

func NewAdtelligentSyncer(externalURL string) Usersyncer {

	redirectURI := fmt.Sprintf("%s/setuid?bidder=adtelligent&uid={uid}", externalURL)
	usersyncURL := "//sync.adtelligent.com/csync?t=p&ep=0&redir="

	return &syncer{
		familyName: "adtelligent",
		syncInfo: &pbs.UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectURI)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
