package hookstage

import (
	"context"
)

// RawAuctionRequest hooks are invoked only for "/openrtb2/auction"
// endpoint after retrieving the account config,
// but before the request is parsed and any additions are made.
//
// At this stage, account config is available,
// so it can be configured at the account-level execution plan,
// the account-level module config is passed to hooks.
//
// Rejection results in sending an empty BidResponse
// with the NBR code indicating the rejection reason.
type RawAuctionRequest interface {
	HandleRawAuctionHook(
		context.Context,
		ModuleInvocationContext,
		RawAuctionRequestPayload,
	) (HookResult[RawAuctionRequestPayload], error)
}

// RawAuctionRequestPayload represents a raw body of the openrtb2.BidRequest.
// Hooks are allowed to modify body using mutations.
type RawAuctionRequestPayload []byte
