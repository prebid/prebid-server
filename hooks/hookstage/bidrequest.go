package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
)

type BidRequest interface {
	HandleBidRequestHook(
		context.Context,
		InvocationContext,
		BidRequestPayload,
	) (HookResult[BidRequestPayload], error)
}

type BidRequestPayload struct {
	BidRequest *openrtb2.BidRequest
}
