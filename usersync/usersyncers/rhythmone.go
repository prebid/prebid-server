package usersyncers

import (
	"net/url"
)

func NewRhythmoneSyncer(usersyncURL string, externalURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D%7B%7Bgdpr%7D%7D%26gdpr_consent%3D%7B%7Bgdpr_consent%7D%7D%26uid%3D%7BRX_UUID%7D"

	return &syncer{
		familyName:          "rhythmone",
		gdprVendorID:        36,
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
