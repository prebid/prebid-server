package tmp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/adcontextprotocol/adcp-go/tmproto"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// callContext signs and POSTs a ContextMatch request to the provider's context
// endpoint. Signatures are computed per-provider-endpoint per the TMP spec.
func (m *Module) callContext(ctx context.Context, p ProviderConfig, req *tmproto.ContextMatchRequest) (*tmproto.ContextMatchResponse, error) {
	epoch := tmproto.CurrentEpoch()
	endpoint := tmproto.NormalizeProviderEndpointURL(p.ContextURL)
	sig := m.signer.SignContextMatch(req, endpoint, epoch)

	raw, err := jsonutil.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("context marshal: %w", err)
	}

	body, err := m.doTMP(ctx, p.ContextURL, raw, sig)
	if err != nil {
		return nil, err
	}
	var resp tmproto.ContextMatchResponse
	if err := jsonutil.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("context decode: %w", err)
	}
	return &resp, nil
}

// callIdentity signs and POSTs an IdentityMatch request to the provider's
// identity endpoint. The wire request keeps the Country field, but signing
// strips it via BuildIdentityMatchSigningInput's canonical form.
//
// The provider→router hop returns ProviderIdentityMatchResponse (eligibility
// plus provider-local TMPX chunks). This module IS the router — publisher-
// facing shape reassembly happens locally via the TmpxMacroMapping config.
func (m *Module) callIdentity(ctx context.Context, p ProviderConfig, req *tmproto.IdentityMatchRequest) (*tmproto.ProviderIdentityMatchResponse, error) {
	epoch := tmproto.CurrentEpoch()
	endpoint := tmproto.NormalizeProviderEndpointURL(p.IdentityURL)
	sig, err := m.signer.SignIdentityMatch(req, endpoint, epoch)
	if err != nil {
		return nil, fmt.Errorf("identity sign: %w", err)
	}

	raw, err := jsonutil.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("identity marshal: %w", err)
	}

	body, err := m.doTMP(ctx, p.IdentityURL, raw, sig)
	if err != nil {
		return nil, err
	}
	if forbidden := findForbiddenProviderResponseFields(body); forbidden != "" {
		return nil, fmt.Errorf("identity response from %s carries forbidden provider-hop field %q; adcp provider-identity-match-response.json rejects router-hop and envelope-extension fields", p.Name, forbidden)
	}
	var resp tmproto.ProviderIdentityMatchResponse
	if err := jsonutil.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("identity decode: %w", err)
	}
	return &resp, nil
}

// forbiddenProviderResponseFields enumerates the top-level fields the
// provider→router shape MUST NOT carry per adcp
// provider-identity-match-response.json's `not: {anyOf: [...]}` clause.
// The router (this module) MUST reject any provider response that carries
// any of these — they belong on the router→publisher hop only, or, for
// `context` / `ext`, are explicitly forbidden to prevent envelope-level
// data leakage across the identity privacy boundary.
var forbiddenProviderResponseFields = []string{
	"tmpx_providers",
	"tmpx",
	"tmpx_values",
	"tmpx_macros",
	"context",
	"ext",
}

// findForbiddenProviderResponseFields returns the first top-level field
// name present in the raw JSON body that the provider→router response
// schema forbids, or "" when the body carries none. Uses a minimal
// map[string]json.RawMessage decode so it costs a single pass and does
// not require touching the strongly-typed response struct.
func findForbiddenProviderResponseFields(body []byte) string {
	var top map[string]json.RawMessage
	if err := jsonutil.Unmarshal(body, &top); err != nil {
		return ""
	}
	for _, f := range forbiddenProviderResponseFields {
		if _, ok := top[f]; ok {
			return f
		}
	}
	return ""
}

// doTMP sends a signed TMP request and returns the raw response body. A
// non-2xx response is surfaced as an error containing the provider's error
// envelope when parseable, so callers can distinguish "unknown package" from
// "provider unavailable".
func (m *Module) doTMP(ctx context.Context, url string, body []byte, signature string) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set(tmproto.HeaderTMPSignature, signature)
	httpReq.Header.Set(tmproto.HeaderTMPKeyID, m.signer.KeyID)

	resp, err := m.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Cap the response so a misbehaving upstream cannot exhaust memory on the
	// auction hot path. The TMP spec targets 200–600 B messages, so 1 MiB is
	// far more than any legitimate response.
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}

	var tmpErr tmproto.ErrorResponse
	if err := jsonutil.Unmarshal(respBody, &tmpErr); err == nil && tmpErr.Code != "" {
		return nil, fmt.Errorf("tmp status %d code=%s: %s", resp.StatusCode, tmpErr.Code, tmpErr.Message)
	}
	return nil, fmt.Errorf("tmp status %d", resp.StatusCode)
}
