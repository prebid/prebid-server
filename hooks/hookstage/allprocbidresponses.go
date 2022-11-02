package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AllProcessedBidResponses interface {
	HandleAllProcBidResponsesHook(
		context.Context,
		InvocationContext,
		AllProcessedBidResponsesPayload,
	) (HookResult[AllProcessedBidResponsesPayload], error)
}

type AllProcessedBidResponsesPayload map[openrtb_ext.BidderName]*exchange.PbsOrtbSeatBid
