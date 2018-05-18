package usersyncers

import (
	"fmt"
	"net/url"
)

func NewConversantSyncer(usersyncURL string, externalURL string) *syncer {
	redirectURI := fmt.Sprintf("%s/setuid?bidder=conversant&uid=", externalURL)

	return &syncer{
		familyName:          "conversant",
		gdprVendorID:        24,
		syncEndpointBuilder: constEndpoint(fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectURI))),
		syncType:            SyncTypeRedirect,
	}
}
