package stages

import (
	"context"

	"github.com/prebid/prebid-server/hooks/invocation"
)

type AllProcBidResponsesHook interface {
	Call(
		context.Context,
		invocation.Context,
		AllProcBidResponsesPayload,
	) (invocation.HookResult[AllProcBidResponsesPayload], error)
}

type AllProcBidResponsesPayload struct {
	// todo: decide what payload to use within hook invocation task
	// initially, we planned to use map[openrtb_ext.BidderName]*exchange.pbsOrtbSeatBid, but the type is not exported
}
