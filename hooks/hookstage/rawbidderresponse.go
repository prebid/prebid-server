package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/v2/adapters"
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

// RawBidderResponsePayload consists of a list of adapters.TypedBid
// objects representing bids returned by a particular bidder.
// Hooks are allowed to modify bids using mutations.
type RawBidderResponsePayload struct {
	Bids   []*adapters.TypedBid
	Bidder string
}
