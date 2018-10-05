package usersyncers

import (
	"net/url"
	"strings"
)

func NewRhythmoneSyncer(usersyncURL string, externalURL string) *syncer {
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%5BRX_UUID%5D"

	return &syncer{
		familyName:          "rhythmone",
		gdprVendorID:        36,
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
