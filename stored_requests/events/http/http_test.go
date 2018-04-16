package http

import (
	"bytes"
	"context"
	"encoding/json"
	httpCore "net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStartupReqsOnly(t *testing.T) {
	server := httptest.NewServer(&mockResponseHandler{
		statusCode: httpCore.StatusOK,
		response:   `{"requests":{"request1":{"value":1}, "request2":{"value":2}}}`,
	})
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	theSave := <-ev.Saves()

	assertLen(t, theSave.Requests, 2)
	assertHasValue(t, theSave.Requests, "request1", `{"value":1}`)
	assertHasValue(t, theSave.Requests, "request2", `{"value":2}`)

	assertLen(t, theSave.Imps, 0)
}

func TestStartupImpsOnly(t *testing.T) {
	server := httptest.NewServer(&mockResponseHandler{
		statusCode: httpCore.StatusOK,
		response:   `{"imps":{"imp1":{"value":1}}}`,
	})
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	theSave := <-ev.Saves()

	assertLen(t, theSave.Requests, 0)

	assertLen(t, theSave.Imps, 1)
	assertHasValue(t, theSave.Imps, "imp1", `{"value":1}`)
}

func TestStartupBothTypes(t *testing.T) {
	server := httptest.NewServer(&mockResponseHandler{
		statusCode: httpCore.StatusOK,
		response:   `{"requests":{"request1":{"value":1}, "request2":{"value":2}},"imps":{"imp1":{"value":1}}}`,
	})
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	theSave := <-ev.Saves()

	assertLen(t, theSave.Requests, 2)
	assertHasValue(t, theSave.Requests, "request1", `{"value":1}`)
	assertHasValue(t, theSave.Requests, "request2", `{"value":2}`)

	assertLen(t, theSave.Imps, 1)
	assertHasValue(t, theSave.Imps, "imp1", `{"value":1}`)
}

func TestUpdates(t *testing.T) {
	handler := &mockResponseHandler{
		statusCode: httpCore.StatusOK,
		response:   `{"requests":{"request1":{"value":1}, "request2":{"value":2}},"imps":{"imp1":{"value":3},"imp2":{"value":4}}}`,
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)

	handler.response = `{"requests":{"request1":{"value":5}, "request2":{"deleted":true}},"imps":{"imp1":{"deleted":true},"imp2":{"value":6}}}`
	timeChan := make(chan time.Time, 1)
	timeChan <- time.Now()
	go ev.refresh(timeChan)
	firstSave := <-ev.Saves()
	secondSave := <-ev.Saves()
	inv := <-ev.Invalidations()

	assertLen(t, firstSave.Requests, 2)
	assertHasValue(t, firstSave.Requests, "request1", `{"value":1}`)
	assertHasValue(t, firstSave.Requests, "request2", `{"value":2}`)
	assertLen(t, firstSave.Imps, 2)
	assertHasValue(t, firstSave.Imps, "imp1", `{"value":3}`)
	assertHasValue(t, firstSave.Imps, "imp2", `{"value":4}`)

	assertLen(t, secondSave.Requests, 1)
	assertHasValue(t, secondSave.Requests, "request1", `{"value":5}`)
	assertLen(t, secondSave.Imps, 1)
	assertHasValue(t, secondSave.Imps, "imp2", `{"value":6}`)

	assertArrLen(t, inv.Requests, 1)
	assertArrContains(t, inv.Requests, "request2")
	assertArrLen(t, inv.Imps, 1)
	assertArrContains(t, inv.Imps, "imp1")
}

func TestErrorResponse(t *testing.T) {
	handler := &mockResponseHandler{
		statusCode: httpCore.StatusInternalServerError,
		response:   "Something horrible happened.",
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	if len(ev.Saves()) != 0 {
		t.Errorf("No saves should be emitted if the HTTP call fails. Got %d", len(ev.Saves()))
	}
}

func TestExpiredContext(t *testing.T) {
	handler := &mockResponseHandler{
		statusCode: httpCore.StatusInternalServerError,
		response:   "Something horrible happened.",
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ctxProducer := func() (context.Context, func()) {
		return context.WithTimeout(context.Background(), -1)
	}

	ev := NewHTTPEvents(server.Client(), server.URL, ctxProducer, -1)
	if len(ev.Saves()) != 0 {
		t.Errorf("No saves should be emitted if the HTTP call is cancelled. Got %d", len(ev.Saves()))
	}
}

func TestMalformedResponse(t *testing.T) {
	handler := &mockResponseHandler{
		statusCode: httpCore.StatusOK,
		response:   "This isn't JSON.",
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	if len(ev.Saves()) != 0 {
		t.Errorf("No updates should be emitted if the HTTP call fails. Got %d", len(ev.Saves()))
	}
}

type mockResponseHandler struct {
	statusCode int
	response   string
}

func (m *mockResponseHandler) ServeHTTP(rw httpCore.ResponseWriter, r *httpCore.Request) {
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
