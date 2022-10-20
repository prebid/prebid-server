package hookexecution

import (
	"context"
	"fmt"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/hooks/invocation"
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
	plan hooks.Plan[hookstage.Entrypoint],
	req *http.Request,
	body []byte,
) (invocation.StageResult[hookstage.EntrypointPayload], []byte, error) {
	var wg sync.WaitGroup
	var stageResult invocation.StageResult[hookstage.EntrypointPayload]

	payload := hookstage.EntrypointPayload{Request: req, Body: body}
	done := make(chan struct{})

	for _, groups := range plan {
		groupResults := make([]invocation.HookResult[hookstage.EntrypointPayload], 0)
		resp := make(chan invocation.HookResponse[hookstage.EntrypointPayload])

		for _, moduleHook := range groups.Hooks {
			hookRespCh := make(chan invocation.HookResponse[hookstage.EntrypointPayload], 1)
			wg.Add(1)
			go func(hw hooks.HookWrapper[hookstage.Entrypoint]) {
				defer wg.Done()

				select {
				case <-done:
					return
				default:
					moduleCtx := invocationCtx.ModuleContextFor(hw.Module)
					if moduleCtx.Config == nil {
						moduleCtx.Config = hw.Config
					}

					go asyncHookCall(moduleCtx, hw.Hook, payload, groups.Timeout, invocationCtx.DebugEnabled, hookRespCh)
					select {
					case res := <-hookRespCh:
						res.Result.ModuleCode = hw.Module
						resp <- res
					case <-time.After(groups.Timeout):
						resp <- invocation.HookResponse[hookstage.EntrypointPayload]{
							Result: invocation.HookResult[hookstage.EntrypointPayload]{ModuleCode: hw.Module},
							Err:    TimeoutError{},
						}
					}
				}
			}(moduleHook)
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
	hook hookstage.Entrypoint,
	pld hookstage.EntrypointPayload,
	timeout time.Duration,
	debugMode bool,
	hrc chan<- invocation.HookResponse[hookstage.EntrypointPayload],
) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result, err := hook.HandleEntrypointHook(ctx, moduleCtx, pld, debugMode)

	hrc <- invocation.HookResponse[hookstage.EntrypointPayload]{
		Result: result,
		Err:    err,
	}
}
