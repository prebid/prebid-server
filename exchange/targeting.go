package exchange

import (
	"strconv"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const maxKeyLength = 20

// targetData tracks information about the winning Bid in each Imp.
//
// All functions on this struct are nil-safe. If the targetData struct is nil, then they behave
// like they would if no targeting information is needed.
//
// All functions on this struct are all nil-safe.
// If the value is nil, then no targeting data will be tracked.
type targetData struct {
	priceGranularity openrtb_ext.PriceGranularity
	includeCache     bool
}

// setTargeting writes all the targeting params into the bids.
// If any errors occur when setting the targeting params for a particular bid, then that bid will be ejected from the auction.
//
// The one exception is the `hb_cache_id` key. Since our APIs explicitly document cache keys to be on a "best effort" basis,
// it's ok if those stay in the auction. For now, this method implements a very naive cache strategy.
// In the future, we should implement a more clever retry & backoff strategy to balance the success rate & performance.
func setTargeting(auc *auction, isApp bool) {
	for impId, topBidsPerImp := range auc.winningBidsByBidder {
		overallWinner := auc.winningBids[impId]
		for bidderName, topBidPerBidder := range topBidsPerImp {
			isOverallWinner := overallWinner == topBidPerBidder

			targets := make(map[string]string, 10)
			if cpm, ok := auc.roundedPrices[topBidPerBidder]; ok {
				addKeys(targets, openrtb_ext.HbpbConstantKey, cpm, bidderName, isOverallWinner)
			}
			addKeys(targets, openrtb_ext.HbBidderConstantKey, string(bidderName), bidderName, isOverallWinner)
			if hbSize := makeHbSize(topBidPerBidder.bid); hbSize != "" {
				addKeys(targets, openrtb_ext.HbSizeConstantKey, hbSize, bidderName, isOverallWinner)
			}
			if cacheId, ok := auc.cacheIds[topBidPerBidder.bid]; ok {
				addKeys(targets, openrtb_ext.HbCacheKey, cacheId, bidderName, isOverallWinner)
			}
			if deal := topBidPerBidder.bid.DealID; len(deal) > 0 {
				addKeys(targets, openrtb_ext.HbDealIdConstantKey, deal, bidderName, isOverallWinner)
			}

			if bidderName == "audienceNetwork" {
				targets[string(openrtb_ext.HbCreativeLoadMethodConstantKey)] = openrtb_ext.HbCreativeLoadMethodDemandSDK
			} else {
				targets[string(openrtb_ext.HbCreativeLoadMethodConstantKey)] = openrtb_ext.HbCreativeLoadMethodHTML
			}

			if isApp {
				addKeys(targets, openrtb_ext.HbEnvKey, openrtb_ext.HbEnvKeyApp, bidderName, isOverallWinner)
			}

			topBidPerBidder.bidTargets = targets
		}
	}
}

func addKeys(keys map[string]string, key openrtb_ext.TargetingKey, value string, bidderName openrtb_ext.BidderName, overallWinner bool) {
	keys[key.BidderKey(bidderName, maxKeyLength)] = value
	if overallWinner {
		keys[string(key)] = value
	}
}

func makeHbSize(bid *openrtb.Bid) string {
	if bid.W != 0 && bid.H != 0 {
		return strconv.FormatUint(bid.W, 10) + "x" + strconv.FormatUint(bid.H, 10)
	}
	return ""
}
