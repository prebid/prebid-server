package identity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

func TestIiqResponseCttl(t *testing.T) {
	assert.Equal(t, time.Duration(0), iiqResponse{}.cttl(), "absent cttl -> 0")
	v := int64(30)
	assert.Equal(t, 30*time.Second, iiqResponse{Cttl: &v}.cttl())
}

func TestFlowContextRoundTrip(t *testing.T) {
	fc := flowContext{start: time.Now(), abTestUUID: "ab", auctionID: "auc"}
	got, ok := getFlowContext(setFlowContext(fc))
	assert.True(t, ok)
	assert.Equal(t, fc, got)
}

func TestGetFlowContextMissing(t *testing.T) {
	_, ok := getFlowContext(nil)
	assert.False(t, ok, "nil module context -> not present")

	_, ok = getFlowContext(hookstage.NewModuleContext())
	assert.False(t, ok, "empty module context -> not present")

	mctx := hookstage.NewModuleContext()
	mctx.Set(flowContextKey, "not a flowContext")
	_, ok = getFlowContext(mctx)
	assert.False(t, ok, "wrong type under key -> not present")
}

// TestNoopMetrics exercises every no-op method so the metric-agnostic fallback is covered and cannot
// panic (it is used whenever Prometheus is disabled).
func TestNoopMetrics(t *testing.T) {
	var m Metrics = noopMetrics{}
	assert.NotPanics(t, func() {
		m.CacheHit("l1", "first_party", "p")
		m.CacheMiss("first_party", "p")
		m.CacheNegativeHit("l2", "device", "p")
		m.CacheInProgress("l1", "third_party", "p")
		m.APISuccess("p")
		m.APIError("p")
		m.APILatency(time.Second, "p")
		m.Enriched("p")
		m.EidsNone("p")
		m.SkipNoEndpoint("p")
		m.TerminationCause(1, "p")
		m.FlowLatency(time.Second, "p")
		m.ImpressionReported("p")
		m.ImpressionError("p")
		m.L1GetError()
		m.L1PutError()
		m.L2GetLatency(time.Second)
		m.L2PutLatency(time.Second)
		m.L2GetError()
		m.L2PutError()
		m.RegisterL1Gauges(func() int64 { return 0 }, func() int64 { return 0 })
		m.RegisterL2Gauges(func() int64 { return 0 }, func() int64 { return 0 })
	})
}
