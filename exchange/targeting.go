package exchange

import (
	"fmt"
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
	includeBidderKeys bool // need to be true
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
func (targData *targetData) setTargeting(auc *auction, isApp bool, categoryMapping map[string]string, truncateTargetAttr *int, adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, multiBidMap ExtMultiBidMap) {
	for originalBidderName, seatBid := range adapterBids {
		impIdToBidMap := getBidsByImpId(seatBid.bids)

		var bidderCodePrefix string
		maxBids := 1
		multiBid, ok := multiBidMap[originalBidderName.String()]
		if ok {
			bidderCodePrefix = multiBid.TargetBidderCodePrefix
			maxBids = *multiBid.MaxBids
		}

		for impId, impBids := range impIdToBidMap {
			overallWinner := auc.winningBids[impId]
			// topBid := auc.winningBidsByBidder[impId][originalBidderName]

			targetingBidderCode := originalBidderName
			for i, bid := range impBids {
				isOverallWinner := overallWinner == bid
				if _, ok := auc.allWinningBidsByBidder[bid]; !ok {
					// not a winning bid for bidder
					continue
				}

				if i > 0 {
					if bidderCodePrefix == "" {
						// no targeting for bids w/o bidderCodePrefix
						continue
					}
					targetingBidderCode = openrtb_ext.BidderName(fmt.Sprintf("%s%d", bidderCodePrefix, i+1))
				}
				bidderName := targetingBidderCode

				targets := make(map[string]string, 10)
				if cpm, ok := auc.roundedPrices[bid]; ok {
					targData.addKeys(targets, openrtb_ext.HbpbConstantKey, cpm, bidderName, isOverallWinner, truncateTargetAttr)
				}
				targData.addKeys(targets, openrtb_ext.HbBidderConstantKey, string(bidderName), bidderName, isOverallWinner, truncateTargetAttr)
				if hbSize := makeHbSize(bid.bid); hbSize != "" {
					targData.addKeys(targets, openrtb_ext.HbSizeConstantKey, hbSize, bidderName, isOverallWinner, truncateTargetAttr)
				}
				if cacheID, ok := auc.cacheIds[bid.bid]; ok {
					targData.addKeys(targets, openrtb_ext.HbCacheKey, cacheID, bidderName, isOverallWinner, truncateTargetAttr)
				}
				if vastID, ok := auc.vastCacheIds[bid.bid]; ok {
					targData.addKeys(targets, openrtb_ext.HbVastCacheKey, vastID, bidderName, isOverallWinner, truncateTargetAttr)
				}
				if targData.includeFormat {
					targData.addKeys(targets, openrtb_ext.HbFormatKey, string(bid.bidType), bidderName, isOverallWinner, truncateTargetAttr)
				}

				if targData.cacheHost != "" {
					targData.addKeys(targets, openrtb_ext.HbConstantCacheHostKey, targData.cacheHost, bidderName, isOverallWinner, truncateTargetAttr)
				}
				if targData.cachePath != "" {
					targData.addKeys(targets, openrtb_ext.HbConstantCachePathKey, targData.cachePath, bidderName, isOverallWinner, truncateTargetAttr)
				}

				if deal := bid.bid.DealID; len(deal) > 0 {
					targData.addKeys(targets, openrtb_ext.HbDealIDConstantKey, deal, bidderName, isOverallWinner, truncateTargetAttr)
				}

				if isApp {
					targData.addKeys(targets, openrtb_ext.HbEnvKey, openrtb_ext.HbEnvKeyApp, bidderName, isOverallWinner, truncateTargetAttr)
				}
				if len(categoryMapping) > 0 {
					targData.addKeys(targets, openrtb_ext.HbCategoryDurationKey, categoryMapping[bid.bid.ID], bidderName, isOverallWinner, truncateTargetAttr)
				}

				bid.bidTargets = targets
				if maxBids > 1 {
					// update targeting key only if multibid is set for bidder
					bid.targetBidderCode = bidderName.String()
				}
			}
		}
	}
}

func (targData *targetData) addKeys(keys map[string]string, key openrtb_ext.TargetingKey, value string, bidderName openrtb_ext.BidderName, overallWinner bool, truncateTargetAttr *int) {
	var maxLength int
	if truncateTargetAttr != nil {
		maxLength = *truncateTargetAttr
		if maxLength < 0 {
			maxLength = MaxKeyLength
		}
	} else {
		maxLength = MaxKeyLength
	}
	if targData.includeBidderKeys {
		keys[key.BidderKey(bidderName, maxLength)] = value
	}
	if targData.includeWinners && overallWinner {
		keys[key.TruncateKey(maxLength)] = value
	}
}

func makeHbSize(bid *openrtb2.Bid) string {
	if bid.W != 0 && bid.H != 0 {
		return strconv.FormatInt(bid.W, 10) + "x" + strconv.FormatInt(bid.H, 10)
	}
	return ""
}
