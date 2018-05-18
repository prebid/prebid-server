package usersyncers

import (
	"net/url"
)

func NewAdtelligentSyncer(externalURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dadtelligent%26uid%3D%7Buid%7D"
	usersyncURL := "//sync.adtelligent.com/csync?t=p&ep=0&redir="

	return &syncer{
		familyName:          "adtelligent",
		syncEndpointBuilder: constEndpoint(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
