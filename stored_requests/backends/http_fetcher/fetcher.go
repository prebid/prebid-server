package http_fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/stored_requests"

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
// Accounts
// GET {endpoint}?account-ids=["acc1","acc2"]
//
// The above endpoints should return a payload like:
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
// or
// {
//   "accounts": {
//     "acc1": { ... config data for acc1 ... },
//     "acc2": { ... config data for acc2 ... },
//   },
// }
//
//
func NewFetcher(client *http.Client, endpoint string) *HttpFetcher {
	// Do some work up-front to figure out if the (configurable) endpoint has a query string or not.
	// When we build requests, we'll either want to add `?request-ids=...&imp-ids=...` _or_
	// `&request-ids=...&imp-ids=...`.

	if _, err := url.Parse(endpoint); err != nil {
		glog.Fatalf(`Invalid endpoint "%s": %v`, endpoint, err)
	}
	glog.Infof("Making http_fetcher for endpoint %v", endpoint)

	urlPrefix := endpoint
	if strings.Contains(endpoint, "?") {
		urlPrefix = urlPrefix + "&"
	} else {
		urlPrefix = urlPrefix + "?"
	}

	return &HttpFetcher{
		client:   client,
		Endpoint: urlPrefix,
	}
}

type HttpFetcher struct {
	client     *http.Client
	Endpoint   string
	hasQuery   bool
	Categories map[string]map[string]stored_requests.Category
}

func (fetcher *HttpFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	if len(requestIDs) == 0 && len(impIDs) == 0 {
		return nil, nil, nil
	}

	httpReq, err := buildRequest(fetcher.Endpoint, requestIDs, impIDs)
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

// FetchAccounts retrieves account configurations
//
// Request format is similar to the one for requests:
// GET {endpoint}?account-ids=["account1","account2",...]
//
// The endpoint is expected to respond with a JSON map with accountID -> json.RawMessage
// {
//   "account1": { ... account json ... }
// }
// The JSON contents of account config is returned as-is (NOT validated)
func (fetcher *HttpFetcher) FetchAccounts(ctx context.Context, accountIDs []string) (map[string]json.RawMessage, []error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", fetcher.Endpoint+"account-ids=[\""+strings.Join(accountIDs, "\",\"")+"\"]", nil)
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
	respBytes, err := ioutil.ReadAll(httpResp.Body)
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
	if err = json.Unmarshal(respBytes, &responseData); err != nil {
		return nil, []error{
			fmt.Errorf(`Error fetching accounts %v via http: failed to parse response: %v`, accountIDs, err),
		}
	}
	errs := convertNullsToErrs(responseData.Accounts, "Account", []error{})
	return responseData.Accounts, errs
}

// FetchAccount fetchers a single accountID and returns its corresponding json
func (fetcher *HttpFetcher) FetchAccount(ctx context.Context, accountID string) (accountJSON json.RawMessage, errs []error) {
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
	return accountJSON, nil
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
		url = fmt.Sprintf("%s/%s/%s.json", strings.TrimSuffix(fetcher.Endpoint, "?"), primaryAdServer, publisherId)
	} else {
		dataName = primaryAdServer
		url = fmt.Sprintf("%s/%s.json", strings.TrimSuffix(fetcher.Endpoint, "?"), primaryAdServer)
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

	respBytes, err := ioutil.ReadAll(httpResp.Body)
	tmp := make(map[string]stored_requests.Category)

	if err := json.Unmarshal(respBytes, &tmp); err != nil {
		return "", fmt.Errorf("Unable to unmarshal categories for adserver: '%s', publisherId: '%s'", primaryAdServer, publisherId)
	}
	fetcher.Categories[dataName] = tmp

	if val, ok := tmp[iabCategory]; ok {
		return val.Id, nil
	} else {
		return "", fmt.Errorf("Unable to find category mapping for adserver: '%s', publisherId: '%s'", primaryAdServer, publisherId)
	}
}

func buildRequest(endpoint string, requestIDs []string, impIDs []string) (*http.Request, error) {
	if len(requestIDs) > 0 && len(impIDs) > 0 {
		return http.NewRequest("GET", endpoint+"request-ids=[\""+strings.Join(requestIDs, "\",\"")+"\"]&imp-ids=[\""+strings.Join(impIDs, "\",\"")+"\"]", nil)
	} else if len(requestIDs) > 0 {
		return http.NewRequest("GET", endpoint+"request-ids=[\""+strings.Join(requestIDs, "\",\"")+"\"]", nil)
	} else {
		return http.NewRequest("GET", endpoint+"imp-ids=[\""+strings.Join(impIDs, "\",\"")+"\"]", nil)
	}
}

func unpackResponse(resp *http.Response) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errs = append(errs, err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		var responseObj responseContract
		if err := json.Unmarshal(respBytes, &responseObj); err != nil {
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
		if bytes.Equal(val, []byte("null")) {
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
