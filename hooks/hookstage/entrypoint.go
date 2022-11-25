package hookstage

import (
	"context"
	"net/http"
)

// Entrypoint hooks are invoked at the very beginning of request processing.
//
// At this stage, account config is not yet available,
// so it can only be defined as part of the host-level execution plan,
// the account-level module config is not available.
//
// Rejection results in sending an empty BidResponse
// with the NBR code indicating the rejection reason.
type Entrypoint interface {
	HandleEntrypointHook(
		context.Context,
		ModuleInvocationContext,
		EntrypointPayload,
	) (HookResult[EntrypointPayload], error)
}

// EntrypointPayload consists of an HTTP request and a raw body of the openrtb2.BidRequest.
// For "/openrtb2/amp" endpoint the body is nil.
// Hooks are allowed to modify this data using mutations.
type EntrypointPayload struct {
	Request *http.Request
	Body    []byte
}
