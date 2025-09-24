package scope3

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
)

type (
	AsyncRequest struct {
		*Module

		// For managing the lifecycle of the request
		// Context is used to pass to the request. This should be the context of the original request
		Context context.Context
		// Cancel can be called to cancel the request
		Cancel context.CancelFunc
		// DoneChannel will be closed when the request is done. When nil, no request was made
		Done chan struct{}

		// Response
		Segments []string
		Err      error
	}
)

// getAsyncRequest gets an AsyncRequest from the pool and initializes it
// The request's context is used to create a cancellable context for the async request, which spans multiple hooks
func (m *Module) getAsyncRequest(req *http.Request) *AsyncRequest {
	ret := m.asyncRequestPool.Get().(*AsyncRequest)
	ret.Module = m
	ret.Context, ret.Cancel = context.WithCancel(req.Context())
	return ret
}

// putAsyncRequest resets an AsyncRequest and puts it back into the pool
func (m *Module) putAsyncRequest(ar *AsyncRequest) {
	ar.reset()
	m.asyncRequestPool.Put(ar)
}

// reset resets the AsyncRequest so it can be reused
func (ar *AsyncRequest) reset() {
	ar.Module = nil
	ar.Context = nil
	ar.Cancel = nil
	ar.Done = nil
	ar.Segments = nil
	ar.Err = nil
}

// fetchScope3SegmentsAsync starts a goroutine to fetch Scope3 segments and immediately returns
// The Done channel will be closed when the request is done
// If the Done channel is nil, no request was made
func (ar *AsyncRequest) fetchScope3SegmentsAsync(request *openrtb2.BidRequest) {
	ar.Done = make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ar.Err = fmt.Errorf("panic in async request: %v", r)
			}
			close(ar.Done)
		}()
		ar.Segments, ar.Err = ar.Module.fetchScope3Segments(ar.Context, request)
	}()
}
