package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/adapters"
)

type RawBidResponse interface {
	HandleRawBidResponseHook(
		context.Context,
		InvocationContext,
		RawBidResponsePayload,
	) (HookResult[RawBidResponsePayload], error)
}

type RawBidResponsePayload struct {
	Bids []*adapters.TypedBid
}
