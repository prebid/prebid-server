package adkernelAdn

import (
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

const adkernelGDPRVendorID = uint16(14)

func NewAdkernelAdnSyncer(cfg *config.Configuration) usersync.Usersyncer {
	// Fixes #736
	usersyncURL := cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdkernelAdn))].UserSyncURL
	externalURL := strings.TrimRight(cfg.ExternalURL, "/") + "/setuid?bidder=adkernelAdn&uid={UID}"
	return adapters.NewSyncer("adkernelAdn", adkernelGDPRVendorID, adapters.ResolveMacros(usersyncURL+url.QueryEscape(externalURL)), adapters.SyncTypeRedirect)
}
