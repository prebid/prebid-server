package hookexecution

import (
	"context"
	"net/http"

	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/hooks/invocation"
)

func ExecuteEntrypointStage(
	invocationCtx *invocation.InvocationContext,
	plan hooks.Plan[hookstage.Entrypoint],
	req *http.Request,
	body []byte,
) (invocation.StageResult[hookstage.EntrypointPayload], []byte, *RejectError) {
	handler := func(
		ctx context.Context,
		moduleCtx *invocation.ModuleContext,
		hook hookstage.Entrypoint,
		payload hookstage.EntrypointPayload,
	) (invocation.HookResult[hookstage.EntrypointPayload], error) {
		return hook.HandleEntrypointHook(ctx, moduleCtx, payload)
	}

	payload := hookstage.EntrypointPayload{Request: req, Body: body}
	stageResult, payload, err := executeStage(invocationCtx, plan, payload, handler)

	return stageResult, payload.Body, err
}
