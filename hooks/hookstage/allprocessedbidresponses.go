package hookstage

import (
	"context"
)

// AllProcessedBidResponses hooks are invoked over a list of all
// processed responses received from bidders before a winner is chosen.
//
// At this stage, account config is available,
// so it can be configured at the account-level execution plan,
// the account-level module config is passed to hooks.
//
// Rejection has no effect and is completely ignored at this stage.
type AllProcessedBidResponses interface {
	HandleAllProcessedBidResponsesHook(
		context.Context,
		ModuleInvocationContext,
		AllProcessedBidResponsesPayload,
	) (HookResult[AllProcessedBidResponsesPayload], error)
}

// AllProcessedBidResponsesPayload consists of a list of all
// processed responses received from bidders.
// Hooks are allowed to modify payload object and discard bids using mutations.
type AllProcessedBidResponsesPayload struct {
	// todo: decide what payload to use within the hook invocation task
}
