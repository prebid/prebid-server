package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
)

// BidRequest hooks are invoked for each bidder participating in auction.
//
// At this stage, account config is available,
// so it can be configured at the account-level execution plan,
// the account-level module config is passed to hooks.
//
// Rejection results in skipping the bidder's request.
type BidRequest interface {
	HandleBidRequestHook(
		context.Context,
		InvocationContext,
		BidRequestPayload,
	) (HookResult[BidRequestPayload], error)
}

// BidRequestPayload consists of the openrtb2.BidRequest object
// distilled for the particular bidder.
// Hooks are allowed to modify openrtb2.BidRequest using mutations.
type BidRequestPayload struct {
	BidRequest *openrtb2.BidRequest
}
