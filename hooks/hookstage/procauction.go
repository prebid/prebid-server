package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
)

// ProcessedAuction hooks are invoked after the request is parsed
// and enriched with additional data.
//
// At this stage, account config is available,
// so it can be configured at the account-level execution plan,
// the account-level module config is passed to hooks.
//
// Rejection results in sending an empty BidResponse
// with the NBR code indicating the rejection reason.
type ProcessedAuction interface {
	HandleProcessedAuctionHook(
		context.Context,
		InvocationContext,
		ProcessedAuctionPayload,
	) (HookResult[ProcessedAuctionPayload], error)
}

// ProcessedAuctionPayload consists of the openrtb2.BidRequest object.
// Hooks are allowed to modify openrtb2.BidRequest using mutations.
type ProcessedAuctionPayload struct {
	BidRequest *openrtb2.BidRequest
}
