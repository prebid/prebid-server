package hookstage

import (
	"context"
)

type RawAuction interface {
	HandleRawAuctionHook(
		context.Context,
		InvocationContext,
		RawAuctionPayload,
	) (HookResult[RawAuctionPayload], error)
}

type RawAuctionPayload []byte
