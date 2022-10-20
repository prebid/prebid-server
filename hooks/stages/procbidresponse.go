package stages

import (
	"context"

	"github.com/prebid/prebid-server/hooks/invocation"
)

type ProcessedBidResponseHook interface {
	Call(
		context.Context,
		invocation.InvocationContext,
		ProcessedBidResponsePayload,
	) (invocation.HookResult[ProcessedBidResponsePayload], error)
}

type ProcessedBidResponsePayload struct {
	// todo: decide what payload to use within hook invocation task
	// initially, we planned to use *exchange.pbsOrtbSeatBid, but the type is not exported
}
