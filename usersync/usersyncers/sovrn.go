package usersyncers

import (
	"net/url"
)

func NewSovrnSyncer(externalURL string, usersyncURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dsovrn%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24UID"

	return &syncer{
		familyName:          "sovrn",
		gdprVendorID:        13,
		syncEndpointBuilder: resolveMacros(usersyncURL + "redir=" + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
