package usersyncers

import (
	"net/url"
)

func NewLifestreetSyncer(externalURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dlifestreet%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24%24visitor_cookie%24%24"
	usersyncURL := "//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl="

	return &syncer{
		familyName:          "lifestreet",
		gdprVendorID:        67,
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
