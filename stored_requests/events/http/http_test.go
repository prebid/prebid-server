package http

import (
	"context"
	"encoding/json"
	"fmt"
	httpCore "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ctxProducer() (context.Context, func()) {
	return context.WithTimeout(context.Background(), -1)
}

func TestStartup(t *testing.T) {
	type testStep struct {
		statusCode    int
		response      string
		timeout       bool
		saves         string
		invalidations string
	}
	testCases := []struct {
		description string
		tests       []testStep
	}{
		{
			description: "Load requests at startup",
			tests: []testStep{
				{
					statusCode: httpCore.StatusOK,
					response:   `{"requests": {"request1": {"value":1}, "request2": {"value":2}}}`,
					saves:      `{"requests": {"request1": {"value":1}, "request2": {"value":2}}, "imps": null, "accounts": null}`,
				},
			},
		},
		{
			description: "Load imps at startup",
			tests: []testStep{
				{
					statusCode: httpCore.StatusOK,
					response:   `{"imps": {"imp1": {"value":1}}}`,
					saves:      `{"imps": {"imp1": {"value":1}}, "requests": null, "accounts": null}`,
				},
			},
		},
		{
			description: "Load requests and imps then update",
			tests: []testStep{
				{
					statusCode: httpCore.StatusOK,
					response:   `{"requests": {"request1": {"value":1}, "request2": {"value":2}}, "imps": {"imp1": {"value":3}, "imp2": {"value":4}}}`,
					saves:      `{"requests": {"request1": {"value":1}, "request2": {"value":2}}, "imps": {"imp1": {"value":3}, "imp2": {"value":4}}, "accounts":null}`,
				},
				{
					statusCode:    httpCore.StatusOK,
					response:      `{"requests": {"request1": {"value":5}, "request2": {"deleted":true}}, "imps": {"imp1": {"deleted":true}, "imp2": {"value":6}}}`,
					saves:         `{"requests": {"request1": {"value":5}}, "imps": {"imp2": {"value":6}}, "accounts":null}`,
					invalidations: `{"requests": ["request2"], "imps": ["imp1"], "accounts": []}`,
				},
			},
		},
		{
			description: "Load accounts then update",
			tests: []testStep{
				{
					statusCode: httpCore.StatusOK,
					response:   `{"accounts":{"account1":{"value":1}, "account2":{"value":2}}}`,
					saves:      `{"accounts":{"account1":{"value":1}, "account2":{"value":2}}, "imps": null, "requests": null}`,
				},
				{
					statusCode:    httpCore.StatusOK,
					response:      `{"accounts":{"account1":{"value":5}, "account2":{"deleted": true}}}`,
					saves:         `{"accounts":{"account1":{"value":5}}, "imps": null, "requests": null}`,
					invalidations: `{"accounts":["account2"], "requests": [], "imps": []}`,
				},
			},
		},
		{
			description: "Load nothing at startup",
			tests: []testStep{
				{
					statusCode: httpCore.StatusOK,
					response:   `{}`,
				},
			},
		},
		{
			description: "Malformed response at startup",
			tests: []testStep{
				{
					statusCode: httpCore.StatusOK,
					response:   `{some bad json`,
				},
			},
		},
		{
			description: "Server error at startup",
			tests: []testStep{
				{
					statusCode: httpCore.StatusInternalServerError,
					response:   ``,
				},
			},
		},
		{
			description: "HTTP timeout error at startup",
			tests: []testStep{
				{
					timeout: true,
				},
			},
		},
	}
	for _, tests := range testCases {
		t.Run(tests.description, func(t *testing.T) {
			handler := &mockResponseHandler{}
			server := httptest.NewServer(handler)
			defer server.Close()

			var ev *HTTPEvents

			for i, test := range tests.tests {
				handler.statusCode = test.statusCode
				handler.response = test.response
				if i == 0 { // NewHTTPEvents() calls the API immediately
					if test.timeout {
						ev = NewHTTPEvents(server.Client(), server.URL, ctxProducer, -1) // force timeout
					} else {
						ev = NewHTTPEvents(server.Client(), server.URL, nil, -1)
					}
				} else { // Second test triggers API call by initiating a 1s refresh loop
					timeChan := make(chan time.Time, 1)
					timeChan <- time.Now()
					go ev.refresh(timeChan)
				}
				t.Run(fmt.Sprintf("Step %d", i+1), func(t *testing.T) {
					// Check expected Saves
					if len(test.saves) > 0 {
						saves, err := json.Marshal(<-ev.Saves())
						assert.NoError(t, err, `Failed to marshal event.Save object: %v`, err)
						assert.JSONEq(t, test.saves, string(saves))
					}
					assert.Empty(t, ev.Saves(), "Unexpected additional messages in save channel")
					// Check expected Invalidations
					if len(test.invalidations) > 0 {
						invalidations, err := json.Marshal(<-ev.Invalidations())
						assert.NoError(t, err, `Failed to marshal event.Invalidation object: %v`, err)
						assert.JSONEq(t, test.invalidations, string(invalidations))
					}
					assert.Empty(t, ev.Invalidations(), "Unexpected additional messages in invalidations channel")
				})
			}
		})
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
