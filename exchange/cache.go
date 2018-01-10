package exchange

import (
	"context"
	"encoding/json"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/prebid_cache_client"
)

// cacheBids stores the given Bids in prebid cache, and saves the generated UUIDs in the auction.
// If any cache calls fail, then there's not much anyone can do about it. This function will just log
// the error and save IDs to any bids which are cached successfully.
func cacheBids(ctx context.Context, cache prebid_cache_client.Client, auction *auction) {
	numCachedBids := 0
	auction.forEachBestBid(func(impID string, bidder openrtb_ext.BidderName, bid *openrtb.Bid, winner bool) {
		numCachedBids++
	})
	if numCachedBids < 1 {
		return
	}

	bids := make([]*openrtb.Bid, numCachedBids)
	nextBidIndex := 0
	auction.forEachBestBid(func(impID string, bidder openrtb_ext.BidderName, bid *openrtb.Bid, winner bool) {
		bids[nextBidIndex] = bid
		nextBidIndex++
	})

	// Marshal the bids into JSON payloads. If any errors occur during marshalling, eject that bid from the array.
	// After this block, we expect "bids" and "jsonValues" to have the same number of elements in the same order.
	jsonValues := make([]json.RawMessage, 0, numCachedBids)
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
	auction.cachedBids = make(map[*openrtb.Bid]string, numCachedBids)
	for i := 0; i < numCachedBids; i++ {
		if ids[i] != "" {
			auction.cachedBids[bids[i]] = ids[i]
		}
	}
}
