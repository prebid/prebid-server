package usersyncers

import (
	"fmt"
	"net/url"
)

func NewPubmaticSyncer(externalURL string) *syncer {
	redirectUri := fmt.Sprintf("%s/setuid?bidder=pubmatic&uid=", externalURL)
	usersyncURL := "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect="

	return &syncer{
		familyName:          "pubmatic",
		gdprVendorID:        76,
		syncEndpointBuilder: constEndpoint(fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectUri))),
		syncType:            SyncTypeIframe,
	}
}
