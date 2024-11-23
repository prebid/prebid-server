package hookexecution

import (
	"context"
	"net/http"
	"sync"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/privacy"
)

const (
	EndpointAuction = "/openrtb2/auction"
	EndpointAmp     = "/openrtb2/amp"
)

// An entity specifies the type of object that was processed during the execution of the stage.
type entity string

const (
	entityHttpRequest              entity = "http-request"
	entityAuctionRequest           entity = "auction-request"
	entityAuctionResponse          entity = "auction_response"
	entityAllProcessedBidResponses entity = "all_processed_bid_responses"
)

type StageExecutor interface {
	ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError)
	ExecuteRawAuctionStage(body []byte) ([]byte, *RejectError)
	ExecuteProcessedAuctionStage(req *openrtb_ext.RequestWrapper) error
	ExecuteBidderRequestStage(req *openrtb_ext.RequestWrapper, bidder string) *RejectError
	ExecuteRawBidderResponseStage(response *adapters.BidderResponse, bidder string) *RejectError
	ExecuteAllProcessedBidResponsesStage(adapterBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)
	ExecuteAuctionResponseStage(response *openrtb2.BidResponse)
}

type HookStageExecutor interface {
	StageExecutor
	SetAccount(account *config.Account)
	SetActivityControl(activityControl privacy.ActivityControl)
	GetOutcomes() []StageOutcome
}

type hookExecutor struct {
	account         *config.Account
	accountID       string
	endpoint        string
	planBuilder     hooks.ExecutionPlanBuilder
	stageOutcomes   []StageOutcome
	moduleContexts  *moduleContexts
	metricEngine    metrics.MetricsEngine
	activityControl privacy.ActivityControl
	// Mutex needed for BidderRequest and RawBidderResponse Stages as they are run in several goroutines
	sync.Mutex
}

func NewHookExecutor(builder hooks.ExecutionPlanBuilder, endpoint string, me metrics.MetricsEngine) *hookExecutor {
	return &hookExecutor{
		endpoint:       endpoint,
		planBuilder:    builder,
		stageOutcomes:  []StageOutcome{},
		moduleContexts: &moduleContexts{ctxs: make(map[string]hookstage.ModuleContext)},
		metricEngine:   me,
	}
}

func (e *hookExecutor) SetAccount(account *config.Account) {
	if account == nil {
		return
	}

	e.account = account
	e.accountID = account.ID
}

func (e *hookExecutor) SetActivityControl(activityControl privacy.ActivityControl) {
	e.activityControl = activityControl
}

func (e *hookExecutor) GetOutcomes() []StageOutcome {
	return e.stageOutcomes
}

func (e *hookExecutor) ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError) {
	plan := e.planBuilder.PlanForEntrypointStage(e.endpoint)
	if len(plan) == 0 {
		return body, nil
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.Entrypoint,
		payload hookstage.EntrypointPayload,
	) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
		return hook.HandleEntrypointHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageEntrypoint.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.EntrypointPayload{Request: req, Body: body}

	outcome, payload, contexts, rejectErr := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entityHttpRequest
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)

	return payload.Body, rejectErr
}

func (e *hookExecutor) ExecuteRawAuctionStage(requestBody []byte) ([]byte, *RejectError) {
	plan := e.planBuilder.PlanForRawAuctionStage(e.endpoint, e.account)
	if len(plan) == 0 {
		return requestBody, nil
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.RawAuctionRequest,
		payload hookstage.RawAuctionRequestPayload,
	) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
		return hook.HandleRawAuctionHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageRawAuctionRequest.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.RawAuctionRequestPayload(requestBody)

	outcome, payload, contexts, reject := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entityAuctionRequest
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)

	return payload, reject
}

func (e *hookExecutor) ExecuteProcessedAuctionStage(request *openrtb_ext.RequestWrapper) error {
	plan := e.planBuilder.PlanForProcessedAuctionStage(e.endpoint, e.account)
	if len(plan) == 0 {
		return nil
	}

	if err := request.RebuildRequest(); err != nil {
		return err
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.ProcessedAuctionRequest,
		payload hookstage.ProcessedAuctionRequestPayload,
	) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
		return hook.HandleProcessedAuctionHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageProcessedAuctionRequest.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.ProcessedAuctionRequestPayload{Request: request}

	outcome, _, contexts, reject := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entityAuctionRequest
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)

	// remove type information if there is no rejection
	if reject == nil {
		return nil
	}

	return reject
}

