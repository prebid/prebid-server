package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/adapters"
)

// RawBidResponse hooks are invoked for each bidder participating in auction.
//
// At this stage, account config is available,
// so it can be configured at the account-level execution plan,
// the account-level module config is passed to hooks.
//
// Rejection results in ignoring the bidder's response.
type RawBidResponse interface {
	HandleRawBidResponseHook(
		context.Context,
		InvocationContext,
		RawBidResponsePayload,
	) (HookResult[RawBidResponsePayload], error)
}

// RawBidResponsePayload consists of a list of adapters.TypedBid
// objects representing bids returned by a particular bidder.
// Hooks are allowed to modify bids using mutations.
type RawBidResponsePayload struct {
	Bids []*adapters.TypedBid
}
