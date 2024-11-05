package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// ProcessedAuctionRequest hooks are invoked after the request is parsed
// and enriched with additional data.
//
// At this stage, account config is available,
// so it can be configured at the account-level execution plan,
// the account-level module config is passed to hooks.
//
// Rejection results in sending an empty BidResponse
// with the NBR code indicating the rejection reason.
type ProcessedAuctionRequest interface {
	HandleProcessedAuctionHook(
		context.Context,
		ModuleInvocationContext,
		ProcessedAuctionRequestPayload,
	) (HookResult[ProcessedAuctionRequestPayload], error)
}

// ProcessedAuctionRequestPayload consists of the openrtb_ext.RequestWrapper object.
// Hooks are allowed to modify openrtb_ext.RequestWrapper using mutations.
type ProcessedAuctionRequestPayload struct {
	Request *openrtb_ext.RequestWrapper
}

func (parp *ProcessedAuctionRequestPayload) GetBidderRequestPayload() *openrtb_ext.RequestWrapper {
	return parp.Request
}

func (parp *ProcessedAuctionRequestPayload) SetBidderRequestPayload(br *openrtb_ext.RequestWrapper) {
	parp.Request = br
}
