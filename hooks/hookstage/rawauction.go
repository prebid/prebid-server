package hookstage

import (
	"context"
)

type RawAuction interface {
	HandleRawAuctionHook(
		context.Context,
		*ModuleContext,
		RawAuctionPayload,
	) (HookResult[RawAuctionPayload], error)
}

type RawAuctionPayload []byte
