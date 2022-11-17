package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
)

type ProcessedAuction interface {
	HandleProcessedAuctionHook(
		context.Context,
		ModuleContext,
		ProcessedAuctionPayload,
	) (HookResult[ProcessedAuctionPayload], error)
}

type ProcessedAuctionPayload struct {
	BidRequest *openrtb2.BidRequest
}
