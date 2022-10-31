package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AllProcessedBidResponses interface {
	HandleAllProcBidResponsesHook(
		context.Context,
		invocation.InvocationContext,
		AllProcessedBidResponsesPayload,
	) (invocation.HookResult[AllProcessedBidResponsesPayload], error)
}

type AllProcessedBidResponsesPayload map[openrtb_ext.BidderName]*exchange.PbsOrtbSeatBid
