package tmp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/require"
)

// asyncRequestKey aliases the module-internal constant.
const asyncRequestKey = moduleContextAsyncKey

func TestHandleEntrypointHook_StoresAsyncRequest(t *testing.T) {
	mod, err := Builder(json.RawMessage(`{"router_url":"https://r","seller_agent_url":"https://us"}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	require.NoError(t, err)
	m := mod.(*Module)

	miCtx := hookstage.ModuleInvocationContext{}
	payload := hookstage.EntrypointPayload{Request: httptest.NewRequest("POST", "/openrtb2/auction", nil)}
	result, err := m.HandleEntrypointHook(context.Background(), miCtx, payload)
	require.NoError(t, err)
	require.NotNil(t, result.ModuleContext)

	stored, ok := result.ModuleContext.Get(asyncRequestKey)
	require.True(t, ok)
	_, isAR := stored.(*AsyncRequest)
	require.True(t, isAR)
}

func TestBuilder_EmptyConfig(t *testing.T) {
	m, err := Builder(json.RawMessage(`{}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "router_url is required")
	require.Nil(t, m)
}

func TestBuilder_Validation(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantError string
	}{
		{
			name:      "missing router_url",
			config:    `{"seller_agent_url":"https://example.com"}`,
			wantError: "router_url is required",
		},
		{
			name:      "missing seller_agent_url",
			config:    `{"router_url":"https://tmp.interchange.io"}`,
			wantError: "seller_agent_url is required",
		},
		{
			name:      "too many preserve_eids",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","masking":{"enabled":true,"user":{"preserve_eids":["a","b","c","d"]}}}`,
			wantError: "preserve_eids exceeds spec limit of 3 entries",
		},
		{
			name:      "negative lat_long_precision",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","masking":{"enabled":true,"geo":{"lat_long_precision":-1}}}`,
			wantError: "lat_long_precision cannot be negative",
		},
		{
			name:      "lat_long_precision over 4",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","masking":{"enabled":true,"geo":{"lat_long_precision":5}}}`,
			wantError: "lat_long_precision cannot exceed 4 decimal places for privacy protection",
		},
		{
			name:      "negative timeout_ms",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","timeout_ms":-1}`,
			wantError: "timeout_ms must be positive",
		},
		{
			name:   "valid minimal config",
			config: `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deps := moduledeps.ModuleDeps{HTTPClient: &http.Client{}}
			m, err := Builder(json.RawMessage(tc.config), deps)
			if tc.wantError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantError)
				require.Nil(t, m)
			} else {
				require.NoError(t, err)
				require.NotNil(t, m)
			}
		})
	}
}

func TestHandleProcessedAuctionHook_KicksOffGoroutine(t *testing.T) {
	var ctxHit atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tmp/context" {
			ctxHit.Store(true)
		}
		var rid string
		if r.URL.Path == "/tmp/context" {
			var req ContextMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			rid = req.RequestID
		} else {
			var req IdentityMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			rid = req.RequestID
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "x", "request_id": rid, "offers": []any{}, "eligible_package_ids": []any{}})
	}))
	defer srv.Close()

	mod, _ := Builder(json.RawMessage(`{"router_url":"`+srv.URL+`","seller_agent_url":"https://us","masking":{"enabled":false}}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(context.Background())
	ar.module = m
	mc.Set(moduleContextAsyncKey, ar)

	br := &openrtb2.BidRequest{
		ID:   "a",
		Imp:  []openrtb2.Imp{{ID: "i", TagID: "h"}},
		Site: &openrtb2.Site{Domain: "x.com"},
	}
	miCtx := hookstage.ModuleInvocationContext{
		ModuleContext: mc,
		AccountConfig: json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"p"}}}}`),
	}
	payload := hookstage.ProcessedAuctionRequestPayload{Request: &openrtb_ext.RequestWrapper{BidRequest: br}}
	_, err := m.HandleProcessedAuctionHook(context.Background(), miCtx, payload)
	require.NoError(t, err)

	<-ar.done
	require.True(t, ctxHit.Load(), "context endpoint was called from the goroutine")
}
