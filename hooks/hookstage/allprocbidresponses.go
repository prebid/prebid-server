package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AllProcessedBidResponses interface {
	HandleAllProcessedBidResponsesHook(
		context.Context,
		*ModuleContext,
		AllProcessedBidResponsesPayload,
	) (HookResult[AllProcessedBidResponsesPayload], error)
}

type AllProcessedBidResponsesPayload struct {
	Responses map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
}
