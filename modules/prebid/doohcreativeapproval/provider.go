package doohcreativeapproval

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type approvalProvider interface {
	Lookup(context.Context, moduleConfig, string, []creativeApproval) (map[string]approvalStatus, []string, error)
}

type httpApprovalProvider struct {
	client *http.Client
}

func newHTTPApprovalProvider(client *http.Client) *httpApprovalProvider {
	return &httpApprovalProvider{client: client}
}

func (p *httpApprovalProvider) Lookup(ctx context.Context, cfg moduleConfig, accountID string, creatives []creativeApproval) (map[string]approvalStatus, []string, error) {
	if len(creatives) == 0 {
		return nil, nil, nil
	}

	requestPayload := approvalRequest{
		AccountID: accountID,
		Creatives: creatives,
	}
	body, err := jsonutil.Marshal(requestPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal approval request: %s", err)
	}

	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutMS)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build approval request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for name, value := range cfg.Headers {
		req.Header.Set(name, value)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute approval request: %s", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read approval response: %s", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("approval endpoint returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	var response approvalResponse
	if err := jsonutil.UnmarshalValid(responseBody, &response); err != nil {
		return nil, nil, fmt.Errorf("failed to parse approval response: %s", err)
	}

	return parseApprovalResponse(response, creatives)
}

func parseApprovalResponse(response approvalResponse, requestedCreatives []creativeApproval) (map[string]approvalStatus, []string, error) {
	requested := make(map[string]struct{}, len(requestedCreatives))
	for _, creative := range requestedCreatives {
		requested[creative.CreativeApprovalID] = struct{}{}
	}

	statuses := make(map[string]approvalStatus, len(response.Creatives))
	warnings := make([]string, 0)
	for _, creative := range response.Creatives {
		if creative.CreativeApprovalID == "" {
			warnings = append(warnings, "approval response creative skipped because creative_approval_id is empty")
			continue
		}
		if _, ok := requested[creative.CreativeApprovalID]; !ok {
			warnings = append(warnings, fmt.Sprintf("approval response creative skipped because creative_approval_id %q was not requested", creative.CreativeApprovalID))
			continue
		}
		if !isValidApprovalStatus(creative.Status) {
			warnings = append(warnings, fmt.Sprintf("approval response creative skipped because status %q is not supported", creative.Status))
			continue
		}
		if _, exists := statuses[creative.CreativeApprovalID]; exists {
			warnings = append(warnings, fmt.Sprintf("approval response duplicate creative skipped for creative_approval_id %q", creative.CreativeApprovalID))
			continue
		}

		statuses[creative.CreativeApprovalID] = creative.Status
	}

	return statuses, warnings, nil
}
