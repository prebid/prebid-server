package hookexecution

import (
	"context"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
)

func ExecuteEntrypointStage(executor HookExecutor) (StageOutcome, []byte, *RejectError) {
	handler := func(
		ctx context.Context,
		moduleCtx *hookstage.ModuleContext,
		hook hookstage.Entrypoint,
		payload hookstage.EntrypointPayload,
	) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
		return hook.HandleEntrypointHook(ctx, moduleCtx, payload)
	}

	payload := hookstage.EntrypointPayload{Request: executor.Req, Body: executor.Body}
	stageOutcome, payload, reject := executeStage(executor.InvocationCtx, executor.PlanBuilder.PlanForEntrypointStage(executor.Endpoint), payload, handler)
	stageOutcome.Entity = hookstage.EntityHttpRequest
	stageOutcome.Stage = hooks.StageEntrypoint

	return stageOutcome, payload.Body, reject
}
