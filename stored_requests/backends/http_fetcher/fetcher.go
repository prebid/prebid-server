package http_fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"

	"github.com/golang/glog"
	"golang.org/x/net/context/ctxhttp"
)

// NewFetcher returns a Fetcher which uses the Client to pull data from the endpoint.
//
// This file expects the endpoint to satisfy the following API:
//
// Stored requests
// GET {endpoint}?request-ids=["req1","req2"]&imp-ids=["imp1","imp2","imp3"]
//
// If useRfcCompliantBuilder is true (symbols will be URLEncoded)
// GET {endpoint}?request-id=req1&request-id=req2&imp-id=imp1&imp-id=imp2&imp-id=imp3
//
// Accounts
// GET {endpoint}?account-ids=["acc1","acc2"]
//
// If UseRfcCompliantBuilder is true (symbols will be URLEncoded)
// GET {endpoint}?account-id=acc1&account-id=acc2
//
// The above endpoints should return a payload like:
//
//	{
//	  "requests": {
//	    "req1": { ... stored data for req1 ... },
//	    "req2": { ... stored data for req2 ... },
//	  },
//	  "imps": {
//	    "imp1": { ... stored data for imp1 ... },
//	    "imp2": { ... stored data for imp2 ... },
//	    "imp3": null // If imp3 is not found
//	  }
//	}
//
// or
//
//	{
//	  "accounts": {
//	    "acc1": { ... config data for acc1 ... },
//	    "acc2": { ... config data for acc2 ... },
//	  },
//	}
func NewFetcher(client *http.Client, endpoint string, useRfcCompliantBuilder bool) *HttpFetcher {
	endpointURL, err := url.Parse(endpoint)

	if err != nil {
		glog.Fatalf(`Invalid endpoint "%s": %v`, endpoint, err)
	}
	glog.Infof("Making http_fetcher for endpoint %v", endpoint)

	return &HttpFetcher{
		client:                 client,
		EndpointURL:            endpointURL,
		UseRfcCompliantBuilder: useRfcCompliantBuilder,
	}
}

type HttpFetcher struct {
	client                 *http.Client
	EndpointURL            *url.URL
	UseRfcCompliantBuilder bool
	Categories             map[string]map[string]stored_requests.Category
}

func (fetcher *HttpFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	if len(requestIDs) == 0 && len(impIDs) == 0 {
		return nil, nil, nil
	}

	httpReq, err := fetcher.buildRequest(requestIDs, impIDs)
	if err != nil {
		return nil, nil, []error{err}
	}

	httpResp, err := ctxhttp.Do(ctx, fetcher.client, httpReq)
	if err != nil {
		return nil, nil, []error{err}
	}
	defer httpResp.Body.Close()
	requestData, impData, errs = unpackResponse(httpResp)
	return
}

func (fetcher *HttpFetcher) FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	return nil, nil
}

// FetchAccounts retrieves account configurations
//
// Request format is similar to the one for requests:
// GET {endpoint}?account-ids=["account1","account2",...]
//
// If UseRfcCompliantBuilder is true (symbols will be URLEncoded):
// GET {endpoint}?account-id=account1&account-id=account2&...
//
// The endpoint is expected to respond with a JSON map with accountID -> json.RawMessage
//
//	{
//	  "account1": { ... account json ... }
//	}
//
// The JSON contents of account config is returned as-is (NOT validated)
func (fetcher *HttpFetcher) FetchAccounts(ctx context.Context, accountIDs []string) (map[string]json.RawMessage, []error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	u := fetcher.EndpointURL
	q := u.Query()
	if !fetcher.UseRfcCompliantBuilder {
		q.Set("account-ids", `["`+strings.Join(accountIDs, `","`)+`"]`)
	} else {
		for _, id := range accountIDs {
			q.Add("account-id", id)
		}
	}
	u.RawQuery = q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, []error{
			fmt.Errorf(`Error fetching accounts %v via http: build request failed with %v`, accountIDs, err),
		}
	}
	httpResp, err := ctxhttp.Do(ctx, fetcher.client, httpReq)
	if err != nil {
		return nil, []error{
			fmt.Errorf(`Error fetching accounts %v via http: %v`, accountIDs, err),
		}
	}
	defer httpResp.Body.Close()
	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, []error{
			fmt.Errorf(`Error fetching accounts %v via http: error reading response: %v`, accountIDs, err),
		}
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, []error{
			fmt.Errorf(`Error fetching accounts %v via http: unexpected response status %d`, accountIDs, httpResp.StatusCode),
		}
	}
	var responseData accountsResponseContract
	if err = jsonutil.UnmarshalValid(respBytes, &responseData); err != nil {
		return nil, []error{
			fmt.Errorf(`Error fetching accounts %v via http: failed to parse response: %v`, accountIDs, err),
		}
	}
	errs := convertNullsToErrs(responseData.Accounts, "Account", []error{})
	return responseData.Accounts, errs
}

