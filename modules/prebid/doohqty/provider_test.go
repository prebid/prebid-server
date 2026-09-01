package doohqty

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPValueProviderLookupResponseWarnings(t *testing.T) {
	provider := newHTTPValueProvider(&http.Client{Transport: doohQtyRoundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"values": [
					{"path": "", "key": "missing-path", "multiplier": 1},
					{"path": "dooh.id", "key": "unrequested", "multiplier": 1},
					{"path": "dooh.id", "key": "screen-1", "multiplier": 2},
					{"path": "dooh.id", "key": "screen-1", "multiplier": 3}
				]
			}`)),
		}, nil
	})})
	cfg := defaultModuleConfig()
	cfg.Source.Endpoint = "https://values.example.com/lookup"
	lookup := lookupKey{AccountID: testAccountID, Path: lookupPathDOOHID, Key: "screen-1"}

	values, warnings, err := provider.Lookup(context.Background(), cfg, testAccountID, []lookupKey{lookup})

	require.NoError(t, err)
	assert.Equal(t, 2.0, values[lookup].Multiplier)
	require.Len(t, warnings, 3)
	assert.Contains(t, warnings[0], "path or key is empty")
	assert.Contains(t, warnings[1], "was not requested")
	assert.Contains(t, warnings[2], "duplicate lookup response value")
}

func TestHTTPValueProviderLookupErrors(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		statusCode  int
		expectedErr string
	}{
		{
			name:        "non-2xx",
			body:        "nope",
			statusCode:  http.StatusInternalServerError,
			expectedErr: "lookup endpoint returned status 500",
		},
		{
			name:        "malformed-json",
			body:        "not-json",
			statusCode:  http.StatusOK,
			expectedErr: "failed to parse lookup response",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := newHTTPValueProvider(&http.Client{Transport: doohQtyRoundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: test.statusCode,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(test.body)),
				}, nil
			})})
			cfg := defaultModuleConfig()
			cfg.Source.Endpoint = "https://values.example.com/lookup"

			_, _, err := provider.Lookup(context.Background(), cfg, testAccountID, []lookupKey{{Path: lookupPathDOOHID, Key: "screen-1"}})

			require.Error(t, err)
			assert.Contains(t, err.Error(), test.expectedErr)
		})
	}
}

func TestHTTPValueProviderTimeout(t *testing.T) {
	provider := newHTTPValueProvider(&http.Client{Transport: doohQtyRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		select {
		case <-time.After(50 * time.Millisecond):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}, nil
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	})})
	cfg := defaultModuleConfig()
	cfg.Source.Endpoint = "https://values.example.com/lookup"
	cfg.TimeoutMS = 1

	_, _, err := provider.Lookup(context.Background(), cfg, testAccountID, []lookupKey{{Path: lookupPathDOOHID, Key: "screen-1"}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute lookup request")
}

func TestBulkLookupRequestOmitsLookupAccountID(t *testing.T) {
	var received bulkLookupRequest
	provider := newHTTPValueProvider(&http.Client{Transport: doohQtyRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"values":[]}`)),
		}, nil
	})})
	cfg := defaultModuleConfig()
	cfg.Source.Endpoint = "https://values.example.com/lookup"

	_, _, err := provider.Lookup(context.Background(), cfg, testAccountID, []lookupKey{{AccountID: testAccountID, Path: lookupPathDOOHID, Key: "screen-1"}})

	require.NoError(t, err)
	assert.Equal(t, testAccountID, received.AccountID)
	require.Len(t, received.Lookups, 1)
	assert.Empty(t, received.Lookups[0].AccountID)
}

type doohQtyRoundTripFunc func(*http.Request) (*http.Response, error)

func (f doohQtyRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
