package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
)

type AuctionResponse interface {
	HandleAuctionResponseHook(
		context.Context,
		InvocationContext,
		*openrtb2.BidResponse,
	) (HookResult[*openrtb2.BidResponse], error)
}
