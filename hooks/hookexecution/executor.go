package hookexecution

import (
	"context"
	"net/http"
	"sync"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
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
	ExecuteRawBidderResponseStage(response *adapters.BidderResponse, bidder string) *RejectError
	ExecuteAllProcessedBidResponsesStage(responses []*adapters.BidderResponse) //TODO: check that responses is the necessary param
	ExecuteAuctionResponseStage(response *openrtb2.BidResponse)
}

type HookStageExecutor interface {
	StageExecutor
	SetAccount(account *config.Account)
	GetOutcomes() []StageOutcome
}

type hookExecutor struct {
	invocationCtx *hookstage.InvocationContext
	endpoint      string
	planBuilder   hooks.ExecutionPlanBuilder
	stageOutcomes []StageOutcome
	// Mutex needed for BidderRequest and RawBidderResponse Stages as they are run in several goroutines
	mu sync.Mutex
}

func NewHookExecutor(builder hooks.ExecutionPlanBuilder, endpoint string) *hookExecutor {
	return &hookExecutor{
		invocationCtx: &hookstage.InvocationContext{},
		endpoint:      endpoint,
		planBuilder:   builder,
		stageOutcomes: []StageOutcome{},
	}
}

func (executor *hookExecutor) SetAccount(account *config.Account) {
	executor.invocationCtx.Account = account
	executor.invocationCtx.AccountId = account.ID
}

func (executor *hookExecutor) GetOutcomes() []StageOutcome {
	return executor.stageOutcomes
}

func (executor *hookExecutor) ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError) {
	plan := executor.planBuilder.PlanForEntrypointStage(executor.endpoint)
	if len(plan) == 0 {
		return body, nil
	}

	stageName := hooks.StageEntrypoint.String()
	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleContext,
		hook hookstage.Entrypoint,
		payload hookstage.EntrypointPayload,
	) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
		return hook.HandleEntrypointHook(ctx, moduleCtx, payload)
	}

	executor.invocationCtx.Stage = stageName

	payload := hookstage.EntrypointPayload{Request: req, Body: body}
	stageOutcome, payload, stageModuleContexts, reject := executeStage(executor.invocationCtx, plan, payload, handler)
	stageOutcome.Entity = hookstage.EntityHttpRequest
	stageOutcome.Stage = stageName

	executor.saveModuleContexts(stageModuleContexts)
	executor.stageOutcomes = append(executor.stageOutcomes, stageOutcome)

	return payload.Body, reject
}

func (executor *hookExecutor) ExecuteRawAuctionStage(body []byte) ([]byte, *RejectError) {
	//TODO: implement
	return nil, nil
}

func (executor *hookExecutor) ExecuteProcessedAuctionStage(req *openrtb2.BidRequest) *RejectError {
	//TODO: implement
	return nil
}

func (executor *hookExecutor) ExecuteBidderRequestStage(req *openrtb2.BidRequest, bidder string) *RejectError {
	//TODO: implement
	return nil
}

func (executor *hookExecutor) ExecuteRawBidderResponseStage(response *adapters.BidderResponse, bidder string) *RejectError {
	//TODO: implement
	return nil
}

func (executor *hookExecutor) ExecuteAllProcessedBidResponsesStage(responses []*adapters.BidderResponse) {
	//TODO: implement
}

func (executor *hookExecutor) ExecuteAuctionResponseStage(response *openrtb2.BidResponse) {
	//TODO: implement
}

func (executor *hookExecutor) saveModuleContexts(ctxs hookstage.StageModuleContext) {
	for _, mcs := range ctxs.GroupCtx {
		for k, mc := range mcs {
			if mc.Ctx != nil {
				executor.invocationCtx.SetModuleContext(k, mc)
			}
		}
	}
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

func (executor *EmptyHookExecutor) ExecuteBidderRequestStage(_ *openrtb2.BidRequest, _ string) *RejectError {
	return nil
}

func (executor *EmptyHookExecutor) ExecuteRawBidderResponseStage(_ *adapters.BidderResponse, _ string) *RejectError {
	return nil
}

func (executor *EmptyHookExecutor) ExecuteAllProcessedBidResponsesStage(_ []*adapters.BidderResponse) {
}

func (executor *EmptyHookExecutor) ExecuteAuctionResponseStage(_ *openrtb2.BidResponse) {}
