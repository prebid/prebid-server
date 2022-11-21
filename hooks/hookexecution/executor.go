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

type entity string

const (
	entityHttpRequest              entity = "http-request"
	entityAuctionRequest           entity = "auction-request"
	entityAuctionResponse          entity = "auction-response"
	entityAllProcessedBidResponses entity = "all-processed-bid-responses"
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
	account        *config.Account
	accountId      string
	endpoint       string
	planBuilder    hooks.ExecutionPlanBuilder
	stageOutcomes  []StageOutcome
	moduleContexts *moduleContexts
	// Mutex needed for BidderRequest and RawBidderResponse Stages as they are run in several goroutines
	sync.Mutex
}

func NewHookExecutor(builder hooks.ExecutionPlanBuilder, endpoint string) *hookExecutor {
	return &hookExecutor{
		endpoint:       endpoint,
		planBuilder:    builder,
		stageOutcomes:  []StageOutcome{},
		moduleContexts: &moduleContexts{ctxs: make(map[string]hookstage.ModuleContext)},
	}
}

func (e *hookExecutor) SetAccount(account *config.Account) {
	if account == nil {
		return
	}

	e.account = account
	e.accountId = account.ID
}

func (e *hookExecutor) GetOutcomes() []StageOutcome {
	return e.stageOutcomes
}

func (e *hookExecutor) ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError) {
	plan := e.planBuilder.PlanForEntrypointStage(e.endpoint)
	if len(plan) == 0 {
		return body, nil
	}

	stageName := hooks.StageEntrypoint.String()
	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.Entrypoint,
		payload hookstage.EntrypointPayload,
	) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
		return hook.HandleEntrypointHook(ctx, moduleCtx, payload)
	}

	executionCtx := executionContext{
		e.endpoint,
		stageName,
		e.accountId,
		e.account,
		e.moduleContexts,
	}

	payload := hookstage.EntrypointPayload{Request: req, Body: body}
	stageOutcome, payload, stageModuleContexts, reject := executeStage(executionCtx, plan, payload, handler)
	stageOutcome.Entity = entityHttpRequest
	stageOutcome.Stage = stageName

	e.saveModuleContexts(stageModuleContexts)
	e.pushStageOutcome(stageOutcome)

	return payload.Body, reject
}

func (e *hookExecutor) ExecuteRawAuctionStage(body []byte) ([]byte, *RejectError) {
	//TODO: implement
	return nil, nil
}

func (e *hookExecutor) ExecuteProcessedAuctionStage(req *openrtb2.BidRequest) *RejectError {
	//TODO: implement
	return nil
}

func (e *hookExecutor) ExecuteBidderRequestStage(req *openrtb2.BidRequest, bidder string) *RejectError {
	//TODO: implement
	return nil
}

func (e *hookExecutor) ExecuteRawBidderResponseStage(response *adapters.BidderResponse, bidder string) *RejectError {
	//TODO: implement
	return nil
}

func (e *hookExecutor) ExecuteAllProcessedBidResponsesStage(responses []*adapters.BidderResponse) {
	//TODO: implement
}

func (e *hookExecutor) ExecuteAuctionResponseStage(response *openrtb2.BidResponse) {
	//TODO: implement
}

func (e *hookExecutor) saveModuleContexts(ctxs stageModuleContext) {
	for _, moduleCtxs := range ctxs.groupCtx {
		for moduleName, moduleCtx := range moduleCtxs {
			e.moduleContexts.put(moduleName, moduleCtx)
		}
	}
}

func (e *hookExecutor) pushStageOutcome(outcome StageOutcome) {
	e.Lock()
	e.stageOutcomes = append(e.stageOutcomes, outcome)
	defer e.Unlock()
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
