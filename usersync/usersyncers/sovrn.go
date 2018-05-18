package usersyncers

import (
	"fmt"
	"net/url"
)

func NewSovrnSyncer(externalURL string, usersyncURL string) *syncer {
	redirectURI := fmt.Sprintf("%s/setuid?bidder=sovrn&uid=$UID", externalURL)

	return &syncer{
		familyName:          "sovrn",
		gdprVendorID:        13,
		syncEndpointBuilder: constEndpoint(fmt.Sprintf("%sredir=%s", usersyncURL, url.QueryEscape(redirectURI))),
		syncType:            SyncTypeRedirect,
	}
}
