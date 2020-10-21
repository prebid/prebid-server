package http_fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSingleReq(t *testing.T) {
	fetcher, close := newTestFetcher(t, []string{"req-1"}, nil)
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), []string{"req-1"}, nil)
	assert.Empty(t, errs, "Unexpected errors fetching known requests")
	assertMapKeys(t, reqData, "req-1")
	assert.Empty(t, impData, "Unexpected imps returned fetching just requests")
}

func TestSeveralReqs(t *testing.T) {
	fetcher, close := newTestFetcher(t, []string{"req-1", "req-2"}, nil)
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), []string{"req-1", "req-2"}, nil)
	assert.Empty(t, errs, "Unexpected errors fetching known requests")
	assertMapKeys(t, reqData, "req-1", "req-2")
	assert.Empty(t, impData, "Unexpected imps returned fetching just requests")
}

func TestSingleImp(t *testing.T) {
	fetcher, close := newTestFetcher(t, nil, []string{"imp-1"})
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), nil, []string{"imp-1"})
	assert.Empty(t, errs, "Unexpected errors fetching known imps")
	assert.Empty(t, reqData, "Unexpected requests returned fetching just imps")
	assertMapKeys(t, impData, "imp-1")
}

func TestSeveralImps(t *testing.T) {
	fetcher, close := newTestFetcher(t, nil, []string{"imp-1", "imp-2"})
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), nil, []string{"imp-1", "imp-2"})
	assert.Empty(t, errs, "Unexpected errors fetching known imps")
	assert.Empty(t, reqData, "Unexpected requests returned fetching just imps")
	assertMapKeys(t, impData, "imp-1", "imp-2")
}

func TestReqsAndImps(t *testing.T) {
	fetcher, close := newTestFetcher(t, []string{"req-1"}, []string{"imp-1"})
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), []string{"req-1"}, []string{"imp-1"})
	assert.Empty(t, errs, "Unexpected errors fetching known reqs and imps")
	assertMapKeys(t, reqData, "req-1")
	assertMapKeys(t, impData, "imp-1")
}

func TestMissingValues(t *testing.T) {
	fetcher, close := newEmptyFetcher(t, []string{"req-1", "req-2"}, []string{"imp-1"})
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), []string{"req-1", "req-2"}, []string{"imp-1"})
	assert.Empty(t, reqData, "Fetching unknown reqs should return no reqs")
	assert.Empty(t, impData, "Fetching unknown imps should return no imps")
	assert.Len(t, errs, 3, "Fetching 3 unknown reqs+imps should return 3 errors")
}

func TestFetchAccounts(t *testing.T) {
	fetcher, close := newTestAccountFetcher(t, []string{"acc-1", "acc-2"})
	defer close()

	accData, errs := fetcher.FetchAccounts(context.Background(), []string{"acc-1", "acc-2"})
	assert.Empty(t, errs, "Unexpected error fetching known accounts")
	assertMapKeys(t, accData, "acc-1", "acc-2")
}

func TestFetchAccountsNoData(t *testing.T) {
	fetcher, close := newFetcherBrokenBackend()
	defer close()

	accData, errs := fetcher.FetchAccounts(context.Background(), []string{"req-1"})
	assert.Len(t, errs, 1, "Fetching unknown account should have returned an error")
	assert.Nil(t, accData, "Fetching unknown account should return nil account map")
}

func TestFetchAccountsBadJSON(t *testing.T) {
	fetcher, close := newFetcherBadJSON()
	defer close()

	accData, errs := fetcher.FetchAccounts(context.Background(), []string{"req-1"})
	assert.Len(t, errs, 1, "Fetching account with broken json should have returned an error")
	assert.Nil(t, accData, "Fetching account with broken json should return nil account map")
}

func TestFetchAccountsNoIDsProvided(t *testing.T) {
	fetcher, close := newTestAccountFetcher(t, []string{"acc-1", "acc-2"})
	defer close()

	accData, errs := fetcher.FetchAccounts(nil, []string{})
	assert.Empty(t, errs, "Unexpected error fetching empty account list")
	assert.Nil(t, accData, "Fetching empty account list should return nil")
}

