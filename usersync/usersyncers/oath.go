package usersyncers

import (
	"net/url"
	"strings"
)

func NewOathSyncer(userSyncURL string, externalURL string) *syncer {
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Doath%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24%7BUID%7D"

	return &syncer{
		familyName:          "oath",
		gdprVendorID:        25,
		syncEndpointBuilder: resolveMacros("http://east-bid.ybp.yahoo.com/sync/appnexuspbs?url=" + redirectURL),
		syncType:            SyncTypeRedirect,
	}
}
