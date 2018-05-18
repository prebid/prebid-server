package usersyncers

import (
	"net/url"
)

func NewPubmaticSyncer(externalURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dpubmatic%26uid%3D"
	usersyncURL := "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect="

	return &syncer{
		familyName:          "pubmatic",
		gdprVendorID:        76,
		syncEndpointBuilder: constEndpoint(usersyncURL + redirectURI),
		syncType:            SyncTypeIframe,
	}
}
