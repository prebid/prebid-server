package metrics

import (
	"github.com/prebid/prebid-server/pbs"
)

// PBSMetrics logs useful metrics to InfluxDB.
//
// Implementations of this interface should be threadsafe, so they can be used in multiple goroutines.
type PBSMetrics interface {
	// ServerStartedRequest should be called whenever PBS starts to serve an incoming request
	ServerStartedRequest(requestInfo *RequestInfo) ServerRequestFollowups

	// BidderStartedRequest should be called just before PBS calls Adapter.Call
	BidderStartedRequest(requestInfo BidderRequestInfo) BidderRequestFollowups
}

type RequestInfo struct {
	Publisher  string
	IsSafari bool
	IsApp    bool
}

type BidderRequestInfo struct {
}

// ServerRequestFollowups contains functions which log followup data for the a particular request.
type ServerRequestFollowups interface {
	// Completed should be called after the server has completed the request successfully.
	Completed()

	// Failed should be called after teh server failed to respond to a request for some reason.
	Failed()
}

type BidderRequestFollowups interface {
	// NoBid should be called if the bidder has responded with a No-Bid request.
	NoBid()

	// GotBids should be called if the bidder responded with bids before the timeout.
	GotBids(pbs.PBSBidSlice)

	// TimedOut should be called if the bidder timed out, so its bid never made it into the auction.
	TimedOut()
}
