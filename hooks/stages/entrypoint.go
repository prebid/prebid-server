package stages

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/prebid/prebid-server/hooks/invocation"
)

type EntrypointHook interface {
	Code() string
	Call(
		context.Context,
		invocation.Context,
		ImmutableEntrypointPayload,
	) (invocation.HookResult[MutableEntrypointPayload], error)
}

type ImmutableEntrypointPayload struct {
	request *http.Request
	params  url.Values
	Body    []byte
}

func (w ImmutableEntrypointPayload) Header(key string) string {
	return w.request.Header.Get(key)
}

func (w ImmutableEntrypointPayload) Param(key string) string {
	return w.params.Get(key)
}

type MutableEntrypointPayload struct {
	Header http.Header
	Params url.Values
	Body   []byte
}

// todo: remove this variable, ExecuteEntrypointHook func must be implemented as a method
// on the HookExecutionPlan type, which will contain all the necessary hooks for actual host/account level
var entrypointHooks []EntrypointHook

func ExecuteEntrypointHook(ctx context.Context, invocationCtx invocation.Context, req *http.Request, body []byte) (
	[]byte,
	error,
) {
	var wg sync.WaitGroup

	// immutable payload - used to execute hooks
	iPayload := ImmutableEntrypointPayload{
		request: req,
		params:  req.URL.Query(),
		Body:    body,
	}

	// mutable payload - used to apply hook mutations
	mPayload := MutableEntrypointPayload{
		Header: req.Header.Clone(),
		Params: req.URL.Query(),
		Body:   body,
	}

	done := make(chan struct{})
	resp := make(chan invocation.HookResponse[MutableEntrypointPayload])

	for _, h := range entrypointHooks {
		wg.Add(1)
		go func(hook EntrypointHook) {
			defer wg.Done()

			select {
			case <-done:
			default:
				result, err := hook.Call(ctx, invocationCtx, iPayload)
				resp <- invocation.HookResponse[MutableEntrypointPayload]{
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
			// todo: process error
			continue
		}

		switch r.Result.Action {
		case invocation.Nop:
			continue
		case invocation.Reject:
			close(done)
			// todo: reject request
			return body, nil
		case invocation.Update:
			for _, mut := range r.Result.Mutations {
				p, err := mut.Apply(mPayload)
				if err != nil {
					// todo: handle mutation applying error
					fmt.Printf("failed applying mutation: %s", err)
					continue
				}
				mPayload = p
			}
		}

		// todo: send hook metrics
	}

	req.Header = mPayload.Header
	req.URL.RawQuery = mPayload.Params.Encode()

	// todo: send all metrics

	return mPayload.Body, nil
}
