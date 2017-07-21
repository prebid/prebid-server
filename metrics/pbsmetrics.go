package metrics

import (
	"github.com/prebid/prebid-server/pbs"
)

// PBSMetrics logs useful metrics to InfluxDB.
//
// Implementations of this interface should be threadsafe, so they can be used in multiple goroutines.
type PBSMetrics interface {
	// ServerStartedRequest should be called each time the /auction endpoint is hit.
	StartAuctionRequest(requestInfo *AuctionRequestInfo) AuctionRequestFollowups

	// BidderStartedRequest should be called just before PBS calls Adapter.Call
	StartBidderRequest(auctionRequestInfo *AuctionRequestInfo, bidRequestInfo *BidRequestInfo) BidderRequestFollowups

	// StartCookieSyncRequest should be called each time the /cookie_sync endpoint is hit.
	StartCookieSyncRequest()
}

// RequestSource is the list of sources where requests might come from.
// This obviously isn't comprehensive... just defined on an as-needed basis.
type RequestSource int

const (
	// Safari has restrictive policies on 3rd party cookies. This helps measure how much of an effect that has.
	SAFARI RequestSource = iota
	APP
	OTHER
	UNKNOWN
)

func (source RequestSource) String() string {
	switch source {
	case SAFARI:
		return "safari"
	case APP:
		return "app"
	case UNKNOWN:
		return "unknown"
	default:
		return "other"
	}
}

// AuctionRequestInfo contains data about the request for bids which came into PBS.
type AuctionRequestInfo struct {
	// AccountId is the ID of the account requesting this auction.
	AccountId string
	// RequestSource specifies the type of Client which is making the request.
	RequestSource RequestSource
	// HasCookie is true if Prebid Server has any bidders who can ID this user.
	HasCookie bool
}

// BidRequestInfo contains data about the particular Bidder who PBS is requesting bids from.
type BidRequestInfo struct {
	// Bidder is the bidder to whom we're making the request.
	Bidder *pbs.PBSBidder

	// HasCookie is true if this user has an ID *for this bidder*, and false otherwise.
	// If AuctionRequestInfo.HasCookie is false, then this also must be false.
	//
	// See PBSRequest.GetUserID
	HasCookie bool
}

// AuctionRequestFollowups contains functions which log followup data for the a particular request.
type AuctionRequestFollowups interface {
	// Completed should be called after the server is done with the request. If successful, err can be nil.
	Completed(err error)
}

// BidderRequestFollowups contains functions which log followup data from a bidder request.
type BidderRequestFollowups interface {
	// BidderResponded should be called with the bidder's response. This is the return from Adapter.Call()
	BidderResponded(pbs.PBSBidSlice, error)

	// BidderSkipped should be called if Prebid-Server never even called the Bidder's Adapter.
	//
	// Currently the only reason this happens is because the bidder didn't ID the user, and reported
	// that it didn't want to serve bids to those users (see Adapter.SkipNoCookies()).
	BidderSkipped()
}
