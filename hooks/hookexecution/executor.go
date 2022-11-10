package hookexecution

import (
	"context"
	"net/http"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/metrics"
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
	MetricEngine  metrics.MetricsEngine
	stageOutcomes []StageOutcome
}

func (executor *HookExecutor) GetOutcomes() []StageOutcome {
	return executor.stageOutcomes
}

func (executor *HookExecutor) ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError) {
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

	executor.stageOutcomes = append(executor.stageOutcomes, stageOutcome)

	return payload.Body, reject
}

func (executor *HookExecutor) ExecuteRawAuctionStage(requestBody []byte, account *config.Account) ([]byte, *RejectError) {
	handler := func(
		ctx context.Context,
		moduleCtx *hookstage.ModuleContext,
		hook hookstage.RawAuction,
		payload hookstage.RawAuctionPayload,
	) (hookstage.HookResult[hookstage.RawAuctionPayload], error) {
		return hook.HandleRawAuctionHook(ctx, moduleCtx, payload)
	}

	executor.InvocationCtx.Stage = hooks.StageRawAuction
	payload := hookstage.RawAuctionPayload(requestBody)
	stageOutcome, payload, reject := executeStage(executor.InvocationCtx, executor.PlanBuilder.PlanForRawAuctionStage(executor.Endpoint, account), payload, handler, executor.MetricEngine)
	stageOutcome.Entity = hookstage.EntityAuctionRequest
	stageOutcome.Stage = hooks.StageRawAuction

	executor.stageOutcomes = append(executor.stageOutcomes, stageOutcome)

	return payload, reject
}

func (executor *HookExecutor) ExecuteProcessedAuctionStage(request *openrtb2.BidRequest, account *config.Account) *RejectError {
	handler := func(
		ctx context.Context,
		moduleCtx *hookstage.ModuleContext,
		hook hookstage.ProcessedAuction,
		payload hookstage.ProcessedAuctionPayload,
	) (hookstage.HookResult[hookstage.ProcessedAuctionPayload], error) {
		return hook.HandleProcessedAuctionHook(ctx, moduleCtx, payload)
	}

	executor.InvocationCtx.Stage = hooks.StageProcessedAuction
	payload := hookstage.ProcessedAuctionPayload{BidRequest: request}
	stageOutcome, _, reject := executeStage(executor.InvocationCtx, executor.PlanBuilder.PlanForProcessedAuctionStage(executor.Endpoint, account), payload, handler, executor.MetricEngine)
	stageOutcome.Entity = hookstage.EntityAuctionRequest
	stageOutcome.Stage = hooks.StageProcessedAuction

	executor.stageOutcomes = append(executor.stageOutcomes, stageOutcome)

	return reject
}
