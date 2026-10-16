package identity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

func debugReq() *openrtb2.BidRequest {
	return &openrtb2.BidRequest{
		ID:     "auc-trace",
		Ext:    json.RawMessage(`{"prebid":{"debug":true}}`),
		Site:   &openrtb2.Site{Domain: "example.com"},
		Device: &openrtb2.Device{IP: "1.2.3.4", UA: "UA-trace"},
	}
}

func newStatusServer(t *testing.T, status int, cap *capture) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.hits++
		w.WriteHeader(status)
		_, _ = w.Write([]byte(emptyBody))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func traceOf(t *testing.T, res hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]) string {
	t.Helper()
	return strings.Join(res.DebugMessages, "\n")
}

func TestTraceOptIn(t *testing.T) {
	for _, tc := range []struct {
		name string
		ext  string
		want bool
	}{
		{"debug true", `{"prebid":{"debug":true}}`, true},
		{"debug false", `{"prebid":{"debug":false}}`, false},
		{"trace verbose", `{"prebid":{"trace":"verbose"}}`, true},
		{"trace basic", `{"prebid":{"trace":"basic"}}`, true},
		{"trace empty", `{"prebid":{"trace":""}}`, false},
		{"no prebid ext", `{"other":1}`, false},
		{"empty ext", ``, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := &openrtb2.BidRequest{}
			if tc.ext != "" {
				req.Ext = json.RawMessage(tc.ext)
			}
			assert.Equal(t, tc.want, requestTraceOptIn(req))
		})
	}
	assert.False(t, requestTraceOptIn(nil), "nil request must not opt in")
}

func TestTraceGatedByConfigAndRequest(t *testing.T) {
	for _, tc := range []struct {
		name         string
		traceEnabled bool
		debugReq     bool
		wantMessages bool
	}{
		{"both on", true, true, true},
		{"config off", false, true, false},
		{"request did not opt in", true, false, false},
		{"both off", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cap := &capture{}
			m := newModule(newServer(t, eidsBody, cap).URL, http.DefaultClient, &countMetrics{})
			m.cfg.TraceEnabled = tc.traceEnabled

			req := &openrtb2.BidRequest{}
			if tc.debugReq {
				req.Ext = json.RawMessage(`{"prebid":{"debug":true}}`)
			}
			res, _ := runHook(t, m, req)

			fc := flowFrom(t, res)
			assert.Equal(t, tc.wantMessages, len(res.DebugMessages) > 0)
			assert.Equal(t, tc.wantMessages, fc.traceEnabled)
			assert.Equal(t, tc.wantMessages, len(fc.trace) > 0)
		})
	}
}

func TestTraceEnrichDirectCallPath(t *testing.T) {
	cap := &capture{}
	m := newModule(newServer(t, eidsBody, cap).URL, http.DefaultClient, &countMetrics{})
	m.cfg.TraceEnabled = true

	res, _ := runHook(t, m, debugReq())
	trace := traceOf(t, res)

	assert.Contains(t, trace, "[enrich] start — partner=123, cache=off")
	assert.Contains(t, trace, "eids=0, device.ifa=none, ip=1.2.3.4, consent=absent")
	assert.Contains(t, trace, "cache disabled — direct S2S call")
	assert.Contains(t, trace, "→ IIQ resolve GET (dpi=123, ip=1.2.3.4, gdpr-consent=absent")
	assert.Contains(t, trace, "[enrich]   url=")
	assert.Contains(t, trace, "← IIQ 200 in")
	assert.Contains(t, trace, "eids=1, tc=none, cttl=none, abTestUuid=none")
	assert.Contains(t, trace, "enriched user.eids += 1 eid(s) [intentiq.com[abc]]")
}

func TestTraceEnrichSkipAndErrorPaths(t *testing.T) {
	t.Run("no endpoint", func(t *testing.T) {
		m := newModule("", http.DefaultClient, &countMetrics{})
		m.cfg.TraceEnabled = true

		res, _ := runHook(t, m, debugReq())
		assert.Contains(t, traceOf(t, res), "skipped: no api_endpoint configured")
	})

	t.Run("zero eids", func(t *testing.T) {
		cap := &capture{}
		m := newModule(newServer(t, emptyBody, cap).URL, http.DefaultClient, &countMetrics{})
		m.cfg.TraceEnabled = true

		res, _ := runHook(t, m, debugReq())
		assert.Contains(t, traceOf(t, res), "0 eids resolved (reason=no_ids) → user.eids unchanged")
	})

	t.Run("upstream error is traced fail-open", func(t *testing.T) {
		cap := &capture{}
		srv := newServer(t, emptyBody, cap)
		badURL := srv.URL
		srv.Close()

		m := newModule(badURL, http.DefaultClient, &countMetrics{})
		m.cfg.TraceEnabled = true

		res, _ := runHook(t, m, debugReq())
		trace := traceOf(t, res)
		assert.Contains(t, trace, "← IIQ transport error after")
		assert.Contains(t, trace, "resolution error (fail-open, request unchanged): reason=transport")
	})

	t.Run("non-2xx carries the status", func(t *testing.T) {
		cap := &capture{}
		srv := newStatusServer(t, http.StatusTooManyRequests, cap)
		m := newModule(srv.URL, http.DefaultClient, &countMetrics{})
		m.cfg.TraceEnabled = true

		res, _ := runHook(t, m, debugReq())
		assert.Contains(t, traceOf(t, res), "reason=status status=429")
	})
}

