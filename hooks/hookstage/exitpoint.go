package hookstage

import (
	"context"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
)

// ExitPoint hooks are invoked only for "/openrtb2/auction"
// just before openrtb2.BidResponse is serialized and passed to
// http.ResponseWriter.
//
// At this stage, only a single hook can be declared,
// cause otherwise the return result becomes unpredictable.
//
// The code can return modified openrtb2.BidResponse or any
// other Go type declared by module and mapped to
// openrtb2.BidResponse
//
// The response must be pointer.
type ExitPoint interface {
	HandleExitPointHook(
		context.Context,
		ModuleInvocationContext,
		ExitPointPayload,
	) (HookResult[ExitPointPayload], error)
}

// ExitPointPayload represents input and output arguments
// of HandleExitPointHook method.
type ExitPointPayload struct {
	Account     *config.Account
	BidRequest  *openrtb2.BidRequest
	BidResponse *openrtb2.BidResponse
	HTTPHeaders http.Header
	Response    any
}
