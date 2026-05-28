package tmp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// asyncRequestKey aliases the module-internal constant.
const asyncRequestKey = moduleContextAsyncKey

func TestHandleEntrypointHook_StoresAsyncRequest(t *testing.T) {
	mod, err := Builder(json.RawMessage(`{"router_url":"https://r"}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
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
			config:    `{}`,
			wantError: "router_url is required",
		},
		{
			name:      "too many preserve_eids",
			config:    `{"router_url":"https://tmp.interchange.io","masking":{"enabled":true,"user":{"preserve_eids":["a","b","c","d"]}}}`,
			wantError: "preserve_eids exceeds spec limit of 3 entries",
		},
		{
			name:      "negative lat_long_precision",
			config:    `{"router_url":"https://tmp.interchange.io","masking":{"enabled":true,"geo":{"lat_long_precision":-1}}}`,
			wantError: "lat_long_precision cannot be negative",
		},
		{
			name:      "lat_long_precision over 4",
			config:    `{"router_url":"https://tmp.interchange.io","masking":{"enabled":true,"geo":{"lat_long_precision":5}}}`,
			wantError: "lat_long_precision cannot exceed 4 decimal places for privacy protection",
		},
		{
			name:      "negative timeout_ms",
			config:    `{"router_url":"https://tmp.interchange.io","timeout_ms":-1}`,
			wantError: "timeout_ms must be positive",
		},
		{
			name:   "valid minimal config",
			config: `{"router_url":"https://tmp.interchange.io"}`,
		},
		{
			name:      "router_url not a valid URL",
			config:    `{"router_url":"::not a url"}`,
			wantError: "router_url",
		},
		{
			name:      "router_url http rejected",
			config:    `{"router_url":"http://example.com"}`,
			wantError: "must use https",
		},
		{
			name:      "router_url http localhost allowed",
			config:    `{"router_url":"http://127.0.0.1:8080"}`,
			wantError: "",
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

	mod, _ := Builder(json.RawMessage(`{"router_url":"`+srv.URL+`","masking":{"enabled":false}}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(context.Background())
	ar.module = m
	mc.Set(moduleContextAsyncKey, ar)

	impExt := json.RawMessage(`{"prebid":{"modules":{"scope3":{"tmp":{"placement_id":"p"}}}}}`)
	br := &openrtb2.BidRequest{
		ID:  "a",
		Imp: []openrtb2.Imp{{ID: "i", Ext: impExt}},
		Site: &openrtb2.Site{Domain: "x.com"},
		Ext: json.RawMessage(`{"prebid":{"modules":{"scope3":{"tmp":{"property_rid":"r"}}}}}`),
	}
	miCtx := hookstage.ModuleInvocationContext{
		ModuleContext: mc,
	}
	payload := hookstage.ProcessedAuctionRequestPayload{Request: &openrtb_ext.RequestWrapper{BidRequest: br}}
	_, err := m.HandleProcessedAuctionHook(context.Background(), miCtx, payload)
	require.NoError(t, err)

	<-ar.done
	require.True(t, ctxHit.Load(), "context endpoint was called from the goroutine")
}

func TestHandleAuctionResponseHook_WritesPerBidExt(t *testing.T) {
	mod, _ := Builder(json.RawMessage(`{"router_url":"https://r","add_to_targeting":true}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.done = make(chan struct{})
	close(ar.done)
	ar.result = &AsyncResult{
		PerPlacement: map[string]PlacementResult{
			"header_728x90": {EligiblePackages: []string{"pkg_abc"}, TargetingKVs: []KeyValuePair{{Key: "buyer_kv", Value: "v1"}}, Segments: []string{"seg_a"}},
		},
		ImpToPlacement: map[string]string{"imp1": "header_728x90"},
		TMPX:           "k1.token",
	}
	mc.Set(moduleContextAsyncKey, ar)

	resp := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{ID: "b1", ImpID: "imp1", Ext: json.RawMessage(`{}`)}},
		}},
		Ext: json.RawMessage(`{}`),
	}
	payload := hookstage.AuctionResponsePayload{BidResponse: resp}
	miCtx := hookstage.ModuleInvocationContext{ModuleContext: mc}

	result, err := m.HandleAuctionResponseHook(context.Background(), miCtx, payload)
	require.NoError(t, err)

	// Apply the mutations from the ChangeSet, like Prebid does in production.
	for _, mut := range result.ChangeSet.Mutations() {
		payload, _ = mut.Apply(payload)
	}

	respExt := gjson.GetBytes(payload.BidResponse.Ext, "scope3.tmp.tmpx")
	require.Equal(t, "k1.token", respExt.String())

	bidExt := payload.BidResponse.SeatBid[0].Bid[0].Ext
	require.Equal(t, "header_728x90", gjson.GetBytes(bidExt, "scope3.tmp.placement_id").String())
	require.Equal(t, "pkg_abc", gjson.GetBytes(bidExt, "scope3.tmp.eligible_packages.0").String())
	require.Equal(t, "k1.token", gjson.GetBytes(bidExt, "prebid.targeting.TMPX").String())
	require.Equal(t, "v1", gjson.GetBytes(bidExt, "prebid.targeting.buyer_kv").String())
}

func TestHandleAuctionResponseHook_PartialFailureNoMutation(t *testing.T) {
	mod, _ := Builder(json.RawMessage(`{"router_url":"https://r"}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.done = make(chan struct{})
	close(ar.done)
	ar.err = errors.New("identity: failed")
	mc.Set(moduleContextAsyncKey, ar)

	resp := &openrtb2.BidResponse{Ext: json.RawMessage(`{}`)}
	payload := hookstage.AuctionResponsePayload{BidResponse: resp}
	miCtx := hookstage.ModuleInvocationContext{ModuleContext: mc}

	result, _ := m.HandleAuctionResponseHook(context.Background(), miCtx, payload)
	require.Empty(t, result.ChangeSet.Mutations(), "P1 strict: no mutation on error")
}

func TestOutboundWireShape_PrivacyGuarantees(t *testing.T) {
	var contextBody, identityBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/tmp/context":
			contextBody = buf
		case "/tmp/identity":
			identityBody = buf
		}
		_, _ = w.Write([]byte(`{"type":"x","request_id":"","offers":[],"eligible_package_ids":[]}`))
	}))
	defer srv.Close()

	mod, _ := Builder(json.RawMessage(`{
		"router_url":"`+srv.URL+`",
		"masking":{"enabled":true,"user":{"preserve_eids":["liveramp.com"]}}
	}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	impExt := json.RawMessage(`{"prebid":{"modules":{"scope3":{"tmp":{"placement_id":"p"}}}}}`)
	br := &openrtb2.BidRequest{
		ID:  "a",
		Imp: []openrtb2.Imp{{ID: "i1", Ext: impExt}},
		Site: &openrtb2.Site{Domain: "x.com"},
		Device: &openrtb2.Device{IP: "1.2.3.4", IFA: "AAA-BBB", Geo: &openrtb2.Geo{Country: "USA"}},
		User: &openrtb2.User{
			ID:       "uid",
			BuyerUID: "buid",
			Ext:      json.RawMessage(`{"eids":[{"source":"liveramp.com","uids":[{"id":"R1"}]},{"source":"criteo.com","uids":[{"id":"DROP"}]}]}`),
		},
		Ext: json.RawMessage(`{"prebid":{"modules":{"scope3":{"tmp":{"property_rid":"r"}}}}}`),
	}

	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.fetchAsync(br, nil, br.Ext)
	<-ar.done

	require.NotEmpty(t, contextBody)
	require.NotEmpty(t, identityBody)

	require.NotContains(t, string(contextBody), `"ip":"1.2.3.4"`)
	require.NotContains(t, string(contextBody), `"ifa":"AAA-BBB"`)
	require.NotContains(t, string(contextBody), `"id":"uid"`)
	require.NotContains(t, string(identityBody), `"ip":"1.2.3.4"`)
	require.NotContains(t, string(identityBody), `"package_ids"`)
	require.NotContains(t, string(identityBody), `"criteo.com"`)
	require.Contains(t, string(identityBody), `"country":"US"`)

	// seller_agent_url must be absent (router fills it in)
	require.NotContains(t, string(identityBody), `"seller_agent_url"`)
	// property_type must be absent (router fills it in)
	require.NotContains(t, string(contextBody), `"property_type"`)

	ctxID := gjson.GetBytes(contextBody, "request_id").String()
	idID := gjson.GetBytes(identityBody, "request_id").String()
	require.NotEqual(t, ctxID, idID, "context and identity request_ids MUST NOT correlate")
}

func TestEndToEnd_SuccessTMPXOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tmp/context":
			data, _ := os.ReadFile("testdata/context_response_empty.json")
			_, _ = w.Write(data)
		case "/tmp/identity":
			data, _ := os.ReadFile("testdata/identity_response_with_tmpx_only.json")
			_, _ = w.Write(data)
		}
	}))
	defer srv.Close()

	mod, _ := Builder(json.RawMessage(`{"router_url":"`+srv.URL+`","masking":{"enabled":false}}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	brData, _ := os.ReadFile("testdata/bid_request_multi_imp_three_placements.json")
	var br openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(brData, &br))

	// property_rid comes from auction ext; placement_ids come from per-imp ext.
	requestExt := json.RawMessage(`{"prebid":{"modules":{"scope3":{"tmp":{"property_rid":"01916f3a-9c4e-7000-8000-000000000010"}}}}}`)
	br.Ext = requestExt

	// Set placement_id on each imp's ext.
	placements := []string{"header_728x90", "sidebar_300x250", "preroll_video"}
	for i := range br.Imp {
		impExtJSON, _ := json.Marshal(map[string]any{
			"prebid": map[string]any{
				"modules": map[string]any{
					"scope3": map[string]any{
						"tmp": map[string]any{
							"placement_id": placements[i],
						},
					},
				},
			},
		})
		br.Imp[i].Ext = impExtJSON
	}

	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.fetchAsync(&br, nil, requestExt)
	<-ar.done

	require.NoError(t, ar.err)
	require.NotNil(t, ar.result)
	require.Equal(t, "k1.tokenABC", ar.result.TMPX, "TMPX emitted even when intersection is empty")
	for _, pr := range ar.result.PerPlacement {
		require.Empty(t, pr.EligiblePackages, "intersection is empty")
	}
}

func TestHandleAuctionResponseHook_RepeatedTargetingKVsBecomeArray(t *testing.T) {
	mod, _ := Builder(json.RawMessage(`{"router_url":"https://r","add_to_targeting":true}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.done = make(chan struct{})
	close(ar.done)
	ar.result = &AsyncResult{
		PerPlacement: map[string]PlacementResult{
			"p1": {
				TargetingKVs: []KeyValuePair{
					{Key: "adcp_pkg", Value: "pkg1"},
					{Key: "adcp_pkg", Value: "pkg2"},
					{Key: "single", Value: "v"},
				},
			},
		},
		ImpToPlacement: map[string]string{"imp1": "p1"},
	}
	mc.Set(moduleContextAsyncKey, ar)

	resp := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{ID: "b1", ImpID: "imp1", Ext: json.RawMessage(`{}`)}},
		}},
		Ext: json.RawMessage(`{}`),
	}
	payload := hookstage.AuctionResponsePayload{BidResponse: resp}
	miCtx := hookstage.ModuleInvocationContext{ModuleContext: mc}

	result, _ := m.HandleAuctionResponseHook(context.Background(), miCtx, payload)
	for _, mut := range result.ChangeSet.Mutations() {
		payload, _ = mut.Apply(payload)
	}

	bidExt := payload.BidResponse.SeatBid[0].Bid[0].Ext
	pkgArr := gjson.GetBytes(bidExt, "prebid.targeting.adcp_pkg")
	require.True(t, pkgArr.IsArray(), "repeated key should be emitted as JSON array")
	require.Equal(t, []string{"pkg1", "pkg2"}, []string{pkgArr.Array()[0].String(), pkgArr.Array()[1].String()})
	require.Equal(t, "v", gjson.GetBytes(bidExt, "prebid.targeting.single").String())
}

func TestHandleAuctionResponseHook_FullyEmptyResultNoMutation(t *testing.T) {
	mod, _ := Builder(json.RawMessage(`{"router_url":"https://r"}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)
	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.done = make(chan struct{})
	close(ar.done)
	ar.result = &AsyncResult{
		PerPlacement:   map[string]PlacementResult{"p1": {}},
		ImpToPlacement: map[string]string{"imp1": "p1"},
		TMPX:           "",
	}
	mc.Set(moduleContextAsyncKey, ar)
	resp := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{Bid: []openrtb2.Bid{{ID: "b1", ImpID: "imp1", Ext: json.RawMessage(`{}`)}}}},
		Ext:     json.RawMessage(`{}`),
	}
	payload := hookstage.AuctionResponsePayload{BidResponse: resp}
	miCtx := hookstage.ModuleInvocationContext{ModuleContext: mc}
	result, _ := m.HandleAuctionResponseHook(context.Background(), miCtx, payload)
	require.Empty(t, result.ChangeSet.Mutations(), "no mutation for fully-empty result")
}

