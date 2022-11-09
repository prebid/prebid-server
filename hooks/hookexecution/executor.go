package hookexecution

import (
	"context"
	"net/http"

	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
)

const (
	EndpointAuction = "/openrtb2/auction"
	EndpointAmp     = "/openrtb2/amp"
)

type Executor interface {
	ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError)
}

type HookExecutor struct {
	InvocationCtx *hookstage.InvocationContext
	Endpoint      string
	PlanBuilder   hooks.ExecutionPlanBuilder
	stageOutcomes []StageOutcome
}

func (executor *HookExecutor) GetOutcomes() []StageOutcome {
	return executor.stageOutcomes
}

func (executor *HookExecutor) ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError) {
	plan := executor.PlanBuilder.PlanForEntrypointStage(executor.Endpoint)
	if len(plan) == 0 {
		return body, nil
	}

	handler := func(
		ctx context.Context,
		moduleCtx *hookstage.ModuleContext,
		hook hookstage.Entrypoint,
		payload hookstage.EntrypointPayload,
	) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
		return hook.HandleEntrypointHook(ctx, moduleCtx, payload)
	}

	payload := hookstage.EntrypointPayload{Request: req, Body: body}
	stageOutcome, payload, reject := executeStage(executor.InvocationCtx, plan, payload, handler)
	stageOutcome.Entity = hookstage.EntityHttpRequest
	stageOutcome.Stage = hooks.StageEntrypoint

	executor.stageOutcomes = append(executor.stageOutcomes, stageOutcome)

	return payload.Body, reject
}
