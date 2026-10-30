package identity

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	iiqenrichment "github.com/Intent-IQ/identity-go/enrichment"
	iiqreporting "github.com/Intent-IQ/identity-go/reporting"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func traceModule(enricher iiqenrichment.Enricher) *Module {
	config := defaultConfig()
	config.APIEndpoint = "https://api.example"
	config.TraceEnabled = true
	return &Module{config: config, enricher: enricher}
}

func debugRequest() *openrtb_ext.RequestWrapper {
	return &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
		ID:     "auction",
		Ext:    json.RawMessage(`{"prebid":{"debug":true}}`),
		Device: &openrtb2.Device{IP: "1.2.3.4", UA: "ua"},
	}}
}

func runEnrichHook(t *testing.T, module *Module, request *openrtb_ext.RequestWrapper) hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload] {
	t.Helper()
	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{},
		hookstage.ProcessedAuctionRequestPayload{Request: request})
	require.NoError(t, err)
	return result
}

func applyTraceMutation(t *testing.T, result hookstage.HookResult[hookstage.AuctionResponsePayload], response *openrtb2.BidResponse) []string {
	t.Helper()
	mutations := result.ChangeSet.Mutations()
	require.Len(t, mutations, 1)
	payload, err := mutations[0].Apply(hookstage.AuctionResponsePayload{BidResponse: response})
	require.NoError(t, err)

	var ext struct {
		Trace struct {
			Identity []string `json:"iiq-identity"`
		} `json:"trace"`
	}
	require.NoError(t, json.Unmarshal(payload.BidResponse.Ext, &ext))
	return ext.Trace.Identity
}

func TestTraceOptIn(t *testing.T) {
	for _, test := range []struct {
		name string
		ext  string
		want bool
	}{
		{"debug true", `{"prebid":{"debug":true}}`, true},
		{"debug false", `{"prebid":{"debug":false}}`, false},
		{"trace verbose", `{"prebid":{"trace":"verbose"}}`, true},
		{"trace empty", `{"prebid":{"trace":""}}`, false},
		{"no prebid ext", `{"other":1}`, false},
		{"empty ext", ``, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := &openrtb2.BidRequest{}
			if test.ext != "" {
				request.Ext = json.RawMessage(test.ext)
			}
			assert.Equal(t, test.want, requestTraceOptIn(request))
		})
	}
	assert.False(t, requestTraceOptIn(nil))
}

func TestTraceGatedByConfigAndRequest(t *testing.T) {
	for _, test := range []struct {
		name         string
		traceEnabled bool
		optIn        bool
		want         bool
	}{
		{"both on", true, true, true},
		{"config off", false, true, false},
		{"request did not opt in", true, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			module := traceModule(&stubEnricher{})
			module.config.TraceEnabled = test.traceEnabled
			request := debugRequest()
			if !test.optIn {
				request.Ext = nil
			}

			result := runEnrichHook(t, module, request)
			stored, ok := loadAuctionContext(result.ModuleContext)
			require.True(t, ok)
			assert.Equal(t, test.want, len(result.DebugMessages) > 0)
			assert.Equal(t, test.want, stored.TraceEnabled)
			assert.Equal(t, test.want, len(stored.Trace) > 0)
		})
	}
}

func TestTraceAccountCanEnable(t *testing.T) {
	module := traceModule(&stubEnricher{})
	module.config.TraceEnabled = false
	result, err := module.HandleProcessedAuctionHook(context.Background(),
		hookstage.ModuleInvocationContext{AccountConfig: json.RawMessage(`{"trace_enabled":true}`)},
		hookstage.ProcessedAuctionRequestPayload{Request: debugRequest()})
	require.NoError(t, err)
	assert.NotEmpty(t, result.DebugMessages)
}

func TestTraceEnrichPaths(t *testing.T) {
	tc := int64(7)
	for _, test := range []struct {
		name     string
		enricher *stubEnricher
		want     []string
	}{
		{
			name: "enriched from s2s",
			enricher: &stubEnricher{result: iiqenrichment.Result{
				Outcome: iiqenrichment.OutcomeEnriched, TerminationCause: &tc, ABTestUUID: "ab",
				EIDs: []openrtb2.EID{{Source: "intentiq.com", UIDs: []openrtb2.UID{{ID: "x"}}}},
			}},
			want: []string{
				"[enrich] start — partner=, cache=off; request signals: eids=0, device.ifa=none, ip=1.2.3.4, consent=absent",
				"[enrich] outcome=enriched via IIQ S2S — eids=1, tc=7, abTestUuid=ab",
				"[enrich] result: enriched user.eids += 1 eid(s) [intentiq.com[x]]",
			},
		},
		{
			name:     "negative cache hit",
			enricher: &stubEnricher{result: iiqenrichment.Result{Outcome: iiqenrichment.OutcomeCachedNoIDs, CacheLayer: iiqenrichment.CacheLayerL2}},
			want: []string{
				"[enrich] outcome=no_ids_cached via cache l2 — eids=0, tc=none, abTestUuid=none",
				"[enrich] result: 0 eids resolved (reason=no_ids_cached) → user.eids unchanged",
			},
		},
		{
			name:     "wait expired",
			enricher: &stubEnricher{result: iiqenrichment.Result{Outcome: iiqenrichment.OutcomeWaitExpired}},
			want:     []string{"[enrich] outcome=wait_expired via none"},
		},
		{
			name:     "no endpoint",
			enricher: &stubEnricher{result: iiqenrichment.Result{Outcome: iiqenrichment.OutcomeNoEndpoint}},
			want:     []string{"[enrich] skipped: no api_endpoint configured (no-op)"},
		},
		{
			name:     "error",
			enricher: &stubEnricher{err: errors.New("boom")},
			want:     []string{"[enrich] resolution error (fail-open, request unchanged): boom"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			trace := strings.Join(runEnrichHook(t, traceModule(test.enricher), debugRequest()).DebugMessages, "\n")
			for _, want := range test.want {
				assert.Contains(t, trace, want)
			}
		})
	}
}

