package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
)

type ProcessedAuction interface {
	HandleProcessedAuctionHook(
		context.Context,
		InvocationContext,
		ProcessedAuctionPayload,
	) (HookResult[ProcessedAuctionPayload], error)
}

type ProcessedAuctionPayload struct {
	BidRequest *openrtb2.BidRequest
}
