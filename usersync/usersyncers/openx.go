package usersyncers

import (
	"net/url"
	"strings"
)

func NewOpenxSyncer(externalURL string) *syncer {
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dopenx%26uid%3D%24%7BUID%7D"

	return &syncer{
		familyName:          "openx",
		gdprVendorID:        69,
		syncEndpointBuilder: constEndpoint("https://rtb.openx.net/sync/prebid?r=" + redirectURL),
		syncType:            SyncTypeRedirect,
	}
}
