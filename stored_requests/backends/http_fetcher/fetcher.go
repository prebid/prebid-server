package http_fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"golang.org/x/net/context/ctxhttp"
)

// NewFetcher returns a Fetcher which uses the Client to pull data from the endpoint.
//
// This file expects the endpoint to satisfy the following API:
//
// GET {endpoint}?req-ids=req1,req2&imp-ids=imp1,imp2,imp3
//
// This endpoint should return a payload like:
//
// {
//   "requests": {
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
//
func NewFetcher(client *http.Client, endpoint string) *httpFetcher {
	// Do some work up-front to figure out if the (configurable) endpoint has a query string or not.
	// When we build requests, we'll either want to add `?req-ids=...&imp-ids=...` _or_
	// `&req-ids=...&imp-ids=...`, depending.
	urlPrefix := endpoint
	if strings.Contains(endpoint, "?") {
		urlPrefix += "&"
	} else {
		urlPrefix += "?"
	}

	glog.Infof("http_fetcher will use: GET " + urlPrefix + "req-ids=%REQUEST_ID_LIST%&imp-ids=%IMP_ID_LIST%")

	return &httpFetcher{
		client:   client,
		endpoint: urlPrefix,
	}
}

type httpFetcher struct {
	client   *http.Client
	endpoint string
	hasQuery bool
}

func (fetcher *httpFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	if len(requestIDs) == 0 && len(impIDs) == 0 {
		return nil, nil, nil
	}

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
	if len(requestIDs) > 0 && len(impIDs) > 0 {
		return http.NewRequest("GET", endpoint+"req-ids="+strings.Join(requestIDs, ",")+"&imp-ids="+strings.Join(impIDs, ","), nil)
	} else if len(requestIDs) > 0 {
		return http.NewRequest("GET", endpoint+"req-ids="+strings.Join(requestIDs, ","), nil)
	} else {
		return http.NewRequest("GET", endpoint+"imp-ids="+strings.Join(impIDs, ","), nil)
	}
}

func unpackResponse(resp *http.Response) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, []error{err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var responseObj responseContract
		if err := json.Unmarshal(respBytes, &responseObj); err != nil {
			return nil, nil, []error{err}
		}

		return responseObj.Requests, responseObj.Imps, nil
	}

	return nil, nil, []error{fmt.Errorf("Error fetching Stored Requests via HTTP. Response code was %d", resp.StatusCode)}
}

// responseContract is used to unmarshal  for the endpoint
type responseContract struct {
	Requests map[string]json.RawMessage `json:"requests"`
	Imps     map[string]json.RawMessage `json:"imps"`
}
