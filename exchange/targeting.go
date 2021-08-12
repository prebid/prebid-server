package exchange

import (
	"strconv"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const MaxKeyLength = 20

// targetData tracks information about the winning Bid in each Imp.
//
// All functions on this struct are nil-safe. If the targetData struct is nil, then they behave
// like they would if no targeting information is needed.
//
// All functions on this struct are all nil-safe.
// If the value is nil, then no targeting data will be tracked.
type targetData struct {
	priceGranularity  openrtb_ext.PriceGranularity
	includeWinners    bool
	includeBidderKeys bool
	includeCacheBids  bool
	includeCacheVast  bool
	includeFormat     bool
	preferDeals       bool
	// cacheHost and cachePath exist to supply cache host and path as targeting parameters
	cacheHost string
	cachePath string
}

// setTargeting writes all the targeting params into the bids.
// If any errors occur when setting the targeting params for a particular bid, then that bid will be ejected from the auction.
//
// The one exception is the `hb_cache_id` key. Since our APIs explicitly document cache keys to be on a "best effort" basis,
// it's ok if those stay in the auction. For now, this method implements a very naive cache strategy.
// In the future, we should implement a more clever retry & backoff strategy to balance the success rate & performance.
func (targData *targetData) setTargeting(auc *auction, isApp bool, categoryMapping map[string]string) {
	for impId, topBidsPerImp := range auc.winningBidsByBidder {
		overallWinner := auc.winningBids[impId]
		for bidderName, topBidPerBidder := range topBidsPerImp {
			isOverallWinner := overallWinner == topBidPerBidder

			targets := make(map[string]string, 10)
			if cpm, ok := auc.roundedPrices[topBidPerBidder]; ok {
				targData.addKeys(targets, openrtb_ext.HbpbConstantKey, cpm, bidderName, isOverallWinner)
			}
			targData.addKeys(targets, openrtb_ext.HbBidderConstantKey, string(bidderName), bidderName, isOverallWinner)
			if hbSize := makeHbSize(topBidPerBidder.bid); hbSize != "" {
				targData.addKeys(targets, openrtb_ext.HbSizeConstantKey, hbSize, bidderName, isOverallWinner)
			}
			if cacheID, ok := auc.cacheIds[topBidPerBidder.bid]; ok {
				targData.addKeys(targets, openrtb_ext.HbCacheKey, cacheID, bidderName, isOverallWinner)
			}
			if vastID, ok := auc.vastCacheIds[topBidPerBidder.bid]; ok {
				targData.addKeys(targets, openrtb_ext.HbVastCacheKey, vastID, bidderName, isOverallWinner)
			}
			if targData.includeFormat {
				targData.addKeys(targets, openrtb_ext.HbFormatKey, string(topBidPerBidder.bidType), bidderName, isOverallWinner)
			}

			if targData.cacheHost != "" {
				targData.addKeys(targets, openrtb_ext.HbConstantCacheHostKey, targData.cacheHost, bidderName, isOverallWinner)
			}
			if targData.cachePath != "" {
				targData.addKeys(targets, openrtb_ext.HbConstantCachePathKey, targData.cachePath, bidderName, isOverallWinner)
			}

			if deal := topBidPerBidder.bid.DealID; len(deal) > 0 {
				targData.addKeys(targets, openrtb_ext.HbDealIDConstantKey, deal, bidderName, isOverallWinner)
			}

			if isApp {
				targData.addKeys(targets, openrtb_ext.HbEnvKey, openrtb_ext.HbEnvKeyApp, bidderName, isOverallWinner)
			}
			if len(categoryMapping) > 0 {
				targData.addKeys(targets, openrtb_ext.HbCategoryDurationKey, categoryMapping[topBidPerBidder.bid.ID], bidderName, isOverallWinner)
			}

			topBidPerBidder.bidTargets = targets
		}
	}
}

func (targData *targetData) addKeys(keys map[string]string, key openrtb_ext.TargetingKey, value string, bidderName openrtb_ext.BidderName, overallWinner bool) {
	if targData.includeBidderKeys {
		keys[key.BidderKey(bidderName, MaxKeyLength)] = value
	}
	if targData.includeWinners && overallWinner {
		keys[string(key)] = value
	}
}

func makeHbSize(bid *openrtb2.Bid) string {
	if bid.W != 0 && bid.H != 0 {
		return strconv.FormatInt(bid.W, 10) + "x" + strconv.FormatInt(bid.H, 10)
	}
	return ""
}
