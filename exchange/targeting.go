package exchange

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs/buckets"
	"strconv"
)

// targetData tracks information about the winning Bid in each Imp.
//
// All functions on this struct are nil-safe. If the targetData struct is nil, then they behave
// like they would if no targeting information is needed.
//
// All functions on this struct are all nil-safe.
// If the value is nil, then no targeting data will be tracked.
type targetData struct {
	lengthMax        int
	priceGranularity openrtb_ext.PriceGranularity
	// These two dictionaries index on imp.id to identify the winning bid for each imp.
	winningBids    map[string]*openrtb.Bid
	winningBidders map[string]openrtb_ext.BidderName
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
	cacheKey := ""

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

	hbPbBidderKey := openrtb_ext.HbpbConstantKey.BidderKey(name, t.lengthMax)
	hbBidderBidderKey := openrtb_ext.HbBidderConstantKey.BidderKey(name, t.lengthMax)
	hbSizeBidderKey := openrtb_ext.HbSizeConstantKey.BidderKey(name, t.lengthMax)
	hbDealIdBidderKey := openrtb_ext.HbDealIdConstantKey.BidderKey(name, t.lengthMax)
	hbCacheIdBidderKey := openrtb_ext.HbCacheIdConstantKey.BidderKey(name, t.lengthMax)

	pbs_kvs := map[string]string{
		hbPbBidderKey:     roundedCpm,
		hbBidderBidderKey: string(name),
	}

	if hbSize != "" {
		pbs_kvs[hbSizeBidderKey] = hbSize
	}
	if len(cacheKey) > 0 {
		pbs_kvs[hbCacheIdBidderKey] = cacheKey
	}
	if len(deal) > 0 {
		pbs_kvs[hbDealIdBidderKey] = deal
	}
	return pbs_kvs, err
}

// addBid should be called for each bid which is "officially" valid for the auction.
//
// This function must be called for all bids before addWinningTargets() is called.
func (t *targetData) addBid(name openrtb_ext.BidderName, bid *openrtb.Bid) {
	if t == nil {
		return
	}
	cpm := bid.Price
	wbid, ok := t.winningBids[bid.ImpID]
	if !ok || cpm > wbid.Price {
		t.winningBidders[bid.ImpID] = name
		t.winningBids[bid.ImpID] = bid
	}
}

// addWinningTargets appends targeting keys to the "winning" bids.
// It should only be called *after* all the calls to addbid().
func (t *targetData) addWinningTargets() {
	if t == nil {
		return
	}

	for id, bid := range t.winningBids {
		bidExt := new(openrtb_ext.ExtBid)
		err1 := json.Unmarshal(bid.Ext, bidExt)
		if err1 == nil && bidExt.Prebid.Targeting != nil {
			hbPbBidderKey := openrtb_ext.HbpbConstantKey.BidderKey(t.winningBidders[id], t.lengthMax)
			hbBidderBidderKey := openrtb_ext.HbBidderConstantKey.BidderKey(t.winningBidders[id], t.lengthMax)
			hbSizeBidderKey := openrtb_ext.HbSizeConstantKey.BidderKey(t.winningBidders[id], t.lengthMax)
			hbDealIdBidderKey := openrtb_ext.HbDealIdConstantKey.BidderKey(t.winningBidders[id], t.lengthMax)
			hbCacheIdBidderKey := openrtb_ext.HbCacheIdConstantKey.BidderKey(t.winningBidders[id], t.lengthMax)

			bidExt.Prebid.Targeting[string(openrtb_ext.HbpbConstantKey)] = bidExt.Prebid.Targeting[hbPbBidderKey]
			bidExt.Prebid.Targeting[string(openrtb_ext.HbBidderConstantKey)] = bidExt.Prebid.Targeting[hbBidderBidderKey]
			if size, ok := bidExt.Prebid.Targeting[hbSizeBidderKey]; ok {
				bidExt.Prebid.Targeting[string(openrtb_ext.HbSizeConstantKey)] = size
			}
			if cache, ok := bidExt.Prebid.Targeting[hbCacheIdBidderKey]; ok {
				bidExt.Prebid.Targeting[string(openrtb_ext.HbCacheIdConstantKey)] = cache
			}
			if deal, ok := bidExt.Prebid.Targeting[hbDealIdBidderKey]; ok {
				bidExt.Prebid.Targeting[string(openrtb_ext.HbDealIdConstantKey)] = deal
			}
			if t.winningBidders[id] == "audienceNetwork" {
				bidExt.Prebid.Targeting[string(openrtb_ext.HbCreativeLoadMethodConstantKey)] = openrtb_ext.HbCreativeLoadMethodDemandSDK
			} else {
				bidExt.Prebid.Targeting[string(openrtb_ext.HbCreativeLoadMethodConstantKey)] = openrtb_ext.HbCreativeLoadMethodHTML
			}
			bid.Ext, err1 = json.Marshal(bidExt)
		}
	}
}
