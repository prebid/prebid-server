package usersyncers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/usersync"
)

func NewSomoaudienceSyncer(externalURL string) *syncer {
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURL := fmt.Sprintf("%s/setuid?bidder=somoaudience&uid=${UID}", externalURL)

	usersyncURL := "//publisher-east.mobileadtrading.com/usersync?gdprg=1&ru="

	return &syncer{
		familyName: "somoaudience",
		syncInfo: &usersync.UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectURL)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
