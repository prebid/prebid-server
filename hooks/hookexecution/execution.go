package hookexecution

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/metrics"
)

const (
	Auction_endpoint = "/openrtb2/auction"
	Amp_endpoint     = "/openrtb2/amp"
)

type HookExecutor struct {
	InvocationCtx *hookstage.InvocationContext
	Endpoint      string
	PlanBuilder   hooks.ExecutionPlanBuilder
	MetricEngine  metrics.MetricsEngine
	Req           *http.Request
	Body          []byte
}

type TimeoutError struct{}

func (e TimeoutError) Error() string {
	return fmt.Sprint("Hook execution timeout")
}

// FailureError indicates expected error occurred during hook execution on the module-side
type FailureError struct{}

func (e FailureError) Error() string {
	return fmt.Sprint("Hook execution failed")
}

type RejectError struct {
	Code   int
	Reason string // is it needed or code is enough?
}

func (e RejectError) Error() string {
	return fmt.Sprintf("Module rejected stage, reason: `%s`", e.Reason)
}

type hookHandler[H any, P any] func(
	context.Context,
	*hookstage.ModuleContext,
	H,
	P,
) (hookstage.HookResult[P], error)

type HookResponse[T any] struct {
	Err           error
	ExecutionTime time.Duration
	HookID        HookID
	Result        hookstage.HookResult[T]
}

func executeStage[H any, P any](
	invocationCtx *hookstage.InvocationContext,
	plan hooks.Plan[H],
	payload P,
	hookHandler hookHandler[H, P],
	metricEngine metrics.MetricsEngine,
) (StageOutcome, P, *RejectError) {
	stageOutcome := StageOutcome{}
	stageOutcome.Groups = make([]GroupOutcome, 0, len(plan))

	for _, group := range plan {
		groupOutcome, newPayload, reject := executeGroup(invocationCtx, group, payload, hookHandler, metricEngine)
		stageOutcome.ExecutionTimeMillis += groupOutcome.ExecutionTimeMillis
		stageOutcome.Groups = append(stageOutcome.Groups, groupOutcome)
		if reject != nil {
			return stageOutcome, payload, reject
		}

		payload = newPayload
	}

	return stageOutcome, payload, nil
}

func executeGroup[H any, P any](
	invocationCtx *hookstage.InvocationContext,
	group hooks.Group[H],
	payload P,
	hookHandler hookHandler[H, P],
	metricEngine metrics.MetricsEngine,
) (GroupOutcome, P, *RejectError) {
	var wg sync.WaitGroup
	done := make(chan struct{})
	resp := make(chan HookResponse[P])

	for _, hook := range group.Hooks {
		wg.Add(1)
		go func(hw hooks.HookWrapper[H], moduleCtx *hookstage.ModuleContext) {
			defer wg.Done()
			executeHook(moduleCtx, hw, payload, hookHandler, group.Timeout, resp, done)
		}(hook, invocationCtx.ModuleContextFor(hook.Module))
	}

	go func() {
		wg.Wait()
		close(resp)
	}()

	hookResponses := collectHookResponses(resp, done)

	return processHookResponses(invocationCtx, hookResponses, payload, metricEngine)
}

func executeHook[H any, P any](
	moduleCtx *hookstage.ModuleContext,
	hw hooks.HookWrapper[H],
	payload P,
	hookHandler hookHandler[H, P],
	timeout time.Duration,
	resp chan<- HookResponse[P],
	done <-chan struct{},
) {
	hookRespCh := make(chan HookResponse[P], 1)

	select {
	case <-done:
		return
	default:
		startTime := time.Now()
		hookId := HookID{hw.Module, hw.Code}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			result, err := hookHandler(ctx, moduleCtx, hw.Hook, payload)
			hookRespCh <- HookResponse[P]{
				Result: result,
				Err:    err,
			}
		}()

		select {
		case res := <-hookRespCh:
			res.HookID = hookId
			res.ExecutionTime = time.Since(startTime)
			resp <- res
		case <-time.After(timeout):
			resp <- HookResponse[P]{
				Err:           TimeoutError{},
				ExecutionTime: time.Since(startTime),
				HookID:        hookId,
				Result:        hookstage.HookResult[P]{},
			}
		case <-done:
			return
		}
	}
}

