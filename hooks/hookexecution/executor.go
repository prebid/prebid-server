package hookexecution

import (
	"context"
	"net/http"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/metrics"
)

const (
	EndpointAuction = "/openrtb2/auction"
	EndpointAmp     = "/openrtb2/amp"
)

type StageExecutor interface {
	ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError)
	ExecuteRawAuctionStage(body []byte) ([]byte, *RejectError)
	ExecuteProcessedAuctionStage(req *openrtb2.BidRequest) *RejectError
	ExecuteBidderRequestStage(req *openrtb2.BidRequest, bidder string) *RejectError
	ExecuteRawBidderResponseStage(response *adapters.BidderResponse) *RejectError
	ExecuteAllProcessedBidResponsesStage(responses []*adapters.BidderResponse) //TODO: check that responses is the necessary param
	ExecuteAuctionResponseStage(response *openrtb2.BidResponse)
}

type HookStageExecutor interface {
	StageExecutor
	SetAccount(account *config.Account)
	GetOutcomes() []StageOutcome
}

type HookExecutor struct {
	InvocationCtx *hookstage.InvocationContext
	Endpoint      string
	PlanBuilder   hooks.ExecutionPlanBuilder
	MetricEngine  metrics.MetricsEngine
	stageOutcomes []StageOutcome
}

func (executor *HookExecutor) SetAccount(account *config.Account) {
	executor.InvocationCtx.Account = account
	executor.InvocationCtx.AccountId = account.ID
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

	executor.InvocationCtx.Stage = hooks.StageEntrypoint
	payload := hookstage.EntrypointPayload{Request: req, Body: body}
	stageOutcome, payload, reject := executeStage(executor.InvocationCtx, plan, payload, handler, executor.MetricEngine)
	stageOutcome.Entity = hookstage.EntityHttpRequest
	stageOutcome.Stage = hooks.StageEntrypoint

	executor.stageOutcomes = append(executor.stageOutcomes, stageOutcome)

	return payload.Body, reject
}

func (executor *HookExecutor) ExecuteRawAuctionStage(requestBody []byte) ([]byte, *RejectError) {
	plan := executor.PlanBuilder.PlanForRawAuctionStage(executor.Endpoint, executor.InvocationCtx.Account)
	if len(plan) == 0 {
		return requestBody, nil
	}

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
	stageOutcome, payload, reject := executeStage(executor.InvocationCtx, plan, payload, handler, executor.MetricEngine)
	stageOutcome.Entity = hookstage.EntityAuctionRequest
	stageOutcome.Stage = hooks.StageRawAuction

	executor.stageOutcomes = append(executor.stageOutcomes, stageOutcome)

	return payload, reject
}

func (executor *HookExecutor) ExecuteProcessedAuctionStage(req *openrtb2.BidRequest) *RejectError {
	//TODO: implement
	return nil
}

func (executor *HookExecutor) ExecuteBidderRequestStage(req *openrtb2.BidRequest, bidder string) *RejectError {
	plan := executor.PlanBuilder.PlanForBidderRequestStage(executor.Endpoint, executor.InvocationCtx.Account)
	if len(plan) == 0 {
		return nil
	}

	handler := func(
		ctx context.Context,
		moduleCtx *hookstage.ModuleContext,
		hook hookstage.BidderRequest,
		payload hookstage.BidderRequestPayload,
	) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
		return hook.HandleBidderRequestHook(ctx, moduleCtx, payload)
	}

	executor.InvocationCtx.Stage = hooks.StageBidderRequest
	payload := hookstage.BidderRequestPayload{BidRequest: req}
	stageOutcome, _, reject := executeStage(executor.InvocationCtx, plan, payload, handler, executor.MetricEngine)
	stageOutcome.Entity = hookstage.Entity(bidder)
	stageOutcome.Stage = hooks.StageBidderRequest

	executor.stageOutcomes = append(executor.stageOutcomes, stageOutcome)

	return reject
}

func (executor *HookExecutor) ExecuteRawBidderResponseStage(response *adapters.BidderResponse) *RejectError {
	//TODO: implement
	return nil
}

func (executor *HookExecutor) ExecuteAllProcessedBidResponsesStage(responses []*adapters.BidderResponse) {
	//TODO: implement
}

func (executor *HookExecutor) ExecuteAuctionResponseStage(response *openrtb2.BidResponse) {
	//TODO: implement
}

type EmptyHookExecutor struct{}

func (executor *EmptyHookExecutor) SetAccount(_ *config.Account) {}

func (executor *EmptyHookExecutor) GetOutcomes() []StageOutcome {
	return []StageOutcome{}
}

func (executor *EmptyHookExecutor) ExecuteEntrypointStage(_ *http.Request, body []byte) ([]byte, *RejectError) {
	return body, nil
}

func (executor *EmptyHookExecutor) ExecuteRawAuctionStage(body []byte) ([]byte, *RejectError) {
	return body, nil
}

func (executor *EmptyHookExecutor) ExecuteProcessedAuctionStage(_ *openrtb2.BidRequest) *RejectError {
	return nil
}

func (executor *EmptyHookExecutor) ExecuteBidderRequestStage(_ *openrtb2.BidRequest, bidder string) *RejectError {
	return nil
}

func (executor *EmptyHookExecutor) ExecuteRawBidderResponseStage(_ *adapters.BidderResponse) *RejectError {
	return nil
}

func (executor *EmptyHookExecutor) ExecuteAllProcessedBidResponsesStage(_ []*adapters.BidderResponse) {
}
func (executor *EmptyHookExecutor) ExecuteAuctionResponseStage(_ *openrtb2.BidResponse) {}
