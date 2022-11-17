package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
)

type AuctionResponse interface {
	HandleAuctionResponseHook(
		context.Context,
		ModuleContext,
		AuctionResponsePayload,
	) (HookResult[AuctionResponsePayload], error)
}

type AuctionResponsePayload struct {
	BidResponse *openrtb2.BidResponse
}
