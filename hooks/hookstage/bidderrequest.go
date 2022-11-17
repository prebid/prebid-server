package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
)

type BidderRequest interface {
	HandleBidderRequestHook(
		context.Context,
		ModuleContext,
		BidderRequestPayload,
	) (HookResult[BidderRequestPayload], error)
}

type BidderRequestPayload struct {
	BidRequest *openrtb2.BidRequest
}
