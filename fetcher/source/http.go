package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/net/context/ctxhttp"
)

// RequestBuilder creates a request for one source key.
type RequestBuilder[K comparable] func(ctx context.Context, endpoint *url.URL, key K) (*http.Request, error)

// ResponseDecoder extracts one source value from a successful HTTP response.
type ResponseDecoder[K comparable] func(key K, body []byte) (raw json.RawMessage, found bool, err error)

// HTTPSource fetches keyed raw values over HTTP using caller-provided protocol
// functions.
type HTTPSource[K comparable] struct {
	client         *http.Client
	endpoint       *url.URL
	buildRequest   RequestBuilder[K]
	decodeResponse ResponseDecoder[K]
}

// NewHTTPSource builds a reusable HTTP source. Request construction and response
// decoding remain with the consuming data type.
func NewHTTPSource[K comparable](client *http.Client, endpoint string, buildRequest RequestBuilder[K], decodeResponse ResponseDecoder[K]) (*HTTPSource[K], error) {
	if client == nil {
		return nil, errors.New("HTTP client is required")
	}
	if buildRequest == nil {
		return nil, errors.New("HTTP request builder is required")
	}
	if decodeResponse == nil {
		return nil, errors.New("HTTP response decoder is required")
	}
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP endpoint %q: %w", endpoint, err)
	}
	return &HTTPSource[K]{
		client:         client,
		endpoint:       endpointURL,
		buildRequest:   buildRequest,
		decodeResponse: decodeResponse,
	}, nil
}

func (s *HTTPSource[K]) Fetch(ctx context.Context, key K) (json.RawMessage, bool, error) {
	endpoint := *s.endpoint
	req, err := s.buildRequest(ctx, &endpoint, key)
	if err != nil {
		return nil, false, fmt.Errorf("build HTTP request for key %v: %w", key, err)
	}
	resp, err := ctxhttp.Do(ctx, s.client, req)
	if err != nil {
		return nil, false, fmt.Errorf("fetch key %v via HTTP: %w", key, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("read HTTP response for key %v: %w", key, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("fetch key %v via HTTP: unexpected response status %d", key, resp.StatusCode)
	}

	raw, found, err := s.decodeResponse(key, body)
	if err != nil {
		return nil, false, fmt.Errorf("decode HTTP response for key %v: %w", key, err)
	}
	return raw, found, nil
}
