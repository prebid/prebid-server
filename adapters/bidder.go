package adapters

import (
	"context"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Bidders participate in prebid-server auctions.
type Bidder interface {
	// Bid gets the bids from this bidder for the given request.
	//
	// Per the OpenRTB spec, a SeatBid may not be empty. If so, then any errors which contribute
	// to the "no bid" bid should be returned here instead.
	Bid(ctx context.Context, request *openrtb.BidRequest) (*PBSOrtbSeatBid, []error)
}

// BidderResponse carries all the data needed for a Bidder's response.
type BidderResponse struct {
	SeatBid *openrtb.SeatBid

	// Bids contains all the bids that the Bidder wants to enter.
	// This can be nil (for no bids), but should not contain nil elements.
	Bids []*openrtb.Bid
	// ServerCalls stores some debugging info.
	// This is only required if the input request.Test was 1.
	ServerCalls []*openrtb_ext.ExtServerCall
	// Errors should contain a list of errors which occurred internally. These should report
	// any conditions which result in "no" or "subpar" bids. For example:
	//
	// 1. The openrtb request needs an ad type which this bidder doesn't support.
	// 2. The auction timed out before all the bids were entered.
	// 3. The remote server returned unexpected input.
	Errors []error
}

// PBSOrtbBid is a Bid returned by a Bidder.
//
// PBSOrtbBid.Bid.Ext will become "response.seatbid[bidder].bid[i].ext.bidder" in the final PBS response.
type PBSOrtbBid struct {
	Bid *openrtb.Bid
	// Cache must not be nil if request.ext.prebid.cache.markup was 1
	Cache *openrtb_ext.ExtResponseCache
	Type openrtb_ext.BidType
	ResponseTimeMillis int
}

// PBSOrtbBid is a SeatBid returned by a Bidder.
//
// PBS does not support the "Group" option from the OpenRTB SeatBid.
// All bids must be winnable independently.
type PBSOrtbSeatBid struct {
	Bid []*PBSOrtbBid
	Ext *openrtb_ext.ExtSeatBid
}
