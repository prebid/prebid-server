package source

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPSourceFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "one", r.URL.Query().Get("id"))
		_, _ = w.Write([]byte(`{"id":"one"}`))
	}))
	defer server.Close()

	source, err := NewHTTPSource(server.Client(), server.URL, testRequestBuilder, rawResponseDecoder)
	require.NoError(t, err)

	raw, found, err := source.Fetch(context.Background(), "one")
	require.NoError(t, err)
	require.True(t, found)
	assert.JSONEq(t, `{"id":"one"}`, string(raw))
}

func TestHTTPSourceNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	source, err := NewHTTPSource(server.Client(), server.URL, testRequestBuilder, rawResponseDecoder)
	require.NoError(t, err)

	raw, found, err := source.Fetch(context.Background(), "missing")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, raw)
}

func TestNewHTTPSourceValidatesDependencies(t *testing.T) {
	client := &http.Client{}

	_, err := NewHTTPSource[string](nil, "http://example.com", testRequestBuilder, rawResponseDecoder)
	require.EqualError(t, err, "HTTP client is required")

	_, err = NewHTTPSource[string](client, "http://example.com", nil, rawResponseDecoder)
	require.EqualError(t, err, "HTTP request builder is required")

	_, err = NewHTTPSource[string](client, "http://example.com", testRequestBuilder, nil)
	require.EqualError(t, err, "HTTP response decoder is required")

	_, err = NewHTTPSource(client, "://bad-url", testRequestBuilder, rawResponseDecoder)
	require.ErrorContains(t, err, "invalid HTTP endpoint")
}

func TestHTTPSourceFetchErrors(t *testing.T) {
	expectedErr := errors.New("expected")
	testCases := []struct {
		name          string
		client        *http.Client
		buildRequest  RequestBuilder[string]
		decode        ResponseDecoder[string]
		expectedError string
	}{
		{
			name: "Request Builder",
			buildRequest: func(context.Context, *url.URL, string) (*http.Request, error) {
				return nil, expectedErr
			},
			decode:        rawResponseDecoder,
			expectedError: "build HTTP request for key one: expected",
		},
		{
			name:          "Transport",
			client:        &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, expectedErr })},
			decode:        rawResponseDecoder,
			expectedError: "fetch key one via HTTP: Get \"http://example.com?id=one\": expected",
		},
		{
			name: "Response Body",
			client: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: errorReader{err: expectedErr}}, nil
			})},
			decode:        rawResponseDecoder,
			expectedError: "read HTTP response for key one: expected",
		},
		{
			name: "Unexpected Status",
			client: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(""))}, nil
			})},
			decode:        rawResponseDecoder,
			expectedError: "fetch key one via HTTP: unexpected response status 500",
		},
		{
			name: "Response Decoder",
			client: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
			})},
			decode: func(string, []byte) (json.RawMessage, bool, error) {
				return nil, false, expectedErr
			},
			expectedError: "decode HTTP response for key one: expected",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := test.client
			if client == nil {
				client = &http.Client{}
			}
			buildRequest := test.buildRequest
			if buildRequest == nil {
				buildRequest = testRequestBuilder
			}
			source, err := NewHTTPSource(client, "http://example.com", buildRequest, test.decode)
			require.NoError(t, err)

			raw, found, err := source.Fetch(context.Background(), "one")

			require.EqualError(t, err, test.expectedError)
			assert.False(t, found)
			assert.Nil(t, raw)
		})
	}
}

func TestHTTPSourcePreservesDecoderNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	source, err := NewHTTPSource(server.Client(), server.URL, testRequestBuilder, func(string, []byte) (json.RawMessage, bool, error) {
		return nil, false, nil
	})
	require.NoError(t, err)

	raw, found, err := source.Fetch(context.Background(), "missing")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, raw)
}

func testRequestBuilder(ctx context.Context, endpoint *url.URL, key string) (*http.Request, error) {
	q := endpoint.Query()
	q.Set("id", key)
	endpoint.RawQuery = q.Encode()
	return http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
}

func rawResponseDecoder(_ string, body []byte) (json.RawMessage, bool, error) {
	return json.RawMessage(body), true, nil
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func (r errorReader) Close() error {
	return nil
}
