package hookstage

import (
	"context"
)

type AllProcessedBidResponses interface {
	HandleAllProcBidResponsesHook(
		context.Context,
		InvocationContext,
		AllProcessedBidResponsesPayload,
	) (HookResult[AllProcessedBidResponsesPayload], error)
}

type AllProcessedBidResponsesPayload struct {
	// todo: decide what payload to use within the hook invocation task
}