func TestTraceEnrichCachePaths(t *testing.T) {
	cap := &capture{}
	body := `{"data":{"eids":[{"source":"intentiq.com","uids":[{"id":"abc"}]}]},"abTestUuid":"ab-1","tc":5}`
	m := newCachedModule(t, newServer(t, body, cap).URL, &countMetrics{})
	m.cfg.TraceEnabled = true

	req := debugReq()
	req.Device.IFA = "ifa-1"

	res1, _ := runHook(t, m, req)
	first := traceOf(t, res1)
	assert.Contains(t, first, "cache=on")
	assert.Contains(t, first, "device.ifa=present")
	assert.Contains(t, first, "cache lookup over")
	assert.Contains(t, first, "cache MISS (keytype=")
	assert.Contains(t, first, "in-progress marker set, calling IIQ")
	assert.Contains(t, first, "cached 1 eid(s) under all keys: [intentiq.com[abc]]")
	require.Equal(t, 1, cap.hits)

	res2, _ := runHook(t, m, req)
	second := traceOf(t, res2)
	assert.Equal(t, 1, cap.hits, "second request must be served from cache")
	assert.Contains(t, second, "cache HIT (layer=l1")
	assert.Contains(t, second, "no S2S call")
	assert.Contains(t, second, "tc=5, abTestUuid=ab-1")
	assert.NotContains(t, second, "→ IIQ resolve GET")
}

func TestTraceEnrichNegativeCachePath(t *testing.T) {
	cap := &capture{}
	m := newCachedModule(t, newServer(t, `{"data":{"eids":[]},"tc":120088}`, cap).URL, &countMetrics{})
	m.cfg.TraceEnabled = true

	req := debugReq()
	req.Device.IFA = "ifa-neg"

	res1, _ := runHook(t, m, req)
	assert.Contains(t, traceOf(t, res1), "cached negative sentinel under all keys (0 eids)")

	res2, _ := runHook(t, m, req)
	trace := traceOf(t, res2)
	assert.Equal(t, 1, cap.hits, "negative entry suppresses the upstream call")
	assert.Contains(t, trace, "cache NEGATIVE (layer=l1")
	assert.Contains(t, trace, "known-unresolvable, no S2S call; tc=120088")
	assert.Contains(t, trace, "0 eids resolved (reason=no_ids_cached)")
}

func TestTraceEnrichInProgressPath(t *testing.T) {
	cap := &capture{}
	m := newCachedModule(t, newServer(t, eidsBody, cap).URL, &countMetrics{})
	m.cfg.TraceEnabled = true

	req := debugReq()
	req.Device.IFA = "ifa-inflight"
	keys := m.keyExtractor.CandidateKeys(req)
	require.NotEmpty(t, keys)
	m.cache.PutInProgress(t.Context(), keys)

	res, _ := runHook(t, m, req)
	trace := traceOf(t, res)
	assert.Equal(t, 0, cap.hits, "in-progress marker must suppress the upstream call")
	assert.Contains(t, trace, "cache IN-PROGRESS (layer=")
	assert.Contains(t, trace, "concurrent resolution already running, no S2S call")
	assert.Contains(t, trace, "0 eids resolved (reason=in_progress)")
}

func TestTraceWrittenToResponseExt(t *testing.T) {
	m := impModule("", http.DefaultClient, &impMetrics{})
	m.cfg.TraceEnabled = true

	fc := flowContext{
		start:        time.Now().Add(-50 * time.Millisecond),
		trace:        []string{"[enrich] line one", "[enrich] line two"},
		traceEnabled: true,
	}
	resp := twoBidResponse()
	res, err := m.HandleAuctionResponseHook(t.Context(), miCtxWithFlow(fc),
		hookstage.AuctionResponsePayload{BidResponse: resp})
	require.NoError(t, err)

	lines := applyTraceMutation(t, res, resp)
	require.GreaterOrEqual(t, len(lines), 3)
	assert.Equal(t, "[enrich] line one", lines[0])
	assert.Equal(t, "[enrich] line two", lines[1])
	assert.Contains(t, lines[2], "[response] start — flow latency (enrich → response)")
	assert.Contains(t, strings.Join(lines, "\n"), "no reports_endpoint configured")
}

