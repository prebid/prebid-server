package usersyncers

import (
	"net/url"
)

func NewEPlanningSyncer(usersyncURL string, externalURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Deplanning%26uid%3D%24UID"

	return &syncer{
		familyName:          "eplanning",
		syncEndpointBuilder: constEndpoint(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
