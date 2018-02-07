package exchange

import (
	"context"
	"encoding/json"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs/buckets"
	"github.com/prebid/prebid-server/prebid_cache_client"
	"strings"
)

// cacheBids mutates the auction so that the highest Bid from each Bidder in each Imp has a Cache ID associated with it.
//
// If any cache calls fail, then there's not much anyone can do about it. This function will just log
// the error and save IDs to any bids which are cached successfully.
func cacheBids(ctx context.Context, cache prebid_cache_client.Client, auction *auction, granularity openrtb_ext.PriceGranularity) {
	bids := make([]*openrtb.Bid, 0, 30) // Arbitrary initial capacity
	nextBidIndex := 0
	auction.forEachBestBid(func(impID string, bidder openrtb_ext.BidderName, bid *openrtb.Bid, winner bool) {
		// Fixes #199
		granularityStr, err := buckets.GetPriceBucketString(bid.Price, granularity)
		if err == nil && strings.ContainsAny(granularityStr, "123456789") {
			bids = append(bids, bid)
			nextBidIndex++
		}
	})

	// Marshal the bids into JSON payloads. If any errors occur during marshalling, eject that bid from the array.
	// After this block, we expect "bids" and "jsonValues" to have the same number of elements in the same order.
	jsonValues := make([]json.RawMessage, 0, len(bids))
	for i := 0; i < len(bids); i++ {
		if jsonBytes, err := json.Marshal(bids[i]); err != nil {
			glog.Errorf("Error marshalling OpenRTB Bid for Prebid Cache: %v", err)
			bids = append(bids[:i], bids[i+1:]...)
			i--
		} else {
			jsonValues = append(jsonValues, jsonBytes)
		}
	}

	ids := cache.PutJson(ctx, jsonValues)
	auction.cachedBids = make(map[*openrtb.Bid]string, len(bids))
	for i := 0; i < len(bids); i++ {
		if ids[i] != "" {
			auction.cachedBids[bids[i]] = ids[i]
		}
	}
}
