package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type BidRequest interface {
	HandleBidRequestHook(
		context.Context,
		invocation.InvocationContext,
		BidRequestPayload,
	) (invocation.HookResult[BidRequestPayload], error)
}

type BidRequestPayload struct {
	BidRequest *openrtb2.BidRequest
}
