package http

import (
	"context"
	"encoding/json"
	httpCore "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests/events"
	"github.com/stretchr/testify/assert"
)

func TestStartupReqsOnly(t *testing.T) {
	server := httptest.NewServer(&mockResponseHandler{
		statusCode: httpCore.StatusOK,
		response:   `{"requests":{"request1":{"value":1}, "request2":{"value":2}}}`,
	})
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)

	assertSaveChanReceive(t, ev.Saves(), map[config.DataType]map[string]json.RawMessage{
		config.RequestDataType: {
			"request1": json.RawMessage(`{"value":1}`),
			"request2": json.RawMessage(`{"value":2}`),
		},
	})
	assert.Empty(t, ev.Saves(), "Unexpected additional messages in save channel")
}

func TestStartupImpsOnly(t *testing.T) {
	server := httptest.NewServer(&mockResponseHandler{
		statusCode: httpCore.StatusOK,
		response:   `{"imps":{"imp1":{"value":1}}}`,
	})
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	assertSaveChanReceive(t, ev.Saves(), map[config.DataType]map[string]json.RawMessage{
		config.ImpDataType: {
			"imp1": json.RawMessage(`{"value":1}`),
		},
	})
	assert.Empty(t, ev.Saves(), "Unexpected additional messages in save channel")
}

func TestStartupBothTypes(t *testing.T) {
	server := httptest.NewServer(&mockResponseHandler{
		statusCode: httpCore.StatusOK,
		response:   `{"requests":{"request1":{"value":1}, "request2":{"value":2}},"imps":{"imp1":{"value":1}}}`,
	})
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)

	assertSaveChanReceive(t, ev.Saves(), map[config.DataType]map[string]json.RawMessage{
		config.RequestDataType: {
			"request1": json.RawMessage(`{"value":1}`),
			"request2": json.RawMessage(`{"value":2}`),
		},
		config.ImpDataType: {
			"imp1": json.RawMessage(`{"value":1}`),
		},
	})
	assert.Empty(t, ev.Saves(), "Unexpected additional messages in save channel")
}

func TestStartupAccounts(t *testing.T) {
	server := httptest.NewServer(&mockResponseHandler{
		statusCode: httpCore.StatusOK,
		response:   `{"accounts":{"acc1":{"value":1}}}`,
	})
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	assertSaveChanReceive(t, ev.Saves(), map[config.DataType]map[string]json.RawMessage{
		config.AccountDataType: {
			"acc1": json.RawMessage(`{"value":1}`),
		},
	})
	assert.Empty(t, ev.Saves(), "Unexpected additional messages in save channel")
}

func TestUpdates(t *testing.T) {
	handler := &mockResponseHandler{
		statusCode: httpCore.StatusOK,
		response:   `{"requests":{"request1":{"value":1}, "request2":{"value":2}},"imps":{"imp1":{"value":3},"imp2":{"value":4}},"accounts":{"acc1":{"value":10},"acc2":{"value":11}}}`,
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)

	handler.response = `{"requests":{"request1":{"value":5}, "request2":{"deleted":true}},"imps":{"imp1":{"deleted":true},"imp2":{"value":6}},"accounts":{"acc1":{"deleted":true},"acc2":{"value":12}}}`
	timeChan := make(chan time.Time, 1)
	timeChan <- time.Now()
	go ev.refresh(timeChan)

	assertSaveChanReceive(t, ev.Saves(), map[config.DataType]map[string]json.RawMessage{
		config.RequestDataType: {
			"request1": json.RawMessage(`{"value":1}`),
			"request2": json.RawMessage(`{"value":2}`),
		},
		config.ImpDataType: {
			"imp1": json.RawMessage(`{"value":3}`),
			"imp2": json.RawMessage(`{"value":4}`),
		},
		config.AccountDataType: {
			"acc1": json.RawMessage(`{"value":10}`),
			"acc2": json.RawMessage(`{"value":11}`),
		},
	})
	assertSaveChanReceive(t, ev.Saves(), map[config.DataType]map[string]json.RawMessage{
		config.RequestDataType: {
			"request1": json.RawMessage(`{"value":5}`),
		},
		config.ImpDataType: {
			"imp2": json.RawMessage(`{"value":6}`),
		},
		config.AccountDataType: {
			"acc2": json.RawMessage(`{"value":12}`),
		},
	})
	assert.Empty(t, ev.Saves(), "Unexpected additional messages in save channel")

	assertInvalidationChanReceive(t, ev.Invalidations(), map[config.DataType][]string{
		config.RequestDataType: {"request2"},
		config.ImpDataType:     {"imp1"},
		config.AccountDataType: {"acc1"},
	})
	assert.Empty(t, ev.Invalidations(), "Unexpected additional messages in save channel")
}

func TestErrorResponse(t *testing.T) {
	handler := &mockResponseHandler{
		statusCode: httpCore.StatusInternalServerError,
		response:   "Something horrible happened.",
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	assert.Empty(t, ev.Saves(), "No saves should be emitted if the HTTP call fails. Got %d", len(ev.Saves()))
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
	assert.Empty(t, ev.Saves(), "No saves should be emitted if the HTTP call times out. Got %d", len(ev.Saves()))
}

func TestMalformedResponse(t *testing.T) {
	handler := &mockResponseHandler{
		statusCode: httpCore.StatusOK,
		response:   "This isn't JSON.",
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ev := NewHTTPEvents(server.Client(), server.URL, nil, -1)
	assert.Empty(t, ev.Saves(), "No saves should be emitted if the HTTP call fails. Got %d", len(ev.Saves()))
}

type mockResponseHandler struct {
	statusCode int
	response   string
}

func (m *mockResponseHandler) ServeHTTP(rw httpCore.ResponseWriter, r *httpCore.Request) {
	rw.WriteHeader(m.statusCode)
	rw.Write([]byte(m.response))
}

func assertSaveChanReceive(t *testing.T, ch <-chan events.Save, expected map[config.DataType]map[string]json.RawMessage) {
	t.Helper()
	for len(expected) > 0 {
		select {
		case event := <-ch:
			if data, ok := expected[event.DataType]; ok {
				assert.Equal(t, data, event.Data)
				delete(expected, event.DataType)
			}
		case <-time.After(20 * time.Millisecond):
			assert.FailNow(t, "Did not receive all expected messages in time", "%v", expected)
		}
	}
}

func assertInvalidationChanReceive(t *testing.T, ch <-chan events.Invalidation, expected map[config.DataType][]string) {
	t.Helper()
	for len(expected) > 0 {
		select {
		case event := <-ch:
			if data, ok := expected[event.DataType]; ok {
				assert.Equal(t, data, event.Data)
				delete(expected, event.DataType)
			}
		case <-time.After(20 * time.Millisecond):
			assert.FailNow(t, "Did not receive all expected messages in time", "%v", expected)
		}
	}
}
