package hookstage

import (
	"context"
	"net/http"
)

type Entrypoint interface {
	HandleEntrypointHook(
		context.Context,
		InvocationContext,
		EntrypointPayload,
	) (HookResult[EntrypointPayload], error)
}

type EntrypointPayload struct {
	Request *http.Request
	Body    []byte
}
