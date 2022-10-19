package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type ProcessedAuction interface {
	HandleProcessedAuctionHook(
		context.Context,
		invocation.Context,
		ProcessedAuctionPayload,
	) (invocation.HookResult[ProcessedAuctionPayload], error)
}

type ProcessedAuctionPayload struct {
	BidRequest *openrtb2.BidRequest
}
