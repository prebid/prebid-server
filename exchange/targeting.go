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

	hbPbBidderKey := openrtb_ext.HbpbConstantKey.BidderKey(name, t.lengthMax)
	hbBidderBidderKey := openrtb_ext.HbBidderConstantKey.BidderKey(name, t.lengthMax)
	hbSizeBidderKey := openrtb_ext.HbSizeConstantKey.BidderKey(name, t.lengthMax)
	hbDealIdBidderKey := openrtb_ext.HbDealIdConstantKey.BidderKey(name, t.lengthMax)

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

// addTargetsToCompletedAuction takes a _completed_ auction, and adds all the appropriate targeting keys to it.
// Once this has been called, auction.addBid() should _not_ be called anymore.
func (t *targetData) addTargetsToCompletedAuction(auction *auction) {
	if t == nil {
		return
	}

	auction.forEachWinner(func(id string, bidderName openrtb_ext.BidderName, bid *openrtb.Bid) {
		bidExt := new(openrtb_ext.ExtBid)
		err1 := json.Unmarshal(bid.Ext, bidExt)
		if err1 == nil && bidExt.Prebid.Targeting != nil {
			hbPbBidderKey := openrtb_ext.HbpbConstantKey.BidderKey(bidderName, t.lengthMax)
			hbBidderBidderKey := openrtb_ext.HbBidderConstantKey.BidderKey(bidderName, t.lengthMax)
			hbSizeBidderKey := openrtb_ext.HbSizeConstantKey.BidderKey(bidderName, t.lengthMax)
			hbDealIdBidderKey := openrtb_ext.HbDealIdConstantKey.BidderKey(bidderName, t.lengthMax)

			bidExt.Prebid.Targeting[string(openrtb_ext.HbpbConstantKey)] = bidExt.Prebid.Targeting[hbPbBidderKey]
			bidExt.Prebid.Targeting[string(openrtb_ext.HbBidderConstantKey)] = bidExt.Prebid.Targeting[hbBidderBidderKey]
			if size, ok := bidExt.Prebid.Targeting[hbSizeBidderKey]; ok {
				bidExt.Prebid.Targeting[string(openrtb_ext.HbSizeConstantKey)] = size
			}
			if cacheId, ok := auction.cacheId(bid); ok {
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
			bid.Ext, err1 = json.Marshal(bidExt)
		}
	})

	auction.forEachDeal(func(bid *openrtb.Bid) {
		if cacheId, ok := auction.cacheId(bid); ok {
			bidExt := new(openrtb_ext.ExtBid)
			err1 := json.Unmarshal(bid.Ext, bidExt)
			if err1 == nil && bidExt.Prebid.Targeting != nil {
				bidExt.Prebid.Targeting[openrtb_ext.HbCacheKey.DealKey(bid.DealID, t.lengthMax)] = cacheId
			}
		}
	})
}
