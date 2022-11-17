package hookexecution

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/metrics"
)

type TimeoutError struct{}

func (e TimeoutError) Error() string {
	return fmt.Sprint("Hook execution timeout")
}

// FailureError indicates expected error occurred during hook execution on the module-side
// A moduleFailed metric will be sent in such case
type FailureError struct{}

func (e FailureError) Error() string {
	return fmt.Sprint("Hook execution failed")
}

type RejectError struct {
	Code   int
	Reason string // is it needed or code is enough?
}

func (e RejectError) Error() string {
	return fmt.Sprintf(`Module rejected stage, reason: "%s"`, e.Reason)
}

type hookHandler[H any, P any] func(
	context.Context,
	hookstage.ModuleContext,
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
) (StageOutcome, P, hookstage.StageModuleContext, *RejectError) {
	stageOutcome := StageOutcome{}
	stageOutcome.Groups = make([]GroupOutcome, 0, len(plan))
	stageModuleCtx := hookstage.StageModuleContext{}
	stageModuleCtx.GroupCtx = make([]hookstage.GroupModuleContext, 0, len(plan))

	for _, group := range plan {
		groupOutcome, newPayload, moduleContexts, reject := executeGroup(invocationCtx, group, payload, hookHandler, metricEngine)
		stageOutcome.ExecutionTimeMillis += groupOutcome.ExecutionTimeMillis
		stageOutcome.Groups = append(stageOutcome.Groups, groupOutcome)
		stageModuleCtx.GroupCtx = append(stageModuleCtx.GroupCtx, moduleContexts)
		if reject != nil {
			return stageOutcome, payload, stageModuleCtx, reject
		}

		payload = newPayload
	}

	return stageOutcome, payload, stageModuleCtx, nil
}

func executeGroup[H any, P any](
	invocationCtx *hookstage.InvocationContext,
	group hooks.Group[H],
	payload P,
	hookHandler hookHandler[H, P],
	metricEngine metrics.MetricsEngine,
) (GroupOutcome, P, hookstage.GroupModuleContext, *RejectError) {
	var wg sync.WaitGroup
	done := make(chan struct{})
	resp := make(chan HookResponse[P])

	for _, hook := range group.Hooks {
		mCtx := invocationCtx.GetModuleContext(hook.Module)
		wg.Add(1)
		go func(hw hooks.HookWrapper[H], moduleCtx hookstage.ModuleContext) {
			defer wg.Done()
			executeHook(moduleCtx, hw, payload, hookHandler, group.Timeout, resp, done)
		}(hook, mCtx)
	}

	go func() {
		wg.Wait()
		close(resp)
	}()

	hookResponses := collectHookResponses(resp, done)

	return processHookResponses(invocationCtx, hookResponses, payload, metricEngine)
}

func executeHook[H any, P any](
	moduleCtx hookstage.ModuleContext,
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
) (GroupOutcome, P, hookstage.GroupModuleContext, *RejectError) {
	groupOutcome := GroupOutcome{}
	groupOutcome.InvocationResults = make([]HookOutcome, 0, len(hookResponses))
	moduleContexts := make(map[string]hookstage.ModuleContext, len(hookResponses))

	for i, r := range hookResponses {
		labels := metrics.ModuleLabels{
			Module: r.HookID.ModuleCode,
			Stage:  invocationCtx.Stage,
			PubID:  invocationCtx.AccountId,
		}
		groupOutcome.InvocationResults = append(groupOutcome.InvocationResults, HookOutcome{
			Status:        StatusSuccess,
			HookID:        r.HookID,
			Message:       r.Result.Message,
			DebugMessages: r.Result.DebugMessages,
			AnalyticsTags: r.Result.AnalyticsTags,
			ExecutionTime: ExecutionTime{r.ExecutionTime},
		})
		moduleContexts[r.HookID.ModuleCode] = r.Result.ModuleContext

		metricEngine.RecordModuleCalled(labels)
		metricEngine.RecordModuleDuration(labels, r.ExecutionTime)
		if r.ExecutionTime > groupOutcome.ExecutionTimeMillis {
			groupOutcome.ExecutionTimeMillis = r.ExecutionTime
		}

		if r.Err != nil {
			groupOutcome.InvocationResults[i].Errors = append(groupOutcome.InvocationResults[i].Errors, r.Err.Error())
			switch r.Err.(type) {
			case TimeoutError:
				metricEngine.RecordModuleTimeout(labels)
				groupOutcome.InvocationResults[i].Status = StatusTimeout
			case FailureError:
				metricEngine.RecordModuleTimeout(labels)
				groupOutcome.InvocationResults[i].Status = StatusFailure
			default:
				metricEngine.RecordModuleExecutionError(labels)
				groupOutcome.InvocationResults[i].Status = StatusExecutionFailure
			}
			continue
		}

		if r.Result.Reject {
			reject := &RejectError{Code: r.Result.NbrCode, Reason: r.Result.Message}
			metricEngine.RecordModuleSuccessRejected(labels)
			groupOutcome.InvocationResults[i].Action = ActionReject
			groupOutcome.InvocationResults[i].Errors = append(groupOutcome.InvocationResults[i].Errors, reject.Error())
			return groupOutcome, payload, moduleContexts, reject
		}

		if r.Result.ChangeSet == nil || len(r.Result.ChangeSet.Mutations()) == 0 {
			metricEngine.RecordModuleSuccessNooped(labels)
			groupOutcome.InvocationResults[i].Action = ActionNOP
			continue
		}

		groupOutcome.InvocationResults[i].Action = ActionUpdate
		for _, mut := range r.Result.ChangeSet.Mutations() {
			p, err := mut.Apply(payload)
			if err != nil {
				groupOutcome.InvocationResults[i].Warnings = append(
					groupOutcome.InvocationResults[i].Warnings,
					fmt.Sprintf("failed applying hook mutation: %s", err),
				)
				continue
			}

			payload = p
			groupOutcome.InvocationResults[i].DebugMessages = append(
				groupOutcome.InvocationResults[i].DebugMessages,
				fmt.Sprintf(
					"Hook mutation successfully applied, affected key: %s, mutation type: %s",
					strings.Join(mut.Key(), "."),
					mut.Type(),
				),
			)
		}
		metricEngine.RecordModuleSuccessUpdated(labels)
	}

	return groupOutcome, payload, moduleContexts, nil
}
