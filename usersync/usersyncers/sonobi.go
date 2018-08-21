package usersyncers

import (
	"github.com/prebid/prebid-server/usersync"
	"net/url"
)

func NewSonobiSyncer(externalURL string) usersync.Usersyncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dsonobi%26consent_string%3D{{gdpr}}%26gdpr%3D{{gdpr_consent}}%26uid%3D%24UID"
	usersyncURL := "//sync.go.sonobi.com/us.gif?loc="

	return &syncer{
		familyName:          "sonobi",
		gdprVendorID:        104,
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
