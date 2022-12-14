package hookexecution

import (
	"context"
	"net/http"
	"sync"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/metrics"
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
}

type HookStageExecutor interface {
	StageExecutor
	SetAccount(account *config.Account)
	GetOutcomes() []StageOutcome
}

type hookExecutor struct {
	account        *config.Account
	accountID      string
	endpoint       string
	planBuilder    hooks.ExecutionPlanBuilder
	stageOutcomes  []StageOutcome
	moduleContexts *moduleContexts
	metricEngine   metrics.MetricsEngine
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
		endpoint:       e.endpoint,
		stage:          stageName,
		accountId:      e.accountID,
		account:        e.account,
		moduleContexts: e.moduleContexts,
	}

	payload := hookstage.EntrypointPayload{Request: req, Body: body}
	outcome, payload, contexts, rejectErr := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entityHttpRequest
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)

	return payload.Body, rejectErr
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

func (executor *EmptyHookExecutor) SetAccount(_ *config.Account) {}

func (executor *EmptyHookExecutor) GetOutcomes() []StageOutcome {
	return []StageOutcome{}
}

func (executor *EmptyHookExecutor) ExecuteEntrypointStage(_ *http.Request, body []byte) ([]byte, *RejectError) {
	return body, nil
}
