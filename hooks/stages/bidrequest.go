package stages

import (
	"context"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type BidRequestHook interface {
	Call(
		context.Context,
		invocation.InvocationContext,
		BidRequestPayload,
	) (invocation.HookResult[BidRequestPayload], error)
}

type BidRequestPayload struct {
	BidRequest *openrtb2.BidRequest
}