// FetchAccount fetchers a single accountID and returns its corresponding json
func (fetcher *HttpFetcher) FetchAccount(ctx context.Context, accountDefaultsJSON json.RawMessage, accountID string) (accountJSON json.RawMessage, errs []error) {
	accountData, errs := fetcher.FetchAccounts(ctx, []string{accountID})
	if len(errs) > 0 {
		return nil, errs
	}
	accountJSON, ok := accountData[accountID]
	if !ok {
		return nil, []error{stored_requests.NotFoundError{
			ID:       accountID,
			DataType: "Account",
		}}
	}
	if accountDefaultsJSON == nil {
		return accountJSON, nil
	}
	completeJSON, err := jsonpatch.MergePatch(accountDefaultsJSON, accountJSON)
	if err != nil {
		return nil, []error{err}
	}
	return completeJSON, nil
}

func (fetcher *HttpFetcher) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
	if fetcher.Categories == nil {
		fetcher.Categories = make(map[string]map[string]stored_requests.Category)
	}

	//in NewFetcher function there is a code to add "?" at the end of url
	//in case of categories we don't expect to have any parameters, that's why we need to remove "?"
	var dataName, url string
	if publisherId != "" {
		dataName = fmt.Sprintf("%s_%s", primaryAdServer, publisherId)
		url = fmt.Sprintf("%s/%s/%s.json", fetcher.EndpointURL.String(), primaryAdServer, publisherId)
	} else {
		dataName = primaryAdServer
		url = fmt.Sprintf("%s/%s.json", fetcher.EndpointURL.String(), primaryAdServer)
	}

	if data, ok := fetcher.Categories[dataName]; ok {
		if val, ok := data[iabCategory]; ok {
			return val.Id, nil
		} else {
			return "", fmt.Errorf("Unable to find category mapping for adserver: '%s', publisherId: '%s'", primaryAdServer, publisherId)
		}
	}

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	httpResp, err := ctxhttp.Do(ctx, fetcher.client, httpReq)
	if err != nil {
		return "", err
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("Unable to read response body: %v", err)
	}
	tmp := make(map[string]stored_requests.Category)

	if err := jsonutil.UnmarshalValid(respBytes, &tmp); err != nil {
		return "", fmt.Errorf("Unable to unmarshal categories for adserver: '%s', publisherId: '%s'", primaryAdServer, publisherId)
	}
	fetcher.Categories[dataName] = tmp

	if val, ok := tmp[iabCategory]; ok {
		return val.Id, nil
	} else {
		return "", fmt.Errorf("Unable to find category mapping for adserver: '%s', publisherId: '%s'", primaryAdServer, publisherId)
	}
}

func AddQueryParam(q *url.Values, paramName string, ids []string, useRfcCompliantBuilder bool) {
	if len(ids) > 0 {
		if !useRfcCompliantBuilder {
			q.Set(paramName+"s", `["`+strings.Join(ids, `","`)+`"]`)
		} else {
			for _, requestID := range ids {
				q.Add(paramName, requestID)
			}
		}
	}
}

func (fetcher *HttpFetcher) buildRequest(requestIDs []string, impIDs []string) (*http.Request, error) {
	u := *fetcher.EndpointURL

	q := u.Query()

	AddQueryParam(&q, "request-id", requestIDs, fetcher.UseRfcCompliantBuilder)
	AddQueryParam(&q, "imp-id", impIDs, fetcher.UseRfcCompliantBuilder)

	u.RawQuery = q.Encode()

	return http.NewRequest("GET", u.String(), nil)
}

func unpackResponse(resp *http.Response) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		errs = append(errs, err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		var responseObj responseContract
		if err := jsonutil.UnmarshalValid(respBytes, &responseObj); err != nil {
			errs = append(errs, err)
			return
		}

		requestData = responseObj.Requests
		impData = responseObj.Imps

		errs = convertNullsToErrs(requestData, "Request", errs)
		errs = convertNullsToErrs(impData, "Imp", errs)

		return
	}

	errs = append(errs, fmt.Errorf("Error fetching Stored Requests via HTTP. Response code was %d", resp.StatusCode))
	return
}

func convertNullsToErrs(m map[string]json.RawMessage, dataType string, errs []error) []error {
	for id, val := range m {
		if val == nil {
			delete(m, id)
			errs = append(errs, stored_requests.NotFoundError{
				ID:       id,
				DataType: dataType,
			})
		}
	}
	return errs
}

// responseContract is used to unmarshal  for the endpoint
type responseContract struct {
	Requests map[string]json.RawMessage `json:"requests"`
	Imps     map[string]json.RawMessage `json:"imps"`
}

type accountsResponseContract struct {
	Accounts map[string]json.RawMessage `json:"accounts"`
}
