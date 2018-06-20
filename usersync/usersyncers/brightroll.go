package usersyncers

import (
	"net/url"
	"strings"
)

func NewBrightrollSyncer(userSyncURL string, externalURL string) *syncer {
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dbrightroll%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24%7BUID%7D"
	return &syncer{
		familyName:          "brightroll",
		gdprVendorID:        25, //oath vendor Id
		syncEndpointBuilder: resolveMacros(userSyncURL + redirectURL),
		syncType:            SyncTypeRedirect,
	}
}
