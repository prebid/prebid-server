package metrics

import (
	"github.com/prebid/prebid-server/pbs"
)

// PBSMetrics logs useful metrics to InfluxDB.
//
// Implementations of this interface should be threadsafe, so they can be used in multiple goroutines.
type PBSMetrics interface {
	// ServerStartedRequest should be called whenever PBS starts to serve an incoming request
	StartAuctionRequest(requestInfo *AuctionRequestInfo) ServerRequestFollowups

	// BidderStartedRequest should be called just before PBS calls Adapter.Call
	StartBidRequest(auctionRequestInfo *AuctionRequestInfo, bidRequestInfo *BidRequestInfo) BidderRequestFollowups
}

// RequestSource is the list of sources where requests might come from.
// This obviously isn't comprehensive... just defined on an as-needed basis.
type RequestSource int

const (
	// Safari has restrictive policies on 3rd party cookies. This helps measure how much of an effect that has.
	SAFARI RequestSource = iota
	APP
	OTHER
)

func (source RequestSource) String() string {
	switch source {
	case SAFARI:
		return "safari"
	case APP:
		return "app"
	default:
		return "other"
	}
}

// AuctionRequestInfo contains data about the request for bids which came into PBS.
type AuctionRequestInfo struct {
	AccountId     string
	RequestSource RequestSource
	HasCookie     bool
}

// BidRequestInfo contains data about the particular Bidder who PBS is requesting bids from.
type BidRequestInfo struct {
	// Bidder is the bidder to whom we're making the request.
	Bidder *pbs.PBSBidder
}

// ServerRequestFollowups contains functions which log followup data for the a particular request.
type ServerRequestFollowups interface {
	// Completed should be called after the server is done with the request. If successful, err can be nil.
	Completed(err error)
}

type BidderRequestFollowups interface {
	// BidderResponded should be called with the bidder's response. This is the return from Adapter.Call()
	BidderResponded(pbs.PBSBidSlice, error)
}
