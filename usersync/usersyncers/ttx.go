package usersyncers

import (
	"net/url"
	"strings"
)

func NewTtxSyncer(externalURL string, userSyncUrl string, partnerId string) *syncer {
	externalURL = strings.TrimRight(externalURL, "/")
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dttx%26uid%3D33XUSERID33X"

	return &syncer{
		familyName:          "ttx",
		syncEndpointBuilder: resolveMacros(userSyncUrl + "/?ri=" + partnerId + "&ru=" + redirectURL),
		syncType:            SyncTypeRedirect,
	}
}