func TestTraceWrittenToResponseExt(t *testing.T) {
	reporter := &stubReporter{requests: make(chan iiqreporting.Request, 2)}
	module := traceModule(nil)
	module.config.ReportsEndpoint = "https://reports.example"
	module.reporter = reporter

	tc := int64(7)
	ctx := storeAuctionContext(auctionContext{
		Start: time.Now(), TraceEnabled: true, TerminationCause: &tc,
		Trace: []string{"[enrich] line one", "[enrich] line two"},
	})
	response := &openrtb2.BidResponse{
		Ext:     json.RawMessage(`{"prebid":{"auctiontimestamp":123}}`),
		SeatBid: []openrtb2.SeatBid{{Seat: "bidder", Bid: []openrtb2.Bid{{ID: "1"}, {ID: "2"}}}},
	}
	result, err := module.HandleAuctionResponseHook(context.Background(),
		hookstage.ModuleInvocationContext{ModuleContext: ctx}, hookstage.AuctionResponsePayload{BidResponse: response})
	require.NoError(t, err)

	lines := applyTraceMutation(t, result, response)
	require.GreaterOrEqual(t, len(lines), 4)
	assert.Equal(t, "[enrich] line one", lines[0])
	assert.Equal(t, "[enrich] line two", lines[1])
	assert.Contains(t, lines[2], "[response] start — flow latency (enrich → response)")
	trace := strings.Join(lines, "\n")
	assert.Contains(t, trace, "queued 2 impression report(s) → https://reports.example (fire-and-forget), 0 rejected")
	assert.Contains(t, trace, "tc=7 carried from enrich")
	assert.Contains(t, trace, "[response] response hook took")

	var ext struct {
		Prebid struct {
			AuctionTimestamp int64 `json:"auctiontimestamp"`
		} `json:"prebid"`
	}
	require.NoError(t, json.Unmarshal(response.Ext, &ext))
	assert.Equal(t, int64(123), ext.Prebid.AuctionTimestamp)
}

func TestTraceNoMutationWhenDisabled(t *testing.T) {
	for _, test := range []struct {
		name       string
		invocation hookstage.ModuleInvocationContext
	}{
		{"no module context", hookstage.ModuleInvocationContext{}},
		{"trace disabled", hookstage.ModuleInvocationContext{ModuleContext: storeAuctionContext(auctionContext{Start: time.Now()})}},
	} {
		t.Run(test.name, func(t *testing.T) {
			module := traceModule(nil)
			response := &openrtb2.BidResponse{}
			result, err := module.HandleAuctionResponseHook(context.Background(), test.invocation,
				hookstage.AuctionResponsePayload{BidResponse: response})
			require.NoError(t, err)
			assert.Empty(t, result.ChangeSet.Mutations())
			assert.Empty(t, result.DebugMessages)
			assert.Empty(t, response.Ext)
		})
	}
}

func TestTraceMutationNilResponseIsNoOp(t *testing.T) {
	payload, err := setIdentityTraceMutation([]string{"a"})(hookstage.AuctionResponsePayload{})
	require.NoError(t, err)
	assert.Nil(t, payload.BidResponse)
}

func TestFmtDur(t *testing.T) {
	for _, test := range []struct {
		in   time.Duration
		want string
	}{
		{312 * time.Nanosecond, "312ns"},
		{1500 * time.Nanosecond, "2µs"},
		{84 * time.Millisecond, "84ms"},
		{1390 * time.Millisecond, "1.39s"},
	} {
		assert.Equal(t, test.want, fmtDur(test.in))
	}
}

func TestAbTestShort(t *testing.T) {
	assert.Equal(t, "none", abTestShort(""))
	assert.Equal(t, "short-uuid", abTestShort("short-uuid"))
	assert.Equal(t, "0123456789ab…uvwxyz (36 chars)", abTestShort("0123456789abcdefghijklmnopqrstuvwxyz"))
}

func TestTraceDisabledCollectsNothing(t *testing.T) {
	var nilTrace *flowTrace
	nilTrace.tracef("nil receiver must not panic")

	off := &flowTrace{}
	off.tracef("dropped %d", 1)
	assert.Nil(t, off.msgs)
}
