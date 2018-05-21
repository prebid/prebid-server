package usersyncers

import (
	"net/url"
)

func NewAdtelligentSyncer(externalURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%7Buid%7D"
	usersyncURL := "//sync.adtelligent.com/csync?t=p&ep=0&redir="

	return &syncer{
		familyName:          "adtelligent",
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
