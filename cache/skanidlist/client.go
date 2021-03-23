package skanidlist

import (
	"context"
	"net/http"

	"github.com/prebid/prebid-server/openrtb_ext"
)

type client map[openrtb_ext.BidderName]*cache

// Empty skanIDListClient
var skanIDListClient client = client{}

func cacheClient(bidder openrtb_ext.BidderName) *cache {
	if _, ok := skanIDListClient[bidder]; !ok {
		// Initialize bidder caches at first call
		switch bidder {
		case openrtb_ext.BidderTaurusX:
			skanIDListClient[bidder] = newCache("https://www.taurusx.com/skadnetworkids.json", bidder)
		}
	}

	return skanIDListClient[bidder]
}

func Update(ctx context.Context, httpClient *http.Client, bidder openrtb_ext.BidderName) {
	if c := cacheClient(bidder); c != nil {
		c.update(ctx, httpClient)
	}
}

func Get(bidder openrtb_ext.BidderName) map[string]bool {
	if c := cacheClient(bidder); c != nil {
		return c.get()
	}
	return map[string]bool{}
}
