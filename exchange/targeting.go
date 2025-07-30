package exchange

import (
	"fmt"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

const MaxKeyLength = 20
const MinKeyLength = 12 // longest attribute length without prefix
const DefaultKeyPrefix = "hb"

// targetData tracks information about the winning Bid in each Imp.
//
// All functions on this struct are nil-safe. If the targetData struct is nil, then they behave
// like they would if no targeting information is needed.
//
// All functions on this struct are all nil-safe.
// If the value is nil, then no targeting data will be tracked.
type targetData struct {
	priceGranularity          openrtb_ext.PriceGranularity
	mediaTypePriceGranularity openrtb_ext.MediaTypePriceGranularity
	includeWinners            bool
	includeBidderKeys         bool
	includeCacheBids          bool
	includeCacheVast          bool
	includeFormat             bool
	preferDeals               bool
	alwaysIncludeDeals        bool
	// cacheHost and cachePath exist to supply cache host and path as targeting parameters
	cacheHost string
	cachePath string
	prefix    string
}

// setTargeting writes all the targeting params into the bids.
// If any errors occur when setting the targeting params for a particular bid, then that bid will be ejected from the auction.
//
// The one exception is the `hb_cache_id` key. Since our APIs explicitly document cache keys to be on a "best effort" basis,
// it's ok if those stay in the auction. For now, this method implements a very naive cache strategy.
// In the future, we should implement a more clever retry & backoff strategy to balance the success rate & performance.
func (targData *targetData) setTargeting(auc *auction, isApp bool, categoryMapping map[string]string, truncateTargetAttr *int, multiBidMap map[string]openrtb_ext.ExtMultiBid) {
	for impId, topBidsPerImp := range auc.allBidsByBidder {
		overallWinner := auc.winningBids[impId]
		for originalBidderName, topBidsPerBidder := range topBidsPerImp {
			targetingBidderCode := originalBidderName
			bidderCodePrefix, maxBids := getMultiBidMeta(multiBidMap, originalBidderName.String())

			for i, topBid := range topBidsPerBidder {
				// Limit targeting keys to maxBids (default 1 bid).
				// And, do not apply targeting for more than 1 bid if bidderCodePrefix is not defined.
				if i == maxBids || (i == 1 && bidderCodePrefix == "") {
					break
				}

				if i > 0 { // bidderCode is used for first bid, generated bidderCodePrefix for following bids
					targetingBidderCode = openrtb_ext.BidderName(fmt.Sprintf("%s%d", bidderCodePrefix, i+1))
				}

				if maxBids > openrtb_ext.DefaultBidLimit { // add targetingbiddercode only if multibid is set for this bidder
					topBid.TargetBidderCode = targetingBidderCode.String()
				}

				isOverallWinner := overallWinner == topBid

				bidHasDeal := len(topBid.Bid.DealID) > 0

				targets := make(map[string]string, 10)
				if cpm, ok := auc.roundedPrices[topBid]; ok {
					targData.addKeys(targets, openrtb_ext.PbKey, cpm, targetingBidderCode, isOverallWinner, truncateTargetAttr, bidHasDeal)
				}
				targData.addKeys(targets, openrtb_ext.BidderKey, string(targetingBidderCode), targetingBidderCode, isOverallWinner, truncateTargetAttr, bidHasDeal)
				if hbSize := makeHbSize(topBid.Bid); hbSize != "" {
					targData.addKeys(targets, openrtb_ext.SizeKey, hbSize, targetingBidderCode, isOverallWinner, truncateTargetAttr, bidHasDeal)
				}
				if cacheID, ok := auc.cacheIds[topBid.Bid]; ok {
					targData.addKeys(targets, openrtb_ext.CacheKey, cacheID, targetingBidderCode, isOverallWinner, truncateTargetAttr, bidHasDeal)
				}
				if vastID, ok := auc.vastCacheIds[topBid.Bid]; ok {
					targData.addKeys(targets, openrtb_ext.VastCacheKey, vastID, targetingBidderCode, isOverallWinner, truncateTargetAttr, bidHasDeal)
				}
				if targData.includeFormat {
					targData.addKeys(targets, openrtb_ext.FormatKey, string(topBid.BidType), targetingBidderCode, isOverallWinner, truncateTargetAttr, bidHasDeal)
				}

				if targData.cacheHost != "" {
					targData.addKeys(targets, openrtb_ext.CacheHostKey, targData.cacheHost, targetingBidderCode, isOverallWinner, truncateTargetAttr, bidHasDeal)
				}
				if targData.cachePath != "" {
					targData.addKeys(targets, openrtb_ext.CachePathKey, targData.cachePath, targetingBidderCode, isOverallWinner, truncateTargetAttr, bidHasDeal)
				}

				if bidHasDeal {
					targData.addKeys(targets, openrtb_ext.DealKey, topBid.Bid.DealID, targetingBidderCode, isOverallWinner, truncateTargetAttr, bidHasDeal)
				}

				if isApp {
					targData.addKeys(targets, openrtb_ext.EnvKey, openrtb_ext.EnvAppValue, targetingBidderCode, isOverallWinner, truncateTargetAttr, bidHasDeal)
				}
				if len(categoryMapping) > 0 {
					targData.addKeys(targets, openrtb_ext.CategoryDurationKey, categoryMapping[topBid.Bid.ID], targetingBidderCode, isOverallWinner, truncateTargetAttr, bidHasDeal)
				}
				topBid.BidTargets = targets
			}
		}
	}
}

func (targData *targetData) addKeys(keys map[string]string, key openrtb_ext.TargetingKey, value string, bidderName openrtb_ext.BidderName, overallWinner bool, truncateTargetAttr *int, bidHasDeal bool) {
	maxLength := MaxKeyLength
	if truncateTargetAttr != nil && *truncateTargetAttr > 0 {
		maxLength = *truncateTargetAttr
	}
	if targData.includeBidderKeys || (targData.alwaysIncludeDeals && bidHasDeal) {
		keys[key.BidderKey(targData.prefix, bidderName, maxLength)] = value
	}
	if targData.includeWinners && overallWinner {
		keys[key.TruncateKey(targData.prefix, maxLength)] = value
	}
}

func makeHbSize(bid *openrtb2.Bid) string {
	if bid.W != 0 && bid.H != 0 {
		return strconv.FormatInt(bid.W, 10) + "x" + strconv.FormatInt(bid.H, 10)
	}
	return ""
}

func getMultiBidMeta(multiBidMap map[string]openrtb_ext.ExtMultiBid, bidder string) (string, int) {
	if multiBidMap != nil {
		if multiBid, ok := multiBidMap[bidder]; ok {
			return multiBid.TargetBidderCodePrefix, *multiBid.MaxBids
		}
	}

	return "", openrtb_ext.DefaultBidLimit
}
