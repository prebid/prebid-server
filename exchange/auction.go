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
	// winningBids is a map from imp.id to the highest CPM bid in that imp.
	winningBids map[string]*openrtb.Bid
	// winningBidders is a map from imp.id to the BidderName which made the winning Bid.
	winningBidders map[string]openrtb_ext.BidderName
	// cachedBids stores the cache ID for each bid.
	cachedBids map[*openrtb.Bid]string
}

func newAuction(numImps int) *auction {
	return &auction{
		winningBids:    make(map[string]*openrtb.Bid, numImps),
		winningBidders: make(map[string]openrtb_ext.BidderName, numImps),
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

// forEachWinner runs the callback function on each winning Bid.
func (auction *auction) forEachWinner(callback func(impID string, bidder openrtb_ext.BidderName, bid *openrtb.Bid)) {
	if auction == nil {
		return
	}

	for id, bid := range auction.winningBids {
		callback(id, auction.winningBidders[id], bid)
	}
}
