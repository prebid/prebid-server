package adapters

import (
	"context"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Bidders participate in prebid-server auctions.
type Bidder interface {
	// Bid should return the SeatBid containing all bids used by this bidder.
	//
	// All `Ext` fields from the argument request are generated from contract classes in openrtb_ext.
	// Each bidder may define their own `Ext` format there.
	//
	// All `Ext` fields inside the returned SeatBid must also be generated from the contract classes in openrtb_ext.
	//
	// Bid should still attempt to return a SeatBid, even if some errors occurred. If there are no bids, return nil.
	// Errors will be processed by prebid-server core code and logged or reported to the user as appropriate.
	Bid(ctx context.Context, request *openrtb.BidRequest) *BidderResponse
}

// BidderResponse carries all the data needed for a Bidder's response.
type BidderResponse struct {
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

// BidData packages the Bidder's openrtb Bid with the Prebid extension.
// When a HttpBidder returns this, the only PrebidExt field it **must** populate is the Type.
// The other fields will be populated in the prebid core.
type BidData struct {
	Bid       *openrtb.Bid
	PrebidExt *openrtb_ext.ExtBidPrebid
}
