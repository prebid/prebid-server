package usersyncers

import (
	"net/url"
	"strings"
)

func NewSomoaudienceSyncer(externalURL string) *syncer {
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dmobileadtrading%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24%7BUID%7D"

	usersyncURL := "//publisher-east.mobileadtrading.com/usersync?ru="

	return &syncer{
		familyName:          "mobileadtrading",
		gdprVendorID:        341,
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURL),
		syncType:            SyncTypeRedirect,
	}
}
