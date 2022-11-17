package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/adapters"
)

type RawBidderResponse interface {
	HandleRawBidderResponseHook(
		context.Context,
		ModuleContext,
		RawBidderResponsePayload,
	) (HookResult[RawBidderResponsePayload], error)
}

type RawBidderResponsePayload struct {
	Bids []*adapters.TypedBid
}
