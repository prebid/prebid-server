package usersyncers

import (
	"net/url"
)

func NewAdformSyncer(usersyncURL string, externalURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dadform%26uid%3D%24UID"

	return &syncer{
		familyName:          "adform",
		gdprVendorID:        50,
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
