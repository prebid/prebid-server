package exchange

import (
	"github.com/prebid/prebid-server/adapters/appnexus"
	"net/http"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/config"
)

// The newAdapterMap function is segregated to its own file to make it a simple and clean location for each Adapter
// to register itself. No wading through Exchange code to find it.

func newAdapterMap(client *http.Client, cfg *config.Configuration) map[openrtb_ext.BidderName]adaptedBidder {
	return map[openrtb_ext.BidderName]adaptedBidder{
		openrtb_ext.BidderAppnexus: adaptBidder(appnexus.NewAppNexusBidder(client, cfg.ExternalURL), client),
	}
}
