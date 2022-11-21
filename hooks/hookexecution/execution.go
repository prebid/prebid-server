package hookexecution

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
)

type hookResponse[T any] struct {
	Err           error
	ExecutionTime time.Duration
	HookID        HookID
	Result        hookstage.HookResult[T]
}

type hookHandler[H any, P any] func(
	context.Context,
	hookstage.ModuleInvocationContext,
	H,
	P,
) (hookstage.HookResult[P], error)

func executeStage[H any, P any](
	executionCtx executionContext,
	plan hooks.Plan[H],
	payload P,
	hookHandler hookHandler[H, P],
) (StageOutcome, P, stageModuleContext, *RejectError) {
	stageOutcome := StageOutcome{}
	stageOutcome.Groups = make([]GroupOutcome, 0, len(plan))
	stageModuleCtx := stageModuleContext{}
	stageModuleCtx.groupCtx = make([]groupModuleContext, 0, len(plan))

	for _, group := range plan {
		groupOutcome, newPayload, moduleContexts, reject := executeGroup(executionCtx, group, payload, hookHandler)
		stageOutcome.ExecutionTimeMillis += groupOutcome.ExecutionTimeMillis
		stageOutcome.Groups = append(stageOutcome.Groups, groupOutcome)
		stageModuleCtx.groupCtx = append(stageModuleCtx.groupCtx, moduleContexts)
		if reject != nil {
			return stageOutcome, payload, stageModuleCtx, reject
		}

		payload = newPayload
	}

	return stageOutcome, payload, stageModuleCtx, nil
}

func executeGroup[H any, P any](
	executionCtx executionContext,
	group hooks.Group[H],
	payload P,
	hookHandler hookHandler[H, P],
) (GroupOutcome, P, groupModuleContext, *RejectError) {
	var wg sync.WaitGroup
	done := make(chan struct{})
	resp := make(chan hookResponse[P])

	for _, hook := range group.Hooks {
		mCtx := executionCtx.getModuleContext(hook.Module)
		wg.Add(1)
		go func(hw hooks.HookWrapper[H], moduleCtx hookstage.ModuleInvocationContext) {
			defer wg.Done()
			executeHook(moduleCtx, hw, payload, hookHandler, group.Timeout, resp, done)
		}(hook, mCtx)
	}

	go func() {
		wg.Wait()
		close(resp)
	}()

	hookResponses := collectHookResponses(resp, done)

	return processHookResponses(executionCtx, hookResponses, payload)
}

func executeHook[H any, P any](
	moduleCtx hookstage.ModuleInvocationContext,
	hw hooks.HookWrapper[H],
	payload P,
	hookHandler hookHandler[H, P],
	timeout time.Duration,
	resp chan<- hookResponse[P],
	done <-chan struct{},
) {
	hookRespCh := make(chan hookResponse[P], 1)

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
			hookRespCh <- hookResponse[P]{
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
			resp <- hookResponse[P]{
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
	resp <-chan hookResponse[P],
	done chan<- struct{},
) []hookResponse[P] {
	hookResponses := make([]hookResponse[P], 0)
	for r := range resp {
		hookResponses = append(hookResponses, r)
		if r.Result.Reject {
			close(done)
			break
		}
	}

	return hookResponses
}

func processHookResponses[P any](
	executionCtx executionContext,
	hookResponses []hookResponse[P],
	payload P,
) (GroupOutcome, P, groupModuleContext, *RejectError) {
	stage := hooks.Stage(executionCtx.stage)
	groupOutcome := GroupOutcome{}
	groupOutcome.InvocationResults = make([]HookOutcome, 0, len(hookResponses))
	groupModuleCtx := make(groupModuleContext, len(hookResponses))

	for i, r := range hookResponses {
		groupOutcome.InvocationResults = append(groupOutcome.InvocationResults, HookOutcome{
			Status:        StatusSuccess,
			HookID:        r.HookID,
			Message:       r.Result.Message,
			DebugMessages: r.Result.DebugMessages,
			AnalyticsTags: r.Result.AnalyticsTags,
			ExecutionTime: ExecutionTime{r.ExecutionTime},
		})
		groupModuleCtx[r.HookID.ModuleCode] = r.Result.ModuleContext

		if r.ExecutionTime > groupOutcome.ExecutionTimeMillis {
			groupOutcome.ExecutionTimeMillis = r.ExecutionTime
		}

		if r.Err != nil {
			groupOutcome.InvocationResults[i].Errors = append(groupOutcome.InvocationResults[i].Errors, r.Err.Error())
			switch r.Err.(type) {
			case TimeoutError:
				groupOutcome.InvocationResults[i].Status = StatusTimeout
			case FailureError:
				groupOutcome.InvocationResults[i].Status = StatusFailure
			default:
				groupOutcome.InvocationResults[i].Status = StatusExecutionFailure
			}
			continue
		}

		if r.Result.Reject {
			if !stage.IsRejectable() {
				groupOutcome.InvocationResults[i].Status = StatusExecutionFailure
				groupOutcome.InvocationResults[i].Errors = append(
					groupOutcome.InvocationResults[i].Errors,
					fmt.Sprintf(
						"Module (name: %s, hook code: %s) tried to reject request on the %s stage that does not support rejection",
						r.HookID.ModuleCode,
						r.HookID.HookCode,
						executionCtx.stage,
					),
				)
				continue
			}

			reject := &RejectError{r.Result.NbrCode, r.HookID, executionCtx.stage}
			groupOutcome.InvocationResults[i].Action = ActionReject
			groupOutcome.InvocationResults[i].Errors = append(groupOutcome.InvocationResults[i].Errors, reject.Error())
			return groupOutcome, payload, groupModuleCtx, reject
		}

		if r.Result.ChangeSet == nil || len(r.Result.ChangeSet.Mutations()) == 0 {
			groupOutcome.InvocationResults[i].Action = ActionNoAction
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
	}

	return groupOutcome, payload, groupModuleCtx, nil
}
