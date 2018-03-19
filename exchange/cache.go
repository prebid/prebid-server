package exchange

import (
	"context"
	"encoding/json"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/prebid_cache_client"
)

func cacheBids(ctx context.Context, cache prebid_cache_client.Client, bids []*openrtb.Bid) map[*openrtb.Bid]string {
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
	toReturn := make(map[*openrtb.Bid]string, len(bids))
	for i := 0; i < len(bids); i++ {
		if ids[i] != "" {
			toReturn[bids[i]] = ids[i]
		}
	}
	return toReturn
}
