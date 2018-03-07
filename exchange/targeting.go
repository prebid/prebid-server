package exchange

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs/buckets"
	"github.com/prebid/prebid-server/prebid_cache_client"
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

// makePrebidTargets returns the _bidder specific_ targeting keys and values. For example,
// this map will include "hb_pb_appnexus", but _not_ "hb_pb".
func (t *targetData) makePrebidTargets(name openrtb_ext.BidderName, bid *openrtb.Bid) (map[string]string, error) {
	if t == nil {
		return nil, nil
	}

	cpm := bid.Price
	width := bid.W
	height := bid.H
	deal := bid.DealID

	roundedCpm, err := buckets.GetPriceBucketString(cpm, t.priceGranularity)
	if err != nil {
		// set broken cpm to 0
		roundedCpm = "0.0"
	}

	hbSize := ""
	if width != 0 && height != 0 {
		w := strconv.FormatUint(width, 10)
		h := strconv.FormatUint(height, 10)
		hbSize = w + "x" + h
	}

	hbPbBidderKey := openrtb_ext.HbpbConstantKey.BidderKey(name, maxKeyLength)
	hbBidderBidderKey := openrtb_ext.HbBidderConstantKey.BidderKey(name, maxKeyLength)
	hbSizeBidderKey := openrtb_ext.HbSizeConstantKey.BidderKey(name, maxKeyLength)
	hbDealIdBidderKey := openrtb_ext.HbDealIdConstantKey.BidderKey(name, maxKeyLength)

	pbs_kvs := map[string]string{
		hbPbBidderKey:     roundedCpm,
		hbBidderBidderKey: string(name),
	}

	if hbSize != "" {
		pbs_kvs[hbSizeBidderKey] = hbSize
	}
	if len(deal) > 0 {
		pbs_kvs[hbDealIdBidderKey] = deal
	}
	return pbs_kvs, err
}

func (t *targetData) shouldCache() bool {
	return t != nil && t.includeCache
}

// addTargetsToCompletedAuction takes a _completed_ auction, and adds all the appropriate targeting keys to it.
// Once this has been called, auction.addBid() should _not_ be called anymore.
func (t *targetData) addTargetsToCompletedAuction(auction *auction) {
	if t == nil {
		return
	}

	auction.forEachBestBid(func(id string, bidderName openrtb_ext.BidderName, bid *openrtb.Bid, overallWinner bool) {
		bidExt := new(openrtb_ext.ExtBid)
		err1 := json.Unmarshal(bid.Ext, bidExt)
		if err1 == nil && overallWinner && bidExt.Prebid.Targeting != nil {
			cacheId, hasCacheId := auction.cacheId(bid)
			if overallWinner {
				hbPbBidderKey := openrtb_ext.HbpbConstantKey.BidderKey(bidderName, maxKeyLength)
				hbBidderBidderKey := openrtb_ext.HbBidderConstantKey.BidderKey(bidderName, maxKeyLength)
				hbSizeBidderKey := openrtb_ext.HbSizeConstantKey.BidderKey(bidderName, maxKeyLength)
				hbDealIdBidderKey := openrtb_ext.HbDealIdConstantKey.BidderKey(bidderName, maxKeyLength)

				bidExt.Prebid.Targeting[string(openrtb_ext.HbpbConstantKey)] = bidExt.Prebid.Targeting[hbPbBidderKey]
				bidExt.Prebid.Targeting[string(openrtb_ext.HbBidderConstantKey)] = bidExt.Prebid.Targeting[hbBidderBidderKey]
				if size, ok := bidExt.Prebid.Targeting[hbSizeBidderKey]; ok {
					bidExt.Prebid.Targeting[string(openrtb_ext.HbSizeConstantKey)] = size
				}
				if hasCacheId {
					bidExt.Prebid.Targeting[string(openrtb_ext.HbCacheKey)] = cacheId
				}
				if deal, ok := bidExt.Prebid.Targeting[hbDealIdBidderKey]; ok {
					bidExt.Prebid.Targeting[string(openrtb_ext.HbDealIdConstantKey)] = deal
				}
				if bidderName == "audienceNetwork" {
					bidExt.Prebid.Targeting[string(openrtb_ext.HbCreativeLoadMethodConstantKey)] = openrtb_ext.HbCreativeLoadMethodDemandSDK
				} else {
					bidExt.Prebid.Targeting[string(openrtb_ext.HbCreativeLoadMethodConstantKey)] = openrtb_ext.HbCreativeLoadMethodHTML
				}
			}

			if hasCacheId {
				bidExt.Prebid.Targeting[openrtb_ext.HbCacheKey.BidderKey(bidderName, maxKeyLength)] = cacheId
			}

			bid.Ext, err1 = json.Marshal(bidExt)
		}
	})
}

