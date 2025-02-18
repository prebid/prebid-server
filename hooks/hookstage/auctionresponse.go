package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// AuctionResponse hooks are invoked at the very end of request processing.
// The hooks are invoked even if the request was rejected at earlier stages.
//
// At this stage, account config is available,
// so it can be configured at the account-level execution plan,
// the account-level module config is passed to hooks.
//
// Rejection has no effect and is completely ignored at this stage.
type AuctionResponse interface {
	HandleAuctionResponseHook(
		context.Context,
		ModuleInvocationContext,
		AuctionResponsePayload,
	) (HookResult[AuctionResponsePayload], error)
}

// AuctionResponsePayload consists of a final openrtb2.BidResponse
// object that will be sent back to the requester.
// Hooks are allowed to modify openrtb2.BidResponse object.
type AuctionResponsePayload struct {
	BidResponse *openrtb2.BidResponse
}
