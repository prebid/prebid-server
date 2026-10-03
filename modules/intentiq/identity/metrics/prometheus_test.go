package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMetrics(t *testing.T) (*promMetrics, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	return newPromMetrics(reg), reg
}

func TestMetricsCacheLookup(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.CacheLookup(ResultHit, "l1", "123")
	m.CacheLookup(ResultHit, "l1", "123")
	m.CacheLookup(ResultHit, "l2", "123")
	m.CacheLookup(ResultMiss, "none", "123")

	assert.Equal(t, 2.0, testutil.ToFloat64(m.cacheLookup.WithLabelValues(ResultHit, "l1", "123")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.cacheLookup.WithLabelValues(ResultHit, "l2", "123")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.cacheLookup.WithLabelValues(ResultMiss, "none", "123")))
	assert.Equal(t, 3, testutil.CollectAndCount(m.cacheLookup), "three distinct label sets")
}

func TestMetricsBusinessCounters(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.Requests("p")
	m.APISuccess("p")
	m.APIError("p", "timeout", "")
	m.APIError("p", "status", "500")
	m.Enriched("p")
	m.NotEnriched(ReasonNoIDs, "p")
	m.NotEnriched(ReasonNoIDsCached, "p")
	m.NotEnriched(ReasonInProgress, "p")
	m.NotEnriched(ReasonNoEndpoint, "p")
	m.ImpressionReported("p")
	m.ImpressionError("p")

	assert.Equal(t, 1.0, testutil.ToFloat64(m.requests.WithLabelValues("p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.apiSuccess.WithLabelValues("p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.apiError.WithLabelValues("timeout", "", "p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.apiError.WithLabelValues("status", "500", "p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.enriched.WithLabelValues("p")))
	assert.Equal(t, 4, testutil.CollectAndCount(m.notEnriched), "one series per reason")
	assert.Equal(t, 1.0, testutil.ToFloat64(m.notEnriched.WithLabelValues(ReasonInProgress, "p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.impressionReported.WithLabelValues("p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.impressionError.WithLabelValues("p")))
}

func TestMetricsLatencyHistograms(t *testing.T) {
	m, reg := newTestMetrics(t)

	m.APILatency(150*time.Millisecond, "p")
	m.L2GetLatency(500 * time.Microsecond)
	m.L2PutLatency(1 * time.Millisecond)

	assert.Equal(t, 1, testutil.CollectAndCount(m.apiLatency))

	got, err := reg.Gather()
	require.NoError(t, err)
	names := map[string]bool{}
	for _, mf := range got {
		names[mf.GetName()] = true
	}
	for _, want := range []string{
		Prefix + "api_latency_seconds",
		Prefix + "l2_get_latency_seconds",
		Prefix + "l2_put_latency_seconds",
	} {
		assert.True(t, names[want], "missing histogram %s", want)
	}

	// L2 histograms are global (no partner/layer labels).
	assert.Equal(t, 1, testutil.CollectAndCount(m.l2GetLatency))
	assert.Equal(t, 1, testutil.CollectAndCount(m.l2PutLatency))
}

func TestMetricsL2RequestsAndL1PutError(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.L2Request(OpGet, ResultHit)
	m.L2Request(OpGet, ResultMiss)
	m.L2Request(OpGet, ResultError)
	m.L2Request(OpPut, ResultStored)
	m.L2Request(OpPut, ResultError)
	m.L1PutError()
	m.L1PutError()

	assert.Equal(t, 1.0, testutil.ToFloat64(m.l2Requests.WithLabelValues(OpGet, ResultHit)))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.l2Requests.WithLabelValues(OpGet, ResultMiss)))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.l2Requests.WithLabelValues(OpGet, ResultError)))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.l2Requests.WithLabelValues(OpPut, ResultStored)))
	assert.Equal(t, 5, testutil.CollectAndCount(m.l2Requests))
	assert.Equal(t, 2.0, testutil.ToFloat64(m.l1PutError))
}

// The fully-qualified prefix must stay "iiq_identity_": dashboards and alerts key on it.
func TestMetricNamesUseIIQIdentityPrefix(t *testing.T) {
	m, reg := newTestMetrics(t)
	m.Requests("p")

	got, err := reg.Gather()
	require.NoError(t, err)
	require.NotEmpty(t, got)
	for _, mf := range got {
		assert.True(t, strings.HasPrefix(mf.GetName(), "iiq_identity_"),
			"unexpected metric name %q", mf.GetName())
	}
}

func TestMetricsGaugesLazyAndIdempotent(t *testing.T) {
	m, reg := newTestMetrics(t)

	// Before registration the gauges must not exist.
	assert.Equal(t, 0, countSeriesWithSuffix(t, reg, "l1_size"))

	l1Size, l1Evict := int64(7), int64(3)
	m.RegisterL1Gauges(func() int64 { return l1Size }, func() int64 { return l1Evict })

	// A second call must not panic or double-register.
	m.RegisterL1Gauges(func() int64 { return 100 }, func() int64 { return 100 })

	assert.Equal(t, 7.0, gaugeValue(t, reg, Prefix+"l1_size"))
	assert.Equal(t, 3.0, gaugeValue(t, reg, Prefix+"l1_eviction"))

	// GaugeFunc reads the live closure on each scrape.
	l1Size = 9
	assert.Equal(t, 9.0, gaugeValue(t, reg, Prefix+"l1_size"))
}

func countSeriesWithSuffix(t *testing.T, g prometheus.Gatherer, suffix string) int {
	t.Helper()
	mfs, err := g.Gather()
	require.NoError(t, err)
	n := 0
	for _, mf := range mfs {
		if strings.HasSuffix(mf.GetName(), suffix) {
			n += len(mf.GetMetric())
		}
	}
	return n
}

func gaugeValue(t *testing.T, g prometheus.Gatherer, name string) float64 {
	t.Helper()
	mfs, err := g.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		require.Len(t, mf.GetMetric(), 1)
		return mf.GetMetric()[0].GetGauge().GetValue()
	}
	t.Fatalf("gauge %s not found", name)
	return 0
}