// setTargeting writes all the targeting params into the bids.
// If any errors occur when setting the targeting params for a particular bid, then that bid will be ejected from the auction.
//
// The one exception is the `hb_cache_id` key. Since our APIs explicitly document cache keys to be on a "best effort" basis,
// it's ok if those stay in the auction. For now, this method implements a very naive cache strategy.
// In the future, we should implement a more clever retry & backoff strategy to balance the success rate & performance.
func setTargeting(ctx context.Context, cache prebid_cache_client.Client, numImps int, seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, ext map[openrtb_ext.BidderName]*seatResponseExtra) {
	winningBids, _, winningBidsByBidder := findWinners(seatBids, numImps)
	roundedPrices := makeRoundedPrices(openrtb_ext.PriceGranularityLow /* TODO: Fix */, winningBids, winningBidsByBidder)
	var cacheIds map[*openrtb.Bid]string
	if cache != nil {
		cacheIds = doCache(ctx, cache, winningBids, winningBidsByBidder, roundedPrices)
	}

	setTargetingKeys(roundedPrices, cacheIds, winningBids, winningBidsByBidder)
}

func findWinners(seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, numImps int) (winningBids map[string]*pbsOrtbBid, winningBidders map[string]openrtb_ext.BidderName, winningBidsByBidder map[string]map[openrtb_ext.BidderName]*pbsOrtbBid) {
	// winningBids is a map from imp.id to the highest overall CPM bid in that imp.
	winningBids = make(map[string]*pbsOrtbBid, numImps)
	// winningBidders is a map from imp.id to the BidderName which made the winning Bid.
	winningBidders = make(map[string]openrtb_ext.BidderName, numImps)
	// winningBidsFromBidder stores the highest bid on each imp by each bidder.
	winningBidsByBidder = make(map[string]map[openrtb_ext.BidderName]*pbsOrtbBid, numImps)

	for bidderName, seatBid := range seatBids {
		if seatBid == nil {
			log.Printf("Nil bids for bidder: %s", string(bidderName))
		}
		for _, bid := range seatBid.bids {
			cpm := bid.bid.Price
			wbid, ok := winningBids[bid.bid.ImpID]
			if !ok || cpm > wbid.bid.Price {
				winningBidders[bid.bid.ImpID] = bidderName
				winningBids[bid.bid.ImpID] = bid
			}
			if bidMap, ok := winningBidsByBidder[bid.bid.ImpID]; ok {
				bestSoFar, ok := bidMap[bidderName]
				if !ok || cpm > bestSoFar.bid.Price {
					bidMap[bidderName] = bid
				}
			} else {
				winningBidsByBidder[bid.bid.ImpID] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
				winningBidsByBidder[bid.bid.ImpID][bidderName] = bid
			}
		}
	}

	return
}

func makeRoundedPrices(priceGranularity openrtb_ext.PriceGranularity, winningBids map[string]*pbsOrtbBid, winningBidsByBidder map[string]map[openrtb_ext.BidderName]*pbsOrtbBid) map[*pbsOrtbBid]string {
	roundedPrices := make(map[*pbsOrtbBid]string, 5*len(winningBids))
	for _, topBidsPerImp := range winningBidsByBidder {
		for _, topBidPerBidder := range topBidsPerImp {
			roundedPrice, err := buckets.GetPriceBucketString(topBidPerBidder.bid.Price, priceGranularity)
			if err != nil {
				glog.Errorf(`Error rounding price according to granularity. This shouldn't happen unless /openrtb2 input validation is buggy. Granularity was "%s".`, priceGranularity, err)
			}
			roundedPrices[topBidPerBidder] = roundedPrice
		}
	}
	return roundedPrices
}

func doCache(ctx context.Context, cache prebid_cache_client.Client, winningBids map[string]*pbsOrtbBid, winningBidsByBidder map[string]map[openrtb_ext.BidderName]*pbsOrtbBid, roundedPrices map[*pbsOrtbBid]string) map[*openrtb.Bid]string {
	toCache := make([]*openrtb.Bid, 0, len(roundedPrices))

	for _, topBidsPerImp := range winningBidsByBidder {
		for _, topBidPerBidder := range topBidsPerImp {
			// Fixes #199
			if roundedPrice, ok := roundedPrices[topBidPerBidder]; ok && strings.ContainsAny(roundedPrice, "123456789") {
				toCache = append(toCache, topBidPerBidder.bid)
			}
		}
	}
	return cacheOrtbBids(ctx, cache, toCache)
}

func setTargetingKeys(roundedPrices map[*pbsOrtbBid]string, cacheIds map[*openrtb.Bid]string, winningBids map[string]*pbsOrtbBid, winningBidsByBidder map[string]map[openrtb_ext.BidderName]*pbsOrtbBid) {
	for impId, topBidsPerImp := range winningBidsByBidder {
		overallWinner := winningBids[impId]
		for bidderName, topBidPerBidder := range topBidsPerImp {
			isOverallWinner := overallWinner == topBidPerBidder

			targets := make(map[string]string, 10)
			if cpm, ok := roundedPrices[topBidPerBidder]; ok {
				addKeys(targets, openrtb_ext.HbpbConstantKey, cpm, bidderName, isOverallWinner)
			}
			addKeys(targets, openrtb_ext.HbBidderConstantKey, string(bidderName), bidderName, isOverallWinner)
			if hbSize := makeHbSize(topBidPerBidder.bid); hbSize != "" {
				addKeys(targets, openrtb_ext.HbSizeConstantKey, hbSize, bidderName, isOverallWinner)
			}
			if cacheId, ok := cacheIds[topBidPerBidder.bid]; ok {
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