func (e *hookExecutor) ExecuteBidderRequestStage(req *openrtb_ext.RequestWrapper, bidder string) *RejectError {
	plan := e.planBuilder.PlanForBidderRequestStage(e.endpoint, e.account)
	if len(plan) == 0 {
		return nil
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.BidderRequest,
		payload hookstage.BidderRequestPayload,
	) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
		return hook.HandleBidderRequestHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageBidderRequest.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.BidderRequestPayload{Request: req, Bidder: bidder}
	outcome, _, contexts, reject := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entity(bidder)
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)

	return reject
}

func (e *hookExecutor) ExecuteRawBidderResponseStage(response *adapters.BidderResponse, bidder string) *RejectError {
	plan := e.planBuilder.PlanForRawBidderResponseStage(e.endpoint, e.account)
	if len(plan) == 0 {
		return nil
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.RawBidderResponse,
		payload hookstage.RawBidderResponsePayload,
	) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
		return hook.HandleRawBidderResponseHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageRawBidderResponse.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.RawBidderResponsePayload{BidderResponse: response, Bidder: bidder}

	outcome, payload, contexts, reject := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	response = payload.BidderResponse
	outcome.Entity = entity(bidder)
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)

	return reject
}

func (e *hookExecutor) ExecuteAllProcessedBidResponsesStage(adapterBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid) {
	plan := e.planBuilder.PlanForAllProcessedBidResponsesStage(e.endpoint, e.account)
	if len(plan) == 0 {
		return
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.AllProcessedBidResponses,
		payload hookstage.AllProcessedBidResponsesPayload,
	) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
		return hook.HandleAllProcessedBidResponsesHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageAllProcessedBidResponses.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.AllProcessedBidResponsesPayload{Responses: adapterBids}
	outcome, _, contexts, _ := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entityAllProcessedBidResponses
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)
}

func (e *hookExecutor) ExecuteAuctionResponseStage(response *openrtb2.BidResponse) {
	plan := e.planBuilder.PlanForAuctionResponseStage(e.endpoint, e.account)
	if len(plan) == 0 {
		return
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.AuctionResponse,
		payload hookstage.AuctionResponsePayload,
	) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
		return hook.HandleAuctionResponseHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageAuctionResponse.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.AuctionResponsePayload{BidResponse: response}

	outcome, _, contexts, _ := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entityAuctionResponse
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)
}

func (e *hookExecutor) newContext(stage string) executionContext {
	return executionContext{
		account:         e.account,
		accountID:       e.accountID,
		endpoint:        e.endpoint,
		moduleContexts:  e.moduleContexts,
		stage:           stage,
		activityControl: e.activityControl,
	}
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
	defer e.Unlock()
	e.stageOutcomes = append(e.stageOutcomes, outcome)
}

type EmptyHookExecutor struct{}

func (executor EmptyHookExecutor) SetAccount(_ *config.Account) {}

func (executor EmptyHookExecutor) SetActivityControl(_ privacy.ActivityControl) {}

func (executor EmptyHookExecutor) GetOutcomes() []StageOutcome {
	return []StageOutcome{}
}

func (executor EmptyHookExecutor) ExecuteEntrypointStage(_ *http.Request, body []byte) ([]byte, *RejectError) {
	return body, nil
}

func (executor EmptyHookExecutor) ExecuteRawAuctionStage(body []byte) ([]byte, *RejectError) {
	return body, nil
}

func (executor EmptyHookExecutor) ExecuteProcessedAuctionStage(_ *openrtb_ext.RequestWrapper) error {
	return nil
}

func (executor EmptyHookExecutor) ExecuteBidderRequestStage(_ *openrtb_ext.RequestWrapper, bidder string) *RejectError {
	return nil
}

func (executor EmptyHookExecutor) ExecuteRawBidderResponseStage(_ *adapters.BidderResponse, _ string) *RejectError {
	return nil
}

func (executor EmptyHookExecutor) ExecuteAllProcessedBidResponsesStage(_ map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid) {
}

func (executor EmptyHookExecutor) ExecuteAuctionResponseStage(_ *openrtb2.BidResponse) {}
