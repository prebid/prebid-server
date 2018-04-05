package file_fetcher

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/net/context/ctxhttp"
)

// NewFetcher returns a Fetcher which uses the Client to pull data from the endpoint.
//
// This file expects the endpoint to satisfy the following API:
//
// GET {endpoint}?req-id=req1,req2&imp-ids=imp1,imp2,imp3
//
// This endpoint should return a payload like:
//
// {
//   "request": {
//     "req1": { ... stored data for req1 ... },
//     "req2": { ... stored data for req2 ... },
//   },
//   "imps": {
//     "imp1": { ... stored data for imp1 ... },
//     "imp2": { ... stored data for imp2 ... },
//     "imp3": null // If imp3 is not found
//   }
// }
//
// If the request, or any of the imps are not found, then
func NewFetcher(client *http.Client, endpoint string) *httpFetcher {
	return &httpFetcher{
		client:   client,
		endpoint: endpoint,
	}
}

type httpFetcher struct {
	client   *http.Client
	endpoint string
}

func (fetcher *httpFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	httpReq, err := buildRequest(fetcher.endpoint, requestIDs, impIDs)
	if err != nil {
		return nil, nil, []error{err}
	}

	httpResp, err := ctxhttp.Do(ctx, fetcher.client, httpReq)
	if err != nil {
		return nil, nil, []error{err}
	}
	requestData, impData, errs = unpackResponse(httpResp)
	return
}

func buildRequest(endpoint string, requestIDs []string, impIDs []string) (*http.Request, error) {
	// TODO: Build query params
	return http.NewRequest("GET", endpoint, strings.NewReader(""))
}

func unpackResponse(resp *http.Response) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	// TODO: Implement
	return nil, nil, nil
}
