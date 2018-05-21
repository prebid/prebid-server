package usersyncers

import (
	"net/url"
)

func NewEPlanningSyncer(usersyncURL string, externalURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Deplanning%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24UID"

	return &syncer{
		familyName:          "eplanning",
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
