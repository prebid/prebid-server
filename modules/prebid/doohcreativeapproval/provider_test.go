package doohcreativeapproval

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPApprovalProviderLookup(t *testing.T) {
	var gotRequest approvalRequest
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "secret", r.Header.Get("X-Test-Auth"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotRequest))

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"creatives":[{"creative_approval_id":"v1:approved","status":"approved"}]}`)),
			Header:     make(http.Header),
		}, nil
	})}

	cfg := testModuleConfig()
	cfg.Endpoint = "http://approval.example.com"
	cfg.Headers = map[string]string{"X-Test-Auth": "secret"}
	provider := newHTTPApprovalProvider(client)

	statuses, warnings, err := provider.Lookup(context.Background(), cfg, "acct", []creativeApproval{{
		CreativeApprovalID: "v1:approved",
		Bidder:             "appnexus",
		CreativeID:         "cr-123",
	}})

	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, "acct", gotRequest.AccountID)
	require.Len(t, gotRequest.Creatives, 1)
	assert.Equal(t, "v1:approved", gotRequest.Creatives[0].CreativeApprovalID)
	assert.Equal(t, map[string]approvalStatus{"v1:approved": approvalStatusApproved}, statuses)
}

func TestHTTPApprovalProviderLookupErrors(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		expectedErr string
	}{
		{
			name: "non-2xx",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "nope", http.StatusInternalServerError)
			},
			expectedErr: "approval endpoint returned status 500",
		},
		{
			name: "malformed-json",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`not-json`))
			},
			expectedErr: "failed to parse approval response",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := testModuleConfig()
			cfg.Endpoint = "http://approval.example.com"
			provider := newHTTPApprovalProvider(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				recorder := &responseRecorder{header: make(http.Header)}
				test.handler(recorder, r)
				return recorder.response(), nil
			})})

			_, _, err := provider.Lookup(context.Background(), cfg, "acct", []creativeApproval{{CreativeApprovalID: "v1:creative"}})

			require.Error(t, err)
			assert.Contains(t, err.Error(), test.expectedErr)
		})
	}
}

func TestHTTPApprovalProviderTimeout(t *testing.T) {
	cfg := testModuleConfig()
	cfg.Endpoint = "http://approval.example.com"
	cfg.TimeoutMS = 1
	provider := newHTTPApprovalProvider(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		select {
		case <-time.After(50 * time.Millisecond):
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	})})

	_, _, err := provider.Lookup(context.Background(), cfg, "acct", []creativeApproval{{CreativeApprovalID: "v1:creative"}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute approval request")
}

func TestParseApprovalResponseWarnings(t *testing.T) {
	responseBody := []byte(`{
		"creatives": [
			{"creative_approval_id": "", "status": "approved"},
			{"creative_approval_id": "v1:unknown", "status": "approved"},
			{"creative_approval_id": "v1:bad-status", "status": "maybe"},
			{"creative_approval_id": "v1:approved", "status": "approved"},
			{"creative_approval_id": "v1:approved", "status": "rejected"}
		]
	}`)
	var response approvalResponse
	require.NoError(t, jsonutil.UnmarshalValid(responseBody, &response))

	statuses, warnings, err := parseApprovalResponse(response, []creativeApproval{
		{CreativeApprovalID: "v1:approved"},
		{CreativeApprovalID: "v1:bad-status"},
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]approvalStatus{"v1:approved": approvalStatusApproved}, statuses)
	assert.Len(t, warnings, 4)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type responseRecorder struct {
	statusCode int
	header     http.Header
	body       strings.Builder
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	return r.body.Write(data)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *responseRecorder) response() *http.Response {
	statusCode := r.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     r.header,
		Body:       io.NopCloser(strings.NewReader(r.body.String())),
	}
}
