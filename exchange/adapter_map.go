package exchange

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"net/http"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// The newAdapterMap function is segregated to its own file to make it a simple and clean location for each Adapter
// to register itself. No wading through Exchange code to find it.

func newAdapterMap(client *http.Client) map[openrtb_ext.BidderName]adapters.Bidder {
	return map[openrtb_ext.BidderName]adapters.Bidder{
		openrtb_ext.BidderAppnexus: adapters.AdaptHttpBidder(new(appnexus.AppNexusAdapter), client),
	}
}
