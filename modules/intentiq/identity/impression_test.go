package identity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

// impMetrics is a concurrency-safe Metrics stub for the impression hook (the report fires in a
// detached goroutine).
type impMetrics struct {
	noopMetrics
	reported atomic.Int64
	errored  atomic.Int64
	flow     atomic.Int64
}

func (m *impMetrics) ImpressionReported(string)         { m.reported.Add(1) }
func (m *impMetrics) ImpressionError(string)            { m.errored.Add(1) }
func (m *impMetrics) FlowLatency(time.Duration, string) { m.flow.Add(1) }

func impModule(endpoint string, client *http.Client, metrics Metrics) *Module {
	return &Module{
		cfg:        Config{ReportsEndpoint: endpoint, PartnerID: "part-9", Timeout: 1000},
		httpClient: client,
		metrics:    metrics,
	}
}

func miCtxWithFlow(fc flowContext) hookstage.ModuleInvocationContext {
	return hookstage.ModuleInvocationContext{ModuleContext: setFlowContext(fc)}
}

func twoBidResponse() *openrtb2.BidResponse {
	return &openrtb2.BidResponse{
		Cur: "EUR",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "bidderA",
			Bid: []openrtb2.Bid{
				{ImpID: "imp1", Price: 1.5},
				{ImpID: "imp2", Price: 2.0, Ext: json.RawMessage(`{"origbidcpm":2.2,"origbidcur":"USD"}`)},
			},
		}},
	}
}

func TestImpression_NoEndpoint_NoCall(t *testing.T) {
	m := impModule("", http.DefaultClient, &impMetrics{})
	res, err := m.HandleAuctionResponseHook(t.Context(),
		miCtxWithFlow(flowContext{start: time.Now()}),
		hookstage.AuctionResponsePayload{BidResponse: twoBidResponse()})
	require.NoError(t, err)
	assert.False(t, res.Reject)
}

func TestImpression_FlowLatencyRecordedEvenWithoutEndpoint(t *testing.T) {
	metrics := &impMetrics{}
	m := impModule("", http.DefaultClient, metrics)
	_, err := m.HandleAuctionResponseHook(t.Context(),
		miCtxWithFlow(flowContext{start: time.Now()}),
		hookstage.AuctionResponsePayload{BidResponse: twoBidResponse()})
	require.NoError(t, err)
	assert.Equal(t, int64(1), metrics.flow.Load())
}

func TestImpression_NilResponse_NoPanic(t *testing.T) {
	metrics := &impMetrics{}
	m := impModule("https://reports.example/x", http.DefaultClient, metrics)
	_, err := m.HandleAuctionResponseHook(t.Context(),
		miCtxWithFlow(flowContext{start: time.Now()}),
		hookstage.AuctionResponsePayload{BidResponse: nil})
	require.NoError(t, err)
}

func TestImpression_ReportsOnePerBid(t *testing.T) {
	type reqRec struct {
		query string
		rdata string
	}
	got := make(chan reqRec, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- reqRec{query: r.URL.RawQuery, rdata: r.URL.Query().Get("rdata")}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	metrics := &impMetrics{}
	m := impModule(srv.URL, srv.Client(), metrics)

	fc := flowContext{
		start:      time.Now(),
		abTestUUID: "ab-1",
		auctionID:  "auc-1",
		ref:        "example.com",
		ip:         "5.6.7.8",
		ua:         "UA/1",
	}
	_, err := m.HandleAuctionResponseHook(t.Context(), miCtxWithFlow(fc),
		hookstage.AuctionResponsePayload{BidResponse: twoBidResponse()})
	require.NoError(t, err)

	// Two bids -> two fire-and-forget reports.
	recs := make([]reqRec, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case r := <-got:
			recs = append(recs, r)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for report %d", i+1)
		}
	}

	// Fixed query params on every report.
	for _, r := range recs {
		q, err := url.ParseQuery(r.query)
		require.NoError(t, err)
		assert.Equal(t, "45", q.Get("at"))
		assert.Equal(t, "1", q.Get("rtype"))
		assert.Equal(t, sourcePBSGo, q.Get("source"))
		assert.Equal(t, "part-9", q.Get("dpi"))
	}

	// Locate the imp2 report and assert its rdata JSON round-trips the expected fields (incl.
	// original cpm/currency and the flow-context fields).
	var imp2 map[string]any
	for _, r := range recs {
		var d map[string]any
		require.NoError(t, json.Unmarshal([]byte(r.rdata), &d))
		if d["placementId"] == "imp2" {
			imp2 = d
		}
	}
	require.NotNil(t, imp2, "expected a report for imp2")
	assert.Equal(t, "bidderA", imp2["bidderCode"])
	assert.Equal(t, "part-9", imp2["partnerId"])
	assert.Equal(t, "EUR", imp2["currency"])
	assert.Equal(t, biddingPlatformOpenRTB, imp2["biddingPlatformId"])
	assert.EqualValues(t, 2.2, imp2["originalCpm"])
	assert.Equal(t, "USD", imp2["originalCurrency"])
	assert.Equal(t, "example.com", imp2["vrref"])
	assert.Equal(t, "auc-1", imp2["prebidAuctionId"])
	assert.Equal(t, "auc-1", imp2["partnerAuctionId"])
	assert.Equal(t, "ab-1", imp2["abTestUuid"])
	assert.Equal(t, "5.6.7.8", imp2["ip"])
	assert.Equal(t, "UA/1", imp2["ua"])

	assert.Eventually(t, func() bool { return metrics.reported.Load() == 2 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, int64(1), metrics.flow.Load())
}

func TestImpression_DefaultCurrencyWhenAbsent(t *testing.T) {
	got := make(chan string, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.URL.Query().Get("rdata")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := impModule(srv.URL, srv.Client(), &impMetrics{})
	resp := &openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{{Seat: "s", Bid: []openrtb2.Bid{{ImpID: "i", Price: 1}}}}}
	_, err := m.HandleAuctionResponseHook(t.Context(), miCtxWithFlow(flowContext{start: time.Now()}),
		hookstage.AuctionResponsePayload{BidResponse: resp})
	require.NoError(t, err)

	select {
	case rdata := <-got:
		var d map[string]any
		require.NoError(t, json.Unmarshal([]byte(rdata), &d))
		assert.Equal(t, defaultCurrency, d["currency"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}

func TestImpression_ErrorCounted(t *testing.T) {
	// Point at an unroutable endpoint so the GET fails; the error counter must increment.
	metrics := &impMetrics{}
	m := impModule("http://127.0.0.1:0/reports", &http.Client{Timeout: 500 * time.Millisecond}, metrics)
	resp := &openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{{Seat: "s", Bid: []openrtb2.Bid{{ImpID: "i", Price: 1}}}}}
	_, err := m.HandleAuctionResponseHook(t.Context(), miCtxWithFlow(flowContext{start: time.Now()}),
		hookstage.AuctionResponsePayload{BidResponse: resp})
	require.NoError(t, err)
	assert.Eventually(t, func() bool { return metrics.errored.Load() == 1 }, 3*time.Second, 10*time.Millisecond)
}

// TestOrderedMapMarshalJSON verifies rdata preserves insertion order (matching Java LinkedHashMap).
func TestOrderedMapMarshalJSON(t *testing.T) {
	om := newOrderedMap()
	om.put("b", 1)
	om.put("a", "x")
	om.put("b", 2) // overwrite value, keep position
	out, err := om.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `{"b":2,"a":"x"}`, string(out))
}
