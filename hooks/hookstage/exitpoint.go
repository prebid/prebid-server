package hookstage

import (
	"context"
	"net/http"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// Exitpoint hooks are invoked just before the response is returned.
// These hooks can fully modify the response based on the module’s implementation.
// They are only triggered if there are no 4XX or 5XX errors during request processing.
//
// At this stage, the account configuration is available,
// allowing hooks to be controlled through the account-level execution plan.
// The account-level module configuration is also passed to the hooks.
//
// Any rejection at this stage is ignored and has no effect.
type Exitpoint interface {
	HandleExitpointHook(
		context.Context,
		ModuleInvocationContext,
		ExitpointPayload,
	) (HookResult[ExitpointPayload], error)
}

// ExitpointPayload contains a response of any type and a ResponseWriter.
// The response is typically of type *openrtb2.BidResponse and can be modified
// based on the module’s implementation.
// Modules can also add custom headers depending on their response type.
// When Body is set by a module, the caller writes the raw bytes directly
// instead of JSON-encoding Response, allowing modules to control serialization.
type ExitpointPayload struct {
	Request  *openrtb_ext.RequestWrapper
	Response any
	W        http.ResponseWriter
	Body     []byte
}
