package usersyncers

import (
	"fmt"
	"net/url"
)

func NewAdformSyncer(usersyncURL string, externalURL string) *syncer {
	redirectUri := fmt.Sprintf("%s/setuid?bidder=adform&uid=$UID", externalURL)

	return &syncer{
		familyName:          "adform",
		gdprVendorID:        50,
		syncEndpointBuilder: constEndpoint(fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectUri))),
		syncType:            SyncTypeRedirect,
	}
}
