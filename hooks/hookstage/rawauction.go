package hookstage

import (
	"context"

	"github.com/prebid/prebid-server/hooks/invocation"
)

type RawAuction interface {
	HandleRawAuctionHook(
		context.Context,
		invocation.InvocationContext,
		RawAuctionPayload,
	) (invocation.HookResult[RawAuctionPayload], error)
}

type RawAuctionPayload []byte
