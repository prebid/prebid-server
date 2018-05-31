package usersyncers

import (
	"net/url"
	"strings"
)

func NewSomoaudienceSyncer(externalURL string) *syncer {
	externalURL = strings.TrimRight(externalURL, "/")
<<<<<<< HEAD
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dsomoaudience%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24UID"

	usersyncURL := "//publisher-east.mobileadtrading.com/usersync?gdprg=1&ru="
=======
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dmobileadtrading%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24%7BUID%7D"
>>>>>>> 8615830... Further Tweaks

	return &syncer{
		familyName:          "somoaudience",
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURL),
		syncType:            SyncTypeRedirect,
	}
}
