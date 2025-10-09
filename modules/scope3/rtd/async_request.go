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

// NewAsyncRequest creates a new AsyncRequest object
// The request's context is used to create a cancellable context for the async request and, which spans multiple hooks
func (m *Module) NewAsyncRequest(req *http.Request) *AsyncRequest {
	ret := &AsyncRequest{
		Module: m,
	}
	ret.Context, ret.Cancel = context.WithCancel(req.Context())
	return ret
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
