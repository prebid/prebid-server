package stages

import (
	"context"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type AuctionResponseHook interface {
	Call(
		context.Context,
		invocation.InvocationContext,
		*openrtb2.BidResponse,
	) (invocation.HookResult[*openrtb2.BidResponse], error)
}
