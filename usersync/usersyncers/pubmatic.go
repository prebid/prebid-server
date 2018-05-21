package usersyncers

import (
	"net/url"
)

func NewPubmaticSyncer(externalURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dpubmatic%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D"
	usersyncURL := "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect="

	return &syncer{
		familyName:          "pubmatic",
		gdprVendorID:        76,
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURI),
		syncType:            SyncTypeIframe,
	}
}
