package stages

import (
	"context"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type RawBidResponseHook interface {
	Call(
		context.Context,
		invocation.InvocationContext,
		RawBidResponsePayload,
	) (invocation.HookResult[RawBidResponsePayload], error)
}

type RawBidResponsePayload struct {
	Bids []*adapters.TypedBid
}
