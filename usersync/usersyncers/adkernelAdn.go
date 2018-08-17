package usersyncers

import (
	"net/url"
	"strings"
)

const adkernelGDPRVendorID = 14

func NewAdkernelAdnSyncer(pbServerSyncURL string, adkernelUserSyncURL string) *syncer {
	pbServerSyncURL = strings.TrimRight(pbServerSyncURL, "/") + "/setuid?bidder=adkernelAdn&uid={UID}"
	return &syncer{
		familyName:          "adkernelAdn",
		gdprVendorID:        adkernelGDPRVendorID,
		syncEndpointBuilder: resolveMacros(adkernelUserSyncURL + url.QueryEscape(pbServerSyncURL)),
		syncType:            SyncTypeRedirect,
	}
}
