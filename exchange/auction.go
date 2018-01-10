package exchange

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// auction stores the Bids for a single call to Exchange.HoldAuction().
// Construct these with the newAuction() function.
type auction struct {
	// bidsWithDeals stores all of the bids in the auction which are part of a Deal.
	bidsWithDeals []*openrtb.Bid
	// winningBids is a map from imp.id to the highest overall CPM bid in that imp.
	winningBids map[string]*openrtb.Bid
	// winningBidders is a map from imp.id to the BidderName which made the winning Bid.
	winningBidders map[string]openrtb_ext.BidderName
	// winningBidsFromBidder stores the highest bid on each imp by each bidder.
	winningBidsByBidder map[string]map[openrtb_ext.BidderName]*openrtb.Bid
	// cachedBids stores the cache ID for each bid, if it exists.
	// This is set by cacheBids() in cache.go, and is nil beforehand.
	cachedBids map[*openrtb.Bid]string
}

func newAuction(numImps int) *auction {
	return &auction{
		winningBids:         make(map[string]*openrtb.Bid, numImps),
		winningBidders:      make(map[string]openrtb_ext.BidderName, numImps),
		winningBidsByBidder: make(map[string]map[openrtb_ext.BidderName]*openrtb.Bid, numImps),
	}
}

// addBid should be called for each bid which is "officially" valid for the auction.
func (auction *auction) addBid(name openrtb_ext.BidderName, bid *openrtb.Bid) {
	if auction == nil {
		return
	}

	if bid.DealID != "" {
		auction.bidsWithDeals = append(auction.bidsWithDeals, bid)
	}

	cpm := bid.Price
	wbid, ok := auction.winningBids[bid.ImpID]
	if !ok || cpm > wbid.Price {
		auction.winningBidders[bid.ImpID] = name
		auction.winningBids[bid.ImpID] = bid
	}
	if bidMap, ok := auction.winningBidsByBidder[bid.ImpID]; ok {
		bestSoFar, ok := bidMap[name]
		if !ok || cpm > bestSoFar.Price {
			bidMap[name] = bid
		}
	} else {
		auction.winningBidsByBidder[bid.ImpID] = make(map[openrtb_ext.BidderName]*openrtb.Bid)
		auction.winningBidsByBidder[bid.ImpID][name] = bid
	}
}

func (auction *auction) numImps() int {
	if auction == nil {
		return 0
	} else {
		return len(auction.winningBids)
	}
}

func (auction *auction) numDealBids() int {
	if auction == nil {
		return 0
	} else {
		return len(auction.bidsWithDeals)
	}
}

func (auction *auction) forEachDeal(callback func(bid *openrtb.Bid)) {
	for _, bidWithDeal := range auction.bidsWithDeals {
		callback(bidWithDeal)
	}
}

func (auction *auction) forEachCachedBid(callback func(bid *openrtb.Bid, id string)) {
	for bid, id := range auction.cachedBids {
		callback(bid, id)
	}
}

func (auction *auction) cacheId(bid *openrtb.Bid) (id string, exists bool) {
	id, exists = auction.cachedBids[bid]
	return
}

// forEachBestBid runs the callback function on every bid which is the highest one for each Bidder on each Imp.
func (auction *auction) forEachBestBid(callback func(impID string, bidder openrtb_ext.BidderName, bid *openrtb.Bid, winner bool)) {
	if auction == nil {
		return
	}

	for impId, bidderMap := range auction.winningBidsByBidder {
		overallWinner, _ := auction.winningBids[impId]
		for bidderName, bid := range bidderMap {
			callback(impId, bidderName, bid, bid == overallWinner)
		}
	}
}
