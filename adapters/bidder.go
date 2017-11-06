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
	//
	// A Bidder *may* return two non-nil values here. Errors should describe situations which
	// make the bid (or no-bid) "less than ideal." Common examples include:
	//
	// 1. HTTP connection issues.
	// 2. Imps with Media Types which this Bidder doesn't support.
	// 3. The Context timeout expired before all expected bids were returned.
	// 4. The Server sent back an unexpected Response, so some bids were ignored.
	//
	// Any errors will be user-facing... so the error messages should help publishers understand
	// what might account for "bad" bids.
	Bid(ctx context.Context, request *openrtb.BidRequest) (*PBSOrtbSeatBid, []error)
}

// PBSOrtbBid is a Bid returned by a Bidder.
//
// PBSOrtbBid.Bid.Ext will become "response.seatbid[bidder].bid[i].ext.bidder" in the final PBS response.
type PBSOrtbBid struct {
	Bid *openrtb.Bid
	// Cache must not be nil if request.ext.prebid.cache.markup was 1
	Cache *openrtb_ext.ExtResponseCache
	Type  openrtb_ext.BidType
}

// PBSOrtbBid is a SeatBid returned by a Bidder.
//
// PBS does not support the "Group" option from the OpenRTB SeatBid. All bids must be winnable independently.
type PBSOrtbSeatBid struct {
	// Bids is the list of bids in this SeatBid. If len(Bids) == 0, no SeatBid will be entered for this bidder.
	// This is because the OpenRTB 2.5 spec requires at least one bid for each SeatBid.
	Bids []*PBSOrtbBid
	// ServerCalls will become response.ext.debug.servercalls.{bidder} on the final Response.
	ServerCalls []*openrtb_ext.ExtServerCall
	// Ext will become response.seatbid[i].ext.{bidder} on the final Response, *only if* len(Bids) > 0.
	// If len(Bids) == 0, no SeatBid will be entered, and this field will be ignored.
	Ext openrtb.RawJSON
}

// ExtImpBidder is a struct which can be used by Bidders to unmarshal any request.imp[i].ext.
type ExtImpBidder struct {
	Prebid *openrtb_ext.ExtBidPrebid `json:"prebid"`

	// Bidder will contain the data for the bidder-specific extension.
	// Bidders should unmarshal this using their corresponding openrtb_ext.ExtImp{Bidder} struct.
	Bidder openrtb.RawJSON `json:"bidder"`
}
