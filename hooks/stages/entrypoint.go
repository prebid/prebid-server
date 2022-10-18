package stages

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/prebid/prebid-server/hooks/invocation"
)

type EntrypointHook interface {
	HandleEntrypointHook(
		context.Context,
		invocation.Context,
		EntrypointPayload,
	) (invocation.HookResult[EntrypointPayload], error)
}

type EntrypointPayload struct {
	Request *http.Request
	Body    []byte
}

type HookResponse[T any] struct {
	Result invocation.HookResult[T]
	Err    error
}

// todo: remove this variable, ExecuteEntrypointHook func must be implemented as a method
// on the HookExecutionPlan type, which will contain all the necessary hooks for actual host/account level
var entrypointHooks []EntrypointHook

func ExecuteEntrypointStage(
	ctx context.Context,
	invocationCtx invocation.Context,
	req *http.Request,
	body []byte,
) ([]byte, error) {
	var wg sync.WaitGroup

	payload := EntrypointPayload{Request: req, Body: body}
	done := make(chan struct{})
	resp := make(chan HookResponse[EntrypointPayload])
	results := make([]invocation.HookResult[EntrypointPayload], 0, len(entrypointHooks))

	for _, h := range entrypointHooks {
		wg.Add(1)
		go func(hook EntrypointHook) {
			defer wg.Done()

			select {
			case <-done:
			default:
				result, err := hook.HandleEntrypointHook(ctx, invocationCtx, payload)
				resp <- HookResponse[EntrypointPayload]{
					Result: result,
					Err:    err,
				}
			}
		}(h)
	}

	go func() {
		wg.Wait()
		close(resp)
	}()

	for r := range resp {
		if r.Err != nil {
			// todo: process error, send metric
			continue
		}

		if r.Result.Reject {
			close(done)
			// todo: reject request, send metric
			return body, nil
		}

		results = append(results, r.Result)
	}

	// wait till all hooks executed, so we can safely apply their mutations to avoid race conditions
	for _, r := range results {
		for _, mut := range r.Mutations {
			p, err := mut.Apply(payload)
			if err != nil {
				// todo: handle mutation applying error
				fmt.Printf("failed applying mutation: %s", err)
				continue
			}
			payload = p
		}

		// todo: send hook metrics (NOP metric if r.Result.Mutations empty)
	}

	// todo: send all metrics

	return payload.Body, nil
}
