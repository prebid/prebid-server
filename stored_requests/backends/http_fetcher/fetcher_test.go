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
)

func TestSingleReq(t *testing.T) {
	fetcher, close := newTestFetcher(t, []string{"req-1"}, nil)
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), []string{"req-1"}, nil)
	assertMapKeys(t, reqData, "req-1")
	assertMapKeys(t, impData)
	assertErrLength(t, errs, 0)
}

func TestSeveralReqs(t *testing.T) {
	fetcher, close := newTestFetcher(t, []string{"req-1", "req-2"}, nil)
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), []string{"req-1", "req-2"}, nil)
	assertMapKeys(t, reqData, "req-1", "req-2")
	assertMapKeys(t, impData)
	assertErrLength(t, errs, 0)
}

func TestSingleImp(t *testing.T) {
	fetcher, close := newTestFetcher(t, nil, []string{"imp-1"})
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), nil, []string{"imp-1"})
	assertMapKeys(t, reqData)
	assertMapKeys(t, impData, "imp-1")
	assertErrLength(t, errs, 0)
}

func TestSeveralImps(t *testing.T) {
	fetcher, close := newTestFetcher(t, nil, []string{"imp-1", "imp-2"})
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), nil, []string{"imp-1", "imp-2"})
	assertMapKeys(t, reqData)
	assertMapKeys(t, impData, "imp-1", "imp-2")
	assertErrLength(t, errs, 0)
}

func TestReqsAndImps(t *testing.T) {
	fetcher, close := newTestFetcher(t, []string{"req-1"}, []string{"imp-1"})
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), []string{"req-1"}, []string{"imp-1"})
	assertMapKeys(t, reqData, "req-1")
	assertMapKeys(t, impData, "imp-1")
	assertErrLength(t, errs, 0)
}

func TestMissingValues(t *testing.T) {
	fetcher, close := newEmptyFetcher(t, []string{"req-1", "req-2"}, []string{"imp-1"})
	defer close()

	reqData, impData, errs := fetcher.FetchRequests(context.Background(), []string{"req-1", "req-2"}, []string{"imp-1"})
	assertMapKeys(t, reqData)
	assertMapKeys(t, impData)
	assertErrLength(t, errs, 3)
}

func TestErrResponse(t *testing.T) {
	fetcher, close := newFetcherBrokenBackend()
	defer close()
	reqData, impData, errs := fetcher.FetchRequests(context.Background(), []string{"req-1"}, []string{"imp-1"})
	assertMapKeys(t, reqData)
	assertMapKeys(t, impData)
	assertErrLength(t, errs, 1)
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
		assertMatches(t, query.Get("request-ids"), expectReqIDs)
		assertMatches(t, query.Get("imp-ids"), expectImpIDs)

		gotReqIDs := richSplit(query.Get("request-ids"))
		gotImpIDs := richSplit(query.Get("imp-ids"))

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

func assertMatches(t *testing.T, query string, expected []string) {
	t.Helper()

	queryVals := richSplit(query)
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

func assertErrLength(t *testing.T, errs []error, expected int) {
	t.Helper()

	if len(errs) != expected {
		t.Errorf("Expected %d errors. Got: %v", expected, errs)
	}
}
