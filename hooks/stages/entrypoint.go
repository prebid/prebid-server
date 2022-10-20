package stages

import (
	"context"
	"github.com/prebid/prebid-server/hooks/invocation"
	"net/http"
)

type EntrypointHook interface {
	Call(
		ctx context.Context,
		iCtx *invocation.ModuleContext,
		p EntrypointPayload,
		debugMode bool,
	) (invocation.HookResult[EntrypointPayload], error)
}

type EntrypointPayload struct {
	Request *http.Request
	Body    []byte
}
