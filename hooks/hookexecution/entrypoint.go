package hookexecution

import (
	"context"
	"net/http"

	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
)

func ExecuteEntrypointStage(
	invocationCtx *hookstage.InvocationContext,
	plan hooks.Plan[hookstage.Entrypoint],
	req *http.Request,
	body []byte,
) (StageOutcome, []byte, *RejectError) {
	handler := func(
		ctx context.Context,
		moduleCtx *hookstage.ModuleContext,
		hook hookstage.Entrypoint,
		payload hookstage.EntrypointPayload,
	) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
		return hook.HandleEntrypointHook(ctx, moduleCtx, payload)
	}

	payload := hookstage.EntrypointPayload{Request: req, Body: body}
	stageOutcome, payload, reject := executeStage(invocationCtx, plan, payload, handler)
	stageOutcome.Entity = hookstage.EntityHttpRequest
	stageOutcome.Stage = hooks.StageEntrypoint

	return stageOutcome, payload.Body, reject
}
