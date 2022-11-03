package hookexecution

import (
	"context"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"net/http"
)

func (executor HookExecutor) ExecuteEntrypointStage(req *http.Request, body []byte) (StageOutcome, []byte, *RejectError) {
	handler := func(
		ctx context.Context,
		moduleCtx *hookstage.ModuleContext,
		hook hookstage.Entrypoint,
		payload hookstage.EntrypointPayload,
	) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
		return hook.HandleEntrypointHook(ctx, moduleCtx, payload)
	}

	executor.InvocationCtx.Stage = hooks.StageEntrypoint
	payload := hookstage.EntrypointPayload{Request: req, Body: body}
	stageOutcome, payload, reject := executeStage(executor.InvocationCtx, executor.PlanBuilder.PlanForEntrypointStage(executor.Endpoint), payload, handler, executor.MetricEngine)
	stageOutcome.Entity = hookstage.EntityHttpRequest
	stageOutcome.Stage = hooks.StageEntrypoint

	return stageOutcome, payload.Body, reject
}
