package exchange

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"net/http"
)

// The newAdapterMap function is segregated to its own file to make it a simple and clean location for each Adapter
// to register itself. No wading through Exchange code to find it.

func newAdapterMap(client *http.Client) map[string]adapters.Bidder {
	adapterMap := map[string]adapters.Bidder{
		"appnexus": adapters.AdaptHttpBidder(new(appnexus.AppNexusAdapter), client),
	}

	return adapterMap
}