// Force build request failure by not providing a context
func TestFetchAccountsFailedBuildRequest(t *testing.T) {
	fetcher, close := newTestAccountFetcher(t, []string{"acc-1", "acc-2"})
	defer close()

	accData, errs := fetcher.FetchAccounts(nil, []string{"acc-1"})
	assert.Len(t, errs, 1, "Fetching accounts without context should result in error ")
	assert.Nil(t, accData, "Fetching accounts without context should return nil")
}

// Force http error via request timeout
func TestFetchAccountsContextTimeout(t *testing.T) {
	fetcher, close := newTestAccountFetcher(t, []string{"acc-1", "acc-2"})
	defer close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(0))
	defer cancel()
	accData, errs := fetcher.FetchAccounts(ctx, []string{"acc-1"})
	assert.Len(t, errs, 1, "Expected context timeout error")
	assert.Nil(t, accData, "Unexpected account data returned instead of timeout")
}

func TestFetchAccount(t *testing.T) {
	fetcher, close := newTestAccountFetcher(t, []string{"acc-1"})
	defer close()

	account, errs := fetcher.FetchAccount(context.Background(), "acc-1")
	assert.Empty(t, errs, "Unexpected error fetching existing account")
	assert.JSONEq(t, `"acc-1"`, string(account), "Unexpected account data fetching existing account")
}

func TestFetchAccountNoData(t *testing.T) {
	fetcher, close := newFetcherBrokenBackend()
	defer close()

	unknownAccount, errs := fetcher.FetchAccount(context.Background(), "unknown-acc")
	assert.NotEmpty(t, errs, "Retrieving unknown account should return error")
	assert.Nil(t, unknownAccount, "Retrieving unknown account should return nil json.RawMessage")
}

func TestFetchAccountNoIDProvided(t *testing.T) {
	fetcher, close := newTestAccountFetcher(t, nil)
	defer close()

	account, errs := fetcher.FetchAccount(context.Background(), "")
	assert.Len(t, errs, 1, "Fetching account with empty id should error")
	assert.Nil(t, account, "Fetching account with empty id should return nil")
}

func TestErrResponse(t *testing.T) {
	fetcher, close := newFetcherBrokenBackend()
	defer close()
	reqData, impData, errs := fetcher.FetchRequests(context.Background(), []string{"req-1"}, []string{"imp-1"})
	assertMapKeys(t, reqData)
	assertMapKeys(t, impData)
	assert.Len(t, errs, 1)
}

func assertSameContents(t *testing.T, expected map[string]json.RawMessage, actual map[string]json.RawMessage) {
	if len(expected) != len(actual) {
		t.Errorf("Wrong counts. Expected %d, actual %d", len(expected), len(actual))
		return
	}
	for expectedKey, expectedVal := range expected {
		if actualVal, ok := actual[expectedKey]; ok {
			if !bytes.Equal(expectedVal, actualVal) {
				t.Errorf("actual[%s] value %s does not match expected: %s", expectedKey, string(actualVal), string(actualVal))
			}
		} else {
			t.Errorf("actual map missing expected key %s", expectedKey)
		}
	}
}

func assertSameErrMsgs(t *testing.T, expected []string, actual []error) {
	if len(expected) != len(actual) {
		t.Errorf("Wrong error counts. Expected %d, actual %d", len(expected), len(actual))
		return
	}
	for i, expectedErr := range expected {
		if actual[i].Error() != expectedErr {
			t.Errorf("Wrong error[%d]. Expected %s, got %s", i, expectedErr, actual[i].Error())
		}
	}
}

type closeWrapper struct {
	io.Reader
}

func (w closeWrapper) Close() error {
	return nil
}

func newFetcherBrokenBackend() (fetcher *HttpFetcher, closer func()) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	return NewFetcher(server.Client(), server.URL), server.Close
}

func newFetcherBadJSON() (fetcher *HttpFetcher, closer func()) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`broken JSON`))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	return NewFetcher(server.Client(), server.URL), server.Close
}

func newEmptyFetcher(t *testing.T, expectReqIDs []string, expectImpIDs []string) (fetcher *HttpFetcher, closer func()) {
	handler := newHandler(t, expectReqIDs, expectImpIDs, jsonifyToNull)
	server := httptest.NewServer(http.HandlerFunc(handler))
	return NewFetcher(server.Client(), server.URL), server.Close
}

func newTestFetcher(t *testing.T, expectReqIDs []string, expectImpIDs []string) (fetcher *HttpFetcher, closer func()) {
	handler := newHandler(t, expectReqIDs, expectImpIDs, jsonifyID)
	server := httptest.NewServer(http.HandlerFunc(handler))
	return NewFetcher(server.Client(), server.URL), server.Close
}

