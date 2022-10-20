package hookstage

import (
	"context"
	"net/http"

	"github.com/prebid/prebid-server/hooks/invocation"
)

type Entrypoint interface {
	HandleEntrypointHook(
		context.Context,
		invocation.Context,
		EntrypointPayload,
	) (invocation.HookResult[EntrypointPayload], error)
}

type EntrypointPayload struct {
	Request *http.Request
	Body    []byte
}
