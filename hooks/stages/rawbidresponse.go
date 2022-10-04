package stages

import (
	"context"

	"github.com/prebid/prebid-server/hooks/invocation"
)

type RawBidResponseHook interface {
	Code() string
	Call(
		context.Context,
		invocation.Context,
		[]byte,
	) (invocation.HookResult[[]byte], error)
}
