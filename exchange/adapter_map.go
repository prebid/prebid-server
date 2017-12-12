package exchange

import (
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/rubicon"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

// The newAdapterMap function is segregated to its own file to make it a simple and clean location for each Adapter
// to register itself. No wading through Exchange code to find it.

func newAdapterMap(client *http.Client, cfg *config.Configuration) map[openrtb_ext.BidderName]adaptedBidder {
	return map[openrtb_ext.BidderName]adaptedBidder{
		openrtb_ext.BidderAppnexus: adaptBidder(appnexus.NewAppNexusBidder(client, cfg.ExternalURL), client),
		openrtb_ext.BidderRubicon: adaptBidder(rubicon.NewRubiconBidder(client, cfg.Adapters["rubicon"].Endpoint, cfg.Adapters["rubicon"].XAPI.Username,
			cfg.Adapters["rubicon"].XAPI.Password, cfg.Adapters["rubicon"].XAPI.Tracker, cfg.Adapters["rubicon"].UserSyncURL), client),
	}
}
