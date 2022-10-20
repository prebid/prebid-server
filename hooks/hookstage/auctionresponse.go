package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type AuctionResponse interface {
	HandleAuctionResponseHook(
		context.Context,
		invocation.InvocationContext,
		*openrtb2.BidResponse,
	) (invocation.HookResult[*openrtb2.BidResponse], error)
}
