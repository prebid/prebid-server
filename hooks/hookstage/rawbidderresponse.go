package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/v3/adapters"
)

// RawBidderResponse hooks are invoked for each bidder participating in auction.
//
// At this stage, account config is available,
// so it can be configured at the account-level execution plan,
// the account-level module config is passed to hooks.
//
// Rejection results in ignoring the bidder's response.
type RawBidderResponse interface {
	HandleRawBidderResponseHook(
		context.Context,
		ModuleInvocationContext,
		RawBidderResponsePayload,
	) (HookResult[RawBidderResponsePayload], error)
}

// RawBidderResponsePayload consists of a bidder response returned by a particular bidder.
// Hooks are allowed to modify bidder response using mutations.
type RawBidderResponsePayload struct {
	BidderResponse *adapters.BidderResponse
	Bidder         string
}
