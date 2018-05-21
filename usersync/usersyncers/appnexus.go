package usersyncers

import (
	"net/url"

	"github.com/prebid/prebid-server/usersync"
)

func NewAppnexusSyncer(externalURL string) usersync.Usersyncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dadnxs%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24UID"
	usersyncURL := "//ib.adnxs.com/getuid?"

	return &syncer{
		familyName:          "adnxs",
		gdprVendorID:        32,
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