func newHandler(t *testing.T, expectReqIDs []string, expectImpIDs []string, jsonifier func(string) json.RawMessage) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		gotReqIDs := richSplit(query.Get("request-ids"))
		gotImpIDs := richSplit(query.Get("imp-ids"))

		assertMatches(t, gotReqIDs, expectReqIDs)
		assertMatches(t, gotImpIDs, expectImpIDs)

		reqIDResponse := make(map[string]json.RawMessage, len(gotReqIDs))
		impIDResponse := make(map[string]json.RawMessage, len(gotImpIDs))

		for _, reqID := range gotReqIDs {
			if reqID != "" {
				reqIDResponse[reqID] = jsonifier(reqID)
			}
		}

		for _, impID := range gotImpIDs {
			if impID != "" {
				impIDResponse[impID] = jsonifier(impID)
			}
		}

		respObj := responseContract{
			Requests: reqIDResponse,
			Imps:     impIDResponse,
		}

		if respBytes, err := json.Marshal(respObj); err != nil {
			t.Errorf("failed to marshal responseContract in test:  %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Write(respBytes)
		}
	}
}

func newTestAccountFetcher(t *testing.T, expectAccIDs []string) (fetcher *HttpFetcher, closer func()) {
	handler := newAccountHandler(t, expectAccIDs, jsonifyID)
	server := httptest.NewServer(http.HandlerFunc(handler))
	return NewFetcher(server.Client(), server.URL), server.Close
}

func newAccountHandler(t *testing.T, expectAccIDs []string, jsonifier func(string) json.RawMessage) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		gotAccIDs := richSplit(query.Get("account-ids"))

		assertMatches(t, gotAccIDs, expectAccIDs)

		accIDResponse := make(map[string]json.RawMessage, len(gotAccIDs))

		for _, accID := range gotAccIDs {
			if accID != "" {
				accIDResponse[accID] = jsonifier(accID)
			}
		}

		respObj := accountsResponseContract{
			Accounts: accIDResponse,
		}

		if respBytes, err := json.Marshal(respObj); err != nil {
			t.Errorf("failed to marshal responseContract in test:  %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Write(respBytes)
		}
	}
}

func assertMatches(t *testing.T, queryVals []string, expected []string) {
	t.Helper()

	if len(queryVals) == 1 && queryVals[0] == "" {
		if len(expected) != 0 {
			t.Errorf("Expected no query vals, but got %v", queryVals)
		}
		return
	}
	if len(queryVals) != len(expected) {
		t.Errorf("Query vals did not match. Expected %v, got %v", expected, queryVals)
		return
	}

	for _, expectedQuery := range expected {
		found := false
		for _, queryVal := range queryVals {
			if queryVal == expectedQuery {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Query string missing expected key %s.", expectedQuery)
		}
	}
}

// Split the id values properly
func richSplit(queryVal string) []string {
	// Get rid of the bounding []
	// Not doing real validation, as this is a test routine, and given a malformed input we want to fail anyway.
	if len(queryVal) > 2 {
		queryVal = queryVal[1 : len(queryVal)-1]
	}
	values := strings.Split(queryVal, "\",\"")
	if len(values) > 0 {
		//Fix leading and trailing "
		if len(values[0]) > 0 {
			values[0] = values[0][1:]
		}
		l := len(values) - 1
		if len(values[l]) > 0 {
			values[l] = values[l][:len(values[l])-1]
		}
	}

	return values
}

func jsonifyID(id string) json.RawMessage {
	if b, err := json.Marshal(id); err != nil {
		return json.RawMessage([]byte("\"error encoding ID=" + id + "\""))
	} else {
		return json.RawMessage(b)
	}
}

func jsonifyToNull(id string) json.RawMessage {
	return json.RawMessage("null")
}

func assertMapKeys(t *testing.T, m map[string]json.RawMessage, keys ...string) {
	t.Helper()

	if len(m) != len(keys) {
		t.Errorf("Expected %d map keys. Map was: %v", len(keys), m)
	}

	for _, key := range keys {
		if _, ok := m[key]; !ok {
			t.Errorf("Map missing expected key %s. Data was %v", key, m)
		}
	}
}
