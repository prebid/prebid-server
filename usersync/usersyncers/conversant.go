package usersyncers

import (
	"net/url"
)

func NewConversantSyncer(usersyncURL string, externalURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dconversant%26uid%3D"

	return &syncer{
		familyName:          "conversant",
		gdprVendorID:        24,
		syncEndpointBuilder: constEndpoint(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
