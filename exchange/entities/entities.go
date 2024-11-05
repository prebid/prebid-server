package entities

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// PbsOrtbSeatBid is a SeatBid returned by an AdaptedBidder.
//
// This is distinct from the openrtb2.SeatBid so that the prebid-server ext can be passed back with typesafety.
type PbsOrtbSeatBid struct {
	// Bids is the list of Bids which this AdaptedBidder wishes to make.
	Bids []*PbsOrtbBid
	// Currency is the Currency in which the Bids are made.
	// Should be a valid Currency ISO code.
	Currency string
	// fledgeAuctionConfigs is quasi-opaque data passed back for in-browser interest group auction.
	// if exists, it should be passed through even if bids[] is empty.
	FledgeAuctionConfigs []*openrtb_ext.FledgeAuctionConfig
	// HttpCalls is the list of debugging info. It should only be populated if the request.test == 1.
	// This will become response.ext.debug.httpcalls.{bidder} on the final Response.
	HttpCalls []*openrtb_ext.ExtHttpCall
	// Seat defines whom these extra Bids belong to.
	Seat string
}

// PbsOrtbBid is a Bid returned by an AdaptedBidder.
//
// PbsOrtbBid.Bid.Ext will become "response.seatbid[i].Bid.ext.bidder" in the final OpenRTB response.
// PbsOrtbBid.BidMeta will become "response.seatbid[i].Bid.ext.prebid.meta" in the final OpenRTB response.
// PbsOrtbBid.BidType will become "response.seatbid[i].Bid.ext.prebid.type" in the final OpenRTB response.
// PbsOrtbBid.BidTargets does not need to be filled out by the Bidder. It will be set later by the exchange.
// PbsOrtbBid.BidVideo is optional but should be filled out by the Bidder if BidType is video.
// PbsOrtbBid.BidEvents is set by exchange when event tracking is enabled
// PbsOrtbBid.BidFloors is set by exchange when floors is enabled
// PbsOrtbBid.DealPriority is optionally provided by adapters and used internally by the exchange to support deal targeted campaigns.
// PbsOrtbBid.DealTierSatisfied is set to true by exchange.updateHbPbCatDur if deal tier satisfied otherwise it will be set to false
// PbsOrtbBid.GeneratedBidID is unique Bid id generated by prebid server if generate Bid id option is enabled in config
type PbsOrtbBid struct {
	Bid               *openrtb2.Bid
	BidMeta           *openrtb_ext.ExtBidPrebidMeta
	BidType           openrtb_ext.BidType
	BidTargets        map[string]string
	BidVideo          *openrtb_ext.ExtBidPrebidVideo
	BidEvents         *openrtb_ext.ExtBidPrebidEvents
	BidFloors         *openrtb_ext.ExtBidPrebidFloors
	DealPriority      int
	DealTierSatisfied bool
	GeneratedBidID    string
	OriginalBidCPM    float64
	OriginalBidCur    string
	TargetBidderCode  string
	AdapterCode       openrtb_ext.BidderName
}
