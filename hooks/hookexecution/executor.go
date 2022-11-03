package hookexecution

import (
	"context"
	"net/http"

	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
)

const (
	Auction_endpoint = "/openrtb2/auction"
	Amp_endpoint     = "/openrtb2/amp"
)

type HookExecutor struct {
	InvocationCtx *hookstage.InvocationContext
	Endpoint      string
	PlanBuilder   hooks.ExecutionPlanBuilder
}

func (executor HookExecutor) ExecuteEntrypointStage(req *http.Request, body []byte) (StageOutcome, []byte, *RejectError) {
	handler := func(
		ctx context.Context,
		moduleCtx *hookstage.ModuleContext,
		hook hookstage.Entrypoint,
		payload hookstage.EntrypointPayload,
	) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
		return hook.HandleEntrypointHook(ctx, moduleCtx, payload)
	}

	payload := hookstage.EntrypointPayload{Request: req, Body: body}
	stageOutcome, payload, reject := executeStage(executor.InvocationCtx, executor.PlanBuilder.PlanForEntrypointStage(executor.Endpoint), payload, handler)
	stageOutcome.Entity = hookstage.EntityHttpRequest
	stageOutcome.Stage = hooks.StageEntrypoint

	return stageOutcome, payload.Body, reject
}
