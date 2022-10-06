package stages

import (
	"context"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type RawBidResponseHook interface {
	Call(
		context.Context,
		invocation.Context,
		RawBidResponsePayload,
	) (invocation.HookResult[RawBidResponsePayload], error)
}

type RawBidResponsePayload struct {
	Bids []*adapters.TypedBid
}
