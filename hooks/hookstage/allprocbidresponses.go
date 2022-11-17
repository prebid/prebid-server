package hookstage

import (
	"context"
)

type AllProcessedBidResponses interface {
	HandleAllProcessedBidResponsesHook(
		context.Context,
		ModuleContext,
		AllProcessedBidResponsesPayload,
	) (HookResult[AllProcessedBidResponsesPayload], error)
}

type AllProcessedBidResponsesPayload struct {
	// Responses []*adapters.BidderResponse
	// todo: decide what payload to use within the hook invocation task
}
