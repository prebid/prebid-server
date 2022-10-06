package stages

import (
	"context"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type ProcessedAuctionHook interface {
	Call(
		context.Context,
		invocation.Context,
		ProcessedAuctionPayload,
	) (invocation.HookResult[ProcessedAuctionPayload], error)
}

type ProcessedAuctionPayload struct {
	BidRequest *openrtb2.BidRequest
}
