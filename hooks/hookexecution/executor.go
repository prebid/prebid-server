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
	entityExitpoint                entity = "exitpoint"
)

type StageExecutor interface {
	ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError)
	ExecuteRawAuctionStage(body []byte) ([]byte, *RejectError)
	ExecuteProcessedAuctionStage(req *openrtb_ext.RequestWrapper) error
	ExecuteBidderRequestStage(req *openrtb_ext.RequestWrapper, bidder string) *RejectError
	ExecuteRawBidderResponseStage(response *adapters.BidderResponse, bidder string) *RejectError
	ExecuteAllProcessedBidResponsesStage(adapterBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)
	ExecuteAuctionResponseStage(response *openrtb2.BidResponse)
	ExecuteExitpointStage(response any, w http.ResponseWriter) any
}

type HookStageExecutor interface {
	StageExecutor
	SetAccount(account *config.Account)
	SetActivityControl(activityControl privacy.ActivityControl)
	GetOutcomes() []StageOutcome
	GetABTestTargetingKeywords() map[string]string
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
	abTests         *ABTests
	// Mutex needed for BidderRequest and RawBidderResponse Stages as they are run in several goroutines
	sync.Mutex
}

func NewHookExecutor(builder hooks.ExecutionPlanBuilder, endpoint string, me metrics.MetricsEngine, abTests *ABTests) *hookExecutor {
	return &hookExecutor{
		endpoint:       endpoint,
		planBuilder:    builder,
		stageOutcomes:  []StageOutcome{},
		moduleContexts: &moduleContexts{ctxs: make(map[string]hookstage.ModuleContext)},
		metricEngine:   me,
		abTests:        abTests,
	}
}

func (e *hookExecutor) SetAccount(account *config.Account) {
	if account == nil {
		return
	}

	e.account = account
	e.accountID = account.ID
	e.abTests.SetAccount(account)
}

func (e *hookExecutor) SetActivityControl(activityControl privacy.ActivityControl) {
	e.activityControl = activityControl
}

func (e *hookExecutor) GetOutcomes() []StageOutcome {
	return e.stageOutcomes
}

