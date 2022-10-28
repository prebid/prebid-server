package hookexecution

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/invocation"
)

type TimeoutError struct{}

func (e TimeoutError) Error() string {
	return fmt.Sprint("Hook execution timeout")
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
	*invocation.ModuleContext,
	H,
	P,
) (invocation.HookResult[P], error)

func executeStage[H any, P any](
	invocationCtx *invocation.InvocationContext,
	plan hooks.Plan[H],
	payload P,
	hookHandler hookHandler[H, P],
) (invocation.StageResult[P], P, *RejectError) {
	var stageResult invocation.StageResult[P]
	stageResult.GroupsResults = make([][]invocation.HookResult[P], 0, len(plan))

	for _, group := range plan {
		groupResult, newPayload, err := executeGroup(invocationCtx, group, payload, hookHandler)
		if err != nil {
			return stageResult, payload, err
		}

		payload = newPayload
		stageResult.GroupsResults = append(stageResult.GroupsResults, groupResult)
	}

	return stageResult, payload, nil
}

func executeGroup[H any, P any](
	invocationCtx *invocation.InvocationContext,
	group hooks.Group[H],
	payload P,
	hookHandler hookHandler[H, P],
) ([]invocation.HookResult[P], P, *RejectError) {
	var wg sync.WaitGroup
	done := make(chan struct{})
	resp := make(chan invocation.HookResponse[P])

	for _, hook := range group.Hooks {
		wg.Add(1)
		go func(hw hooks.HookWrapper[H], moduleCtx *invocation.ModuleContext) {
			defer wg.Done()
			executeHook(moduleCtx, hw, payload, hookHandler, group.Timeout, resp, done)
		}(hook, invocationCtx.ModuleContextFor(hook.Module))
	}

	go func() {
		wg.Wait()
		close(resp)
	}()

	hookResponses, err := collectHookResponses(resp, done)
	if err != nil {
		return nil, payload, err
	}

	groupResult, payload := processHookResponses(hookResponses, payload)

	return groupResult, payload, nil
}

func executeHook[H any, P any](
	moduleCtx *invocation.ModuleContext,
	hw hooks.HookWrapper[H],
	payload P,
	hookHandler hookHandler[H, P],
	timeout time.Duration,
	resp chan<- invocation.HookResponse[P],
	done <-chan struct{},
) {
	hookRespCh := make(chan invocation.HookResponse[P], 1)

	select {
	case <-done:
		return
	default:
		startTime := time.Now()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			result, err := hookHandler(ctx, moduleCtx, hw.Hook, payload)
			hookRespCh <- invocation.HookResponse[P]{
				Result: result,
				Err:    err,
			}
		}()

		select {
		case res := <-hookRespCh:
			res.Result.ModuleCode = hw.Module
			res.Result.ExecutionTime = time.Since(startTime)
			resp <- res
		case <-time.After(timeout):
			resp <- invocation.HookResponse[P]{
				Result: invocation.HookResult[P]{ModuleCode: hw.Module},
				Err:    TimeoutError{},
			}
		case <-done:
			return
		}
	}
}

func collectHookResponses[P any](
	resp <-chan invocation.HookResponse[P],
	done chan<- struct{},
) ([]invocation.HookResponse[P], *RejectError) {
	hookResponses := make([]invocation.HookResponse[P], 0)
	for r := range resp {
		if r.Result.Reject {
			close(done)
			// todo: send metric
			reject := &RejectError{Code: r.Result.NbrCode, Reason: r.Result.Message}
			r.Result.Errors = append(r.Result.Errors, reject.Error())
			return nil, reject
		}

		hookResponses = append(hookResponses, r)
	}

	return hookResponses, nil
}

func processHookResponses[P any](
	hookResponses []invocation.HookResponse[P],
	payload P,
) ([]invocation.HookResult[P], P) {
	groupResult := make([]invocation.HookResult[P], 0, len(hookResponses))
	for i, r := range hookResponses {
		groupResult = append(groupResult, r.Result)

		if r.Err != nil {
			groupResult[i].Errors = append(groupResult[i].Errors, r.Err.Error())
			// todo: send metric
			continue
		}

		if r.Result.ChangeSet == nil || len(r.Result.ChangeSet.Mutations()) == 0 {
			// todo: send hook metrics (NOP metric)
			continue
		}

		for _, mut := range r.Result.ChangeSet.Mutations() {
			p, err := mut.Apply(payload)
			if err != nil {
				groupResult[i].Warnings = append(groupResult[i].Warnings, fmt.Sprintf("failed applying hook mutation: %s", err))
				continue
			}
			payload = p
		}
	}

	return groupResult, payload
}
