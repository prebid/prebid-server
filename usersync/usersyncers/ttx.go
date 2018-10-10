package usersyncers

import (
	"net/url"
	"strings"
)

func NewTtxSyncer(externalURL string) *syncer {
	//TODO: need to update
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dttx%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24%7BUID%7D"

	return &syncer{
		familyName:          "ttx",
		gdprVendorID:        999,
		syncEndpointBuilder: resolveMacros("To_BE_UPDATE" + redirectURL),
		syncType:            SyncTypeRedirect,
	}
}
