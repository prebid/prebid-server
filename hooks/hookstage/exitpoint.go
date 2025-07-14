package hookstage

import (
	"context"
	"net/http"
)

// Exitpoint hooks are invoked when response about to return.
// This hook can completely alter the response according to the implementation by module.
// The hooks are invoked when there is no 4XX/5XX erros while request processing.
//
// At this stage, account config is available,
// so it can be configured at the account-level execution plan,
// the account-level module config is passed to hooks.
//
// Rejection has no effect and is completely ignored at this stage.
type Exitpoint interface {
	HandleExitpointHook(
		context.Context,
		ModuleInvocationContext,
		ExitpointPaylaod,
	) (HookResult[ExitpointPaylaod], error)
}

// ExitpointPaylaod consists of Response of any type and a response writter.
// The type which passed to Response is *openrtb2.BidResponse which can be modified
// according to module implementation.
// Module can add headers of its own according to its response type.
type ExitpointPaylaod struct {
	Response any
	W        http.ResponseWriter
}
