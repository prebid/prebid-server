package stages

import (
	"context"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type AuctionResponseHook interface {
	HandleAuctionResponseHook(
		context.Context,
		invocation.Context,
		*openrtb2.BidResponse,
	) (invocation.HookResult[*openrtb2.BidResponse], error)
}
