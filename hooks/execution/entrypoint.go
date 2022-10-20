package execution

import (
	"context"
	"fmt"
	"github.com/prebid/prebid-server/hooks/hep"
	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/hooks/stages"
	"net/http"
	"sync"
	"time"
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

func ExecuteEntrypointStage(
	invocationCtx *invocation.InvocationContext,
	plan hep.Plan[stages.EntrypointHook],
	req *http.Request,
	body []byte,
) (invocation.StageResult[stages.EntrypointPayload], []byte, error) {
	var wg sync.WaitGroup
	var stageResult invocation.StageResult[stages.EntrypointPayload]

	payload := stages.EntrypointPayload{Request: req, Body: body}
	done := make(chan struct{})

	for _, groups := range plan {
		groupResults := make([]invocation.HookResult[stages.EntrypointPayload], 0)
		resp := make(chan invocation.HookResponse[stages.EntrypointPayload])

		for _, moduleHook := range groups.Hooks {
			hookRespCh := make(chan invocation.HookResponse[stages.EntrypointPayload], 1)
			wg.Add(1)
			go func(hook stages.EntrypointHook, moduleCode string) {
				defer wg.Done()

				select {
				case <-done:
					return
				default:
					go asyncHookCall(invocationCtx.ModuleContextFor(moduleCode), hook, payload, groups.Timeout, invocationCtx.DebugEnabled, hookRespCh)
					select {
					case res := <-hookRespCh:
						res.Result.ModuleCode = moduleCode
						resp <- res
					case <-time.After(groups.Timeout):
						resp <- invocation.HookResponse[stages.EntrypointPayload]{
							Result: invocation.HookResult[stages.EntrypointPayload]{ModuleCode: moduleCode},
							Err:    TimeoutError{},
						}
					}
				}
			}(moduleHook.Hook, moduleHook.Module)
		}

		go func() {
			wg.Wait()
			close(resp)
		}()

		for r := range resp {
			if r.Result.Reject {
				close(done)
				// todo: send metric
				return stageResult, body, RejectError{Code: r.Result.NbrCode, Reason: r.Result.Message}
			}

			if r.Err != nil {
				// todo: process error?
				// todo: send metric
			}

			groupResults = append(groupResults, r.Result)
		}

		for _, r := range groupResults {
			for _, mut := range r.Mutations {
				p, err := mut.Apply(payload)
				if err != nil {
					r.Errors = append(r.Errors, fmt.Sprintf("failed applying hook mutation: %s", err))
					continue
				}
				payload.Body = p.Body
			}

			// todo: send hook metrics (NOP metric if r.Result.Mutations empty)
		}

		stageResult.GroupsResults = append(stageResult.GroupsResults, groupResults)
	}

	// todo: send all metrics

	return stageResult, payload.Body, nil
}

func asyncHookCall(
	moduleCtx *invocation.ModuleContext,
	hook stages.EntrypointHook,
	pld stages.EntrypointPayload,
	timeout time.Duration,
	debugMode bool,
	hrc chan<- invocation.HookResponse[stages.EntrypointPayload],
) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result, err := hook.Call(ctx, moduleCtx, pld, debugMode)

	hrc <- invocation.HookResponse[stages.EntrypointPayload]{
		Result: result,
		Err:    err,
	}
}
