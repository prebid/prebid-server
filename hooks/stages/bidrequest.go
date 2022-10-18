package stages

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type BidRequestHook interface {
	HandleBidRequestHook(
		context.Context,
		invocation.Context,
		BidRequestPayload,
	) (invocation.HookResult[BidRequestPayload], error)
}

type BidRequestPayload struct {
	BidRequest *openrtb2.BidRequest
}
