package stored_requests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStartupReqsOnly(t *testing.T) {
	server := httptest.NewServer(&mockResponseHandler{
		statusCode: http.StatusOK,
		response:   `{"requests":{"request1":{"value":1}, "request2":{"value":2}}}`,
	})
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	theUpdate := <-ev.Updates()

	assertLen(t, theUpdate.Requests, 2)
	assertHasValue(t, theUpdate.Requests, "request1", `{"value":1}`)
	assertHasValue(t, theUpdate.Requests, "request2", `{"value":2}`)

	assertLen(t, theUpdate.Imps, 0)
}

func TestStartupImpsOnly(t *testing.T) {
	server := httptest.NewServer(&mockResponseHandler{
		statusCode: http.StatusOK,
		response:   `{"imps":{"imp1":{"value":1}}}`,
	})
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	theUpdate := <-ev.Updates()

	assertLen(t, theUpdate.Requests, 0)

	assertLen(t, theUpdate.Imps, 1)
	assertHasValue(t, theUpdate.Imps, "imp1", `{"value":1}`)
}

func TestStartupBothTypes(t *testing.T) {
	server := httptest.NewServer(&mockResponseHandler{
		statusCode: http.StatusOK,
		response:   `{"requests":{"request1":{"value":1}, "request2":{"value":2}},"imps":{"imp1":{"value":1}}}`,
	})
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	theUpdate := <-ev.Updates()

	assertLen(t, theUpdate.Requests, 2)
	assertHasValue(t, theUpdate.Requests, "request1", `{"value":1}`)
	assertHasValue(t, theUpdate.Requests, "request2", `{"value":2}`)

	assertLen(t, theUpdate.Imps, 1)
	assertHasValue(t, theUpdate.Imps, "imp1", `{"value":1}`)
}

func TestUpdates(t *testing.T) {
	handler := &mockResponseHandler{
		statusCode: http.StatusOK,
		response:   `{"requests":{"request1":{"value":1}, "request2":{"value":2}},"imps":{"imp1":{"value":3},"imp2":{"value":4}}}`,
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)

	handler.response = `{"requests":{"request1":{"value":5}, "request2":{"deleted":true}},"imps":{"imp1":{"deleted":true},"imp2":{"value":6}}}`
	timeChan := make(chan time.Time, 1)
	timeChan <- time.Now()
	go ev.refresh(timeChan)
	firstUpdate := <-ev.Updates()
	secondUpdate := <-ev.Updates()
	inv := <-ev.Invalidations()

	assertLen(t, firstUpdate.Requests, 2)
	assertHasValue(t, firstUpdate.Requests, "request1", `{"value":1}`)
	assertHasValue(t, firstUpdate.Requests, "request2", `{"value":2}`)
	assertLen(t, firstUpdate.Imps, 2)
	assertHasValue(t, firstUpdate.Imps, "imp1", `{"value":3}`)
	assertHasValue(t, firstUpdate.Imps, "imp2", `{"value":4}`)

	assertLen(t, secondUpdate.Requests, 1)
	assertHasValue(t, secondUpdate.Requests, "request1", `{"value":5}`)
	assertLen(t, secondUpdate.Imps, 1)
	assertHasValue(t, secondUpdate.Imps, "imp2", `{"value":6}`)

	assertArrLen(t, inv.Requests, 1)
	assertArrContains(t, inv.Requests, "request2")
	assertArrLen(t, inv.Imps, 1)
	assertArrContains(t, inv.Imps, "imp1")
}

func TestErrorResponse(t *testing.T) {
	handler := &mockResponseHandler{
		statusCode: http.StatusInternalServerError,
		response:   "Something horrible happened.",
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	if len(ev.Updates()) != 0 {
		t.Errorf("No updates should be emitted if the HTTP call fails. Got %d", len(ev.Updates()))
	}
}

func TestExpiredContext(t *testing.T) {
	handler := &mockResponseHandler{
		statusCode: http.StatusInternalServerError,
		response:   "Something horrible happened.",
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ctxProducer := func() (context.Context, func()) {
		return context.WithTimeout(context.Background(), -1)
	}

	ev := NewHTTPEvents(server.Client(), server.URL, ctxProducer, -1)
	if len(ev.Updates()) != 0 {
		t.Errorf("No updates should be emitted if the HTTP call is cancelled. Got %d", len(ev.Updates()))
	}
}

func TestMalformedResponse(t *testing.T) {
	handler := &mockResponseHandler{
		statusCode: http.StatusOK,
		response:   "This isn't JSON.",
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	if len(ev.Updates()) != 0 {
		t.Errorf("No updates should be emitted if the HTTP call fails. Got %d", len(ev.Updates()))
	}
}

type mockResponseHandler struct {
	statusCode int
	response   string
}

func (m *mockResponseHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(m.statusCode)
	rw.Write([]byte(m.response))
}

func assertLen(t *testing.T, m map[string]json.RawMessage, length int) {
	t.Helper()
	if len(m) != length {
		t.Errorf("Expected map with %d elements, but got %v", length, m)
	}
}

func assertArrLen(t *testing.T, list []string, length int) {
	t.Helper()
	if len(list) != length {
		t.Errorf("Expected list with %d elements, but got %v", length, list)
	}
}

func assertArrContains(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, elm := range haystack {
		if elm == needle {
			return
		}
	}
	t.Errorf("expected element %s to be in list %v", needle, haystack)
}

func assertHasValue(t *testing.T, m map[string]json.RawMessage, key string, val string) {
	t.Helper()
	if mapVal, ok := m[key]; ok {
		if !bytes.Equal(mapVal, []byte(val)) {
			t.Errorf("expected map[%s] to be %s, but got %s", key, val, string(mapVal))
		}
	} else {
		t.Errorf("map missing expected key: %s", key)
	}
}
