package usersyncers

import (
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/usersync"
)

func NewAppnexusSyncer(externalURL string) usersync.Usersyncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=adnxs&uid=$UID", externalURL)
	usersyncURL := "//ib.adnxs.com/getuid?"

	return &syncer{
		familyName:          "adnxs",
		gdprVendorID:        32,
		syncEndpointBuilder: constEndpoint(fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri))),
		syncType:            SyncTypeRedirect,
	}
}