func (e *hookExecutor) ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError) {
	e.abTests.SetAccountID(body)
	stagePlan := e.planBuilder.PlanForEntrypointStage(e.endpoint)
	plan := applyABTestPlan(e.abTests, stagePlan)
	if len(plan) == 0 {
		outcome := StageOutcome{
			Entity: entityHttpRequest,
			Stage:  hooks.StageEntrypoint.String(),
		}
		e.abTests.WriteOutcome(&outcome)
		if len(outcome.Groups) > 0 {
			e.pushStageOutcome(outcome)
		}

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
	e.abTests.WriteOutcome(&outcome)
	e.pushStageOutcome(outcome)

	return payload.Body, rejectErr
}

func (e *hookExecutor) ExecuteRawAuctionStage(requestBody []byte) ([]byte, *RejectError) {
	stagePlan := e.planBuilder.PlanForRawAuctionStage(e.endpoint, e.account)
	plan := applyABTestPlan(e.abTests, stagePlan)
	if len(plan) == 0 {
		outcome := StageOutcome{
			Entity: entityHttpRequest,
			Stage:  hooks.StageRawAuctionRequest.String(),
		}
		e.abTests.WriteOutcome(&outcome)
		if len(outcome.Groups) > 0 {
			e.pushStageOutcome(outcome)
		}

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
	e.abTests.WriteOutcome(&outcome)
	e.pushStageOutcome(outcome)

	return payload, reject
}

func (e *hookExecutor) ExecuteProcessedAuctionStage(request *openrtb_ext.RequestWrapper) error {
	stagePlan := e.planBuilder.PlanForProcessedAuctionStage(e.endpoint, e.account)
	plan := applyABTestPlan(e.abTests, stagePlan)
	if len(plan) == 0 {
		outcome := StageOutcome{
			Entity: entityHttpRequest,
			Stage:  hooks.StageEntrypoint.String(),
		}
		e.abTests.WriteOutcome(&outcome)
		if len(outcome.Groups) > 0 {
			e.pushStageOutcome(outcome)
		}

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
	e.abTests.WriteOutcome(&outcome)
	e.pushStageOutcome(outcome)

	// remove type information if there is no rejection
	if reject == nil {
		return nil
	}

	return reject
}

func (e *hookExecutor) ExecuteBidderRequestStage(req *openrtb_ext.RequestWrapper, bidder string) *RejectError {
	stagePlan := e.planBuilder.PlanForBidderRequestStage(e.endpoint, e.account)
	plan := applyABTestPlan(e.abTests, stagePlan)
	if len(plan) == 0 {
		outcome := StageOutcome{
			Entity: entityHttpRequest,
			Stage:  hooks.StageEntrypoint.String(),
		}
		e.abTests.WriteOutcome(&outcome)
		if len(outcome.Groups) > 0 {
			e.pushStageOutcome(outcome)
		}

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
	e.abTests.WriteOutcome(&outcome)
	e.pushStageOutcome(outcome)

	return reject
}

func (e *hookExecutor) ExecuteRawBidderResponseStage(response *adapters.BidderResponse, bidder string) *RejectError {
	stagePlan := e.planBuilder.PlanForRawBidderResponseStage(e.endpoint, e.account)
	plan := applyABTestPlan(e.abTests, stagePlan)
	if len(plan) == 0 {
		outcome := StageOutcome{
			Entity: entityHttpRequest,
			Stage:  hooks.StageEntrypoint.String(),
		}
		e.abTests.WriteOutcome(&outcome)
		if len(outcome.Groups) > 0 {
			e.pushStageOutcome(outcome)
		}

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
	e.abTests.WriteOutcome(&outcome)
	e.pushStageOutcome(outcome)

	return reject
}

func (e *hookExecutor) ExecuteAllProcessedBidResponsesStage(adapterBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid) {
	stagePlan := e.planBuilder.PlanForAllProcessedBidResponsesStage(e.endpoint, e.account)
	plan := applyABTestPlan(e.abTests, stagePlan)
	if len(plan) == 0 {
		outcome := StageOutcome{
			Entity: entityHttpRequest,
			Stage:  hooks.StageEntrypoint.String(),
		}
		e.abTests.WriteOutcome(&outcome)
		if len(outcome.Groups) > 0 {
			e.pushStageOutcome(outcome)
		}
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
	e.abTests.WriteOutcome(&outcome)
	e.pushStageOutcome(outcome)
}

func (e *hookExecutor) ExecuteAuctionResponseStage(response *openrtb2.BidResponse) {
	stagePlan := e.planBuilder.PlanForAuctionResponseStage(e.endpoint, e.account)
	plan := applyABTestPlan(e.abTests, stagePlan)
	if len(plan) == 0 {
		outcome := StageOutcome{
			Entity: entityHttpRequest,
			Stage:  hooks.StageEntrypoint.String(),
		}
		e.abTests.WriteOutcome(&outcome)
		if len(outcome.Groups) > 0 {
			e.pushStageOutcome(outcome)
		}
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
	e.abTests.WriteOutcome(&outcome)
	e.pushStageOutcome(outcome)
}

func (e *hookExecutor) ExecuteExitpointStage(response any, w http.ResponseWriter) any {
	plan := e.planBuilder.PlanForExitpointStage(e.endpoint, e.account)
	if len(plan) == 0 {
		return response
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.Exitpoint,
		payload hookstage.ExitpointPaylaod,
	) (hookstage.HookResult[hookstage.ExitpointPaylaod], error) {
		return hook.HandleExitpointHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageExitpoint.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.ExitpointPaylaod{W: w, Response: response}

	outcome, payload, context, _ := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entityExitpoint
	outcome.Stage = stageName

	e.saveModuleContexts(context)
	e.pushStageOutcome(outcome)

	return payload.Response
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

func (e *hookExecutor) GetABTestTargetingKeywords() map[string]string {
	return e.abTests.GetTargetingKeywords()
}

func applyABTestPlan[T any](ab *ABTests, plan hooks.Plan[T]) hooks.Plan[T] {
	var p hooks.Plan[T]
	for _, group := range plan {
		var g hooks.Group[T]
		g.Timeout = group.Timeout
		for _, hook := range group.Hooks {
			if ab.Run(hook.Module) {
				g.Hooks = append(g.Hooks, hook)
			}
		}
		if len(g.Hooks) > 0 {
			p = append(p, g)
		}
	}

	return p
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

func (executor EmptyHookExecutor) ExecuteAuctionResponseStage(_ *openrtb2.BidResponse) {
}

func (executor EmptyHookExecutor) ExecuteExitpointStage(response any, _ http.ResponseWriter) any {
	return response
}

func (executor EmptyHookExecutor) GetABTestTargetingKeywords() map[string]string {
	return map[string]string{}
}