func TestHandleAuctionResponseHook_EmptyEligiblePackagesNotEmitted(t *testing.T) {
	mod, _ := Builder(json.RawMessage(`{"router_url":"https://r"}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)
	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.done = make(chan struct{})
	close(ar.done)
	ar.result = &AsyncResult{
		PerPlacement:   map[string]PlacementResult{"p1": {EligiblePackages: []string{}, Segments: []string{"seg_a"}}},
		ImpToPlacement: map[string]string{"imp1": "p1"},
		TMPX:           "k1.tok",
	}
	mc.Set(moduleContextAsyncKey, ar)
	resp := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{Bid: []openrtb2.Bid{{ID: "b1", ImpID: "imp1", Ext: json.RawMessage(`{}`)}}}},
		Ext:     json.RawMessage(`{}`),
	}
	payload := hookstage.AuctionResponsePayload{BidResponse: resp}
	miCtx := hookstage.ModuleInvocationContext{ModuleContext: mc}
	result, _ := m.HandleAuctionResponseHook(context.Background(), miCtx, payload)
	for _, mut := range result.ChangeSet.Mutations() {
		payload, _ = mut.Apply(payload)
	}
	bidExt := payload.BidResponse.SeatBid[0].Bid[0].Ext
	// Mutation happened (TMPX + segments present), but eligible_packages should be absent.
	require.False(t, gjson.GetBytes(bidExt, "scope3.tmp.eligible_packages").Exists(), "empty packages should not be written")
	require.True(t, gjson.GetBytes(bidExt, "scope3.tmp.segments").Exists(), "segments should be written")
}
