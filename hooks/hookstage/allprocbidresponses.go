package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/hooks/invocation"
)

type AllProcessedBidResponses interface {
	HandleAllProcBidResponsesHook(
		context.Context,
		invocation.Context,
		AllProcessedBidResponsesPayload,
	) (invocation.HookResult[AllProcessedBidResponsesPayload], error)
}

type AllProcessedBidResponsesPayload struct {
	// todo: decide what payload to use within hook invocation task
	// initially, we planned to use map[openrtb_ext.BidderName]*exchange.pbsOrtbSeatBid, but the type is not exported
}
