package stages

import (
	"context"

	"github.com/prebid/prebid-server/hooks/invocation"
)

type RawAuctionHook interface {
	Call(
		context.Context,
		invocation.InvocationContext,
		BidRequest,
	) (invocation.HookResult[BidRequest], error)
}

type BidRequest []byte
