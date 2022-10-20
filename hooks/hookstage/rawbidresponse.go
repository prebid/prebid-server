package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type RawBidResponse interface {
	HandleRawBidResponseHook(
		context.Context,
		invocation.InvocationContext,
		RawBidResponsePayload,
	) (invocation.HookResult[RawBidResponsePayload], error)
}

type RawBidResponsePayload struct {
	Bids []*adapters.TypedBid
}