func TestTraceResponseReportPath(t *testing.T) {
	m := impModule("https://reports.example/x", http.DefaultClient, &impMetrics{})
	m.cfg.TraceEnabled = true

	tc := int64(7)
	fc := flowContext{start: time.Now(), trace: []string{"[enrich] x"}, traceEnabled: true, terminationCause: &tc}
	resp := twoBidResponse()
	res, err := m.HandleAuctionResponseHook(t.Context(), miCtxWithFlow(fc),
		hookstage.AuctionResponsePayload{BidResponse: resp})
	require.NoError(t, err)

	trace := strings.Join(applyTraceMutation(t, res, resp), "\n")
	assert.Contains(t, trace, "queued 2 impression report(s) → https://reports.example/x")
	assert.Contains(t, trace, "tc=7 carried from enrich")
	assert.Contains(t, trace, "[response] response hook took")
}

func TestTracePreservesExistingResponseExt(t *testing.T) {
	m := impModule("", http.DefaultClient, &impMetrics{})
	m.cfg.TraceEnabled = true

	resp := twoBidResponse()
	resp.Ext = json.RawMessage(`{"prebid":{"auctiontimestamp":123}}`)

	res, err := m.HandleAuctionResponseHook(t.Context(),
		miCtxWithFlow(flowContext{start: time.Now(), traceEnabled: true}),
		hookstage.AuctionResponsePayload{BidResponse: resp})
	require.NoError(t, err)

	applyTraceMutation(t, res, resp)

	var ext struct {
		Prebid struct {
			AuctionTimestamp int64 `json:"auctiontimestamp"`
		} `json:"prebid"`
	}
	require.NoError(t, json.Unmarshal(resp.Ext, &ext))
	assert.Equal(t, int64(123), ext.Prebid.AuctionTimestamp)
}

func TestTraceNoMutationWhenDisabled(t *testing.T) {
	for _, tc := range []struct {
		name string
		mi   hookstage.ModuleInvocationContext
	}{
		{"no flow context", hookstage.ModuleInvocationContext{}},
		{"trace disabled in flow", miCtxWithFlow(flowContext{start: time.Now()})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := impModule("", http.DefaultClient, &impMetrics{})
			resp := twoBidResponse()
			res, err := m.HandleAuctionResponseHook(t.Context(), tc.mi,
				hookstage.AuctionResponsePayload{BidResponse: resp})
			require.NoError(t, err)

			assert.Empty(t, res.ChangeSet.Mutations())
			assert.Empty(t, res.DebugMessages)
			assert.Empty(t, resp.Ext)
		})
	}
}

func TestTraceMutationNilResponseIsNoOp(t *testing.T) {
	mutate := setIdentityTraceMutation([]string{"a"})
	p, err := mutate(hookstage.AuctionResponsePayload{})
	require.NoError(t, err)
	assert.Nil(t, p.BidResponse)
}

func TestFmtDur(t *testing.T) {
	for _, tc := range []struct {
		in   time.Duration
		want string
	}{
		{312 * time.Nanosecond, "312ns"},
		{1500 * time.Nanosecond, "2µs"},
		{84_600 * time.Nanosecond, "85µs"},
		{84 * time.Millisecond, "84ms"},
		{1390 * time.Millisecond, "1.39s"},
	} {
		assert.Equal(t, tc.want, fmtDur(tc.in))
	}
}

func TestAbTestShort(t *testing.T) {
	assert.Equal(t, "none", abTestShort(""))
	assert.Equal(t, "short-uuid", abTestShort("short-uuid"))

	long := "0123456789abcdefghijklmnopqrstuvwxyz"
	assert.Equal(t, "0123456789ab…uvwxyz (36 chars)", abTestShort(long))
}

func TestTraceHelpersRenderAbsentValues(t *testing.T) {
	assert.Equal(t, "none", tcStr(nil))
	assert.Equal(t, "none", orNone(""))
	assert.Equal(t, "429", orNone("429"))
	assert.Equal(t, "off", onOff(false))
	assert.Equal(t, "absent", presentAbsent(false))
	assert.Equal(t, "eids=0, device.ifa=none, ip=, consent=absent", requestSignals(nil))
}

// A disabled trace must not allocate — the tracef calls are on the hot path of every auction.
func TestTraceDisabledCollectsNothing(t *testing.T) {
	var tr *flowTrace
	tr.tracef("nil receiver must not panic")

	off := &flowTrace{}
	off.tracef("dropped %d", 1)
	assert.Nil(t, off.msgs)
}

func applyTraceMutation(t *testing.T, res hookstage.HookResult[hookstage.AuctionResponsePayload], resp *openrtb2.BidResponse) []string {
	t.Helper()
	muts := res.ChangeSet.Mutations()
	require.Len(t, muts, 1, "expected exactly one trace mutation")

	p := hookstage.AuctionResponsePayload{BidResponse: resp}
	p, err := muts[0].Apply(p)
	require.NoError(t, err)

	var ext struct {
		Trace struct {
			Identity []string `json:"iiq-identity"`
		} `json:"trace"`
	}
	require.NoError(t, json.Unmarshal(p.BidResponse.Ext, &ext))
	return ext.Trace.Identity
}
