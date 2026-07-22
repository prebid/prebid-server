package doohqty

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type valueProvider interface {
	Lookup(ctx context.Context, cfg moduleConfig, accountID string, lookups []lookupKey) (map[lookupKey]impressionValue, []string, error)
}

type httpValueProvider struct {
	client *http.Client
}

type bulkLookupRequest struct {
	AccountID string      `json:"account_id"`
	Lookups   []lookupKey `json:"lookups"`
}

type bulkLookupResponse struct {
	Values []impressionValue `json:"values"`
}

func newHTTPValueProvider(client *http.Client) *httpValueProvider {
	return &httpValueProvider{
		client: client,
	}
}

func (p *httpValueProvider) Lookup(ctx context.Context, cfg moduleConfig, accountID string, lookups []lookupKey) (map[lookupKey]impressionValue, []string, error) {
	if len(lookups) == 0 {
		return nil, nil, nil
	}

	requestPayload := bulkLookupRequest{
		AccountID: accountID,
		Lookups:   lookups,
	}
	body, err := jsonutil.Marshal(requestPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal lookup request: %s", err)
	}

	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutMS)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, cfg.Source.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build lookup request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for name, value := range cfg.Source.Headers {
		req.Header.Set(name, value)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute lookup request: %s", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read lookup response: %s", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("lookup endpoint returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	var response bulkLookupResponse
	if err := jsonutil.UnmarshalValid(responseBody, &response); err != nil {
		return nil, nil, fmt.Errorf("failed to parse lookup response: %s", err)
	}

	values := make(map[lookupKey]impressionValue, len(response.Values))
	warnings := make([]string, 0)
	requested := make(map[lookupKey]struct{}, len(lookups))
	for _, lookup := range lookups {
		requested[lookup] = struct{}{}
	}

	for _, value := range response.Values {
		key := lookupKey{
			AccountID: accountID,
			Path:      value.Path,
			Key:       value.Key,
		}
		if value.Path == "" || value.Key == "" {
			warnings = append(warnings, "lookup response value skipped because path or key is empty")
			continue
		}
		if _, ok := requested[key]; !ok {
			warnings = append(warnings, fmt.Sprintf("lookup response value skipped because %s=%q was not requested", value.Path, value.Key))
			continue
		}
		if _, exists := values[key]; exists {
			warnings = append(warnings, fmt.Sprintf("duplicate lookup response value skipped for %s=%q", value.Path, value.Key))
			continue
		}

		values[key] = value
	}

	return values, warnings, nil
}
