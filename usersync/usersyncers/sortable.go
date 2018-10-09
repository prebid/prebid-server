package usersyncers

import (
	"net/url"
)

func NewSortableSyncer(externalURL string, usersyncURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dsortable%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24UID"
	// TODO: Everything
	return &syncer{
		familyName:          "sortable",
		gdprVendorID:        145,
		syncEndpointBuilder: resolveMacros(usersyncURL + "redir=" + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