func collectHookResponses[P any](
	resp <-chan HookResponse[P],
	done chan<- struct{},
) []HookResponse[P] {
	hookResponses := make([]HookResponse[P], 0)
	for r := range resp {
		if r.Result.Reject {
			close(done)
			return []HookResponse[P]{r}
		}

		hookResponses = append(hookResponses, r)
	}

	return hookResponses
}

func processHookResponses[P any](
	invocationCtx *hookstage.InvocationContext,
	hookResponses []HookResponse[P],
	payload P,
	metricEngine metrics.MetricsEngine,
) (GroupOutcome, P, *RejectError) {
	groupOutcome := GroupOutcome{}
	groupOutcome.InvocationResults = make([]*HookOutcome, 0, len(hookResponses))

	for _, r := range hookResponses {
		labels := metrics.ModuleLabels{
			Module: r.HookID.ModuleCode,
			Stage:  invocationCtx.Stage,
			PubID:  invocationCtx.AccountId,
		}
		hookOutcome := &HookOutcome{
			Status:        StatusSuccess,
			HookID:        r.HookID,
			Message:       r.Result.Message,
			DebugMessages: r.Result.DebugMessages,
			AnalyticsTags: r.Result.AnalyticsTags,
			ExecutionTime: ExecutionTime{r.ExecutionTime},
		}

		metricEngine.RecordModuleCalled(labels)
		metricEngine.RecordModuleDuration(labels, r.ExecutionTime)
		groupOutcome.InvocationResults = append(groupOutcome.InvocationResults, hookOutcome)
		if r.ExecutionTime > groupOutcome.ExecutionTimeMillis {
			groupOutcome.ExecutionTimeMillis = r.ExecutionTime
		}

		if r.Err != nil {
			hookOutcome.Errors = append(hookOutcome.Errors, r.Err.Error())
			switch r.Err.(type) {
			case TimeoutError:
				metricEngine.RecordModuleTimeout(labels)
				hookOutcome.Status = StatusTimeout
			case FailureError:
				hookOutcome.Status = StatusFailure
			default:
				metricEngine.RecordModuleExecutionError(labels)
				hookOutcome.Status = StatusExecutionFailure
			}
			// todo: send metric
			continue
		}

		if r.Result.Reject {
			reject := &RejectError{Code: r.Result.NbrCode, Reason: r.Result.Message}
			metricEngine.RecordModuleSuccessRejected(labels)
			hookOutcome.Action = ActionReject
			hookOutcome.Errors = append(hookOutcome.Errors, reject.Error())
			// todo: send metric
			return groupOutcome, payload, reject
		}

		if r.Result.ChangeSet == nil || len(r.Result.ChangeSet.Mutations()) == 0 {
			metricEngine.RecordModuleSuccessNooped(labels)
			hookOutcome.Action = ActionNOP
			continue
		}

		hookOutcome.Action = ActionUpdate
		for _, mut := range r.Result.ChangeSet.Mutations() {
			p, err := mut.Apply(payload)
			if err != nil {
				hookOutcome.Warnings = append(hookOutcome.Warnings, fmt.Sprintf("failed applying hook mutation: %s", err))
				continue
			}

			payload = p
			hookOutcome.DebugMessages = append(hookOutcome.DebugMessages, fmt.Sprintf(
				"Hook mutation successfully applied, affected key: %s, mutation type: %s",
				strings.Join(mut.Key(), "."),
				mut.Type(),
			))
		}
		metricEngine.RecordModuleSuccessUpdated(labels)
	}

	return groupOutcome, payload, nil
}
