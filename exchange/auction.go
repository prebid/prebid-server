package exchange

import (
	"context"
	"strings"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/prebid_cache_client"
)

func newAuction(seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, numImps int) *auction {
	winningBids := make(map[string]*pbsOrtbBid, numImps)
	winningBidsByBidder := make(map[string]map[openrtb_ext.BidderName]*pbsOrtbBid, numImps)

	for bidderName, seatBid := range seatBids {
		if seatBid != nil {
			for _, bid := range seatBid.bids {
				cpm := bid.bid.Price
				wbid, ok := winningBids[bid.bid.ImpID]
				if !ok || cpm > wbid.bid.Price {
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
	}

	return &auction{
		winningBids:         winningBids,
		winningBidsByBidder: winningBidsByBidder,
	}
}

func (a *auction) setRoundedPrices(priceGranularity openrtb_ext.PriceGranularity) {
	roundedPrices := make(map[*pbsOrtbBid]string, 5*len(a.winningBids))
	for _, topBidsPerImp := range a.winningBidsByBidder {
		for _, topBidPerBidder := range topBidsPerImp {
			roundedPrice, err := GetCpmStringValue(topBidPerBidder.bid.Price, priceGranularity)
			if err != nil {
				glog.Errorf(`Error rounding price according to granularity. This shouldn't happen unless /openrtb2 input validation is buggy. Granularity was "%v".`, priceGranularity)
			}
			roundedPrices[topBidPerBidder] = roundedPrice
		}
	}
	a.roundedPrices = roundedPrices
}

func (a *auction) doCache(ctx context.Context, cache prebid_cache_client.Client) {
	toCache := make([]*openrtb.Bid, 0, len(a.roundedPrices))

	for _, topBidsPerImp := range a.winningBidsByBidder {
		for _, topBidPerBidder := range topBidsPerImp {
			// Fixes #199
			if roundedPrice, ok := a.roundedPrices[topBidPerBidder]; ok && strings.ContainsAny(roundedPrice, "123456789") {
				toCache = append(toCache, topBidPerBidder.bid)
			}
		}
	}

	a.cacheIds = cacheBids(ctx, cache, toCache)
}

type auction struct {
	// winningBids is a map from imp.id to the highest overall CPM bid in that imp.
	winningBids map[string]*pbsOrtbBid
	// winningBidsByBidder stores the highest bid on each imp by each bidder.
	winningBidsByBidder map[string]map[openrtb_ext.BidderName]*pbsOrtbBid
	// roundedPrices stores the price strings rounded for each bid according to the price granularity.
	roundedPrices map[*pbsOrtbBid]string
	// cacheIds stores the UUIDs from Prebid Cache for each bid.
	cacheIds map[*openrtb.Bid]string
}
