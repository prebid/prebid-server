package hookstage

import (
	"context"
	"net/http"
)

type Entrypoint interface {
	HandleEntrypointHook(
		ctx context.Context,
		iCtx *ModuleContext,
		p EntrypointPayload,
	) (HookResult[EntrypointPayload], error)
}

type EntrypointPayload struct {
	Request *http.Request
	Body    []byte
}
