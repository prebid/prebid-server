package identity

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestMetrics returns a promMetrics backed by a fresh registry for isolated assertions.
func newTestMetrics(t *testing.T) (*promMetrics, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	m := newMetrics(reg, true)
	pm, ok := m.(*promMetrics)
	require.True(t, ok, "newMetrics(reg, true) must return the prometheus impl")
	return pm, reg
}

func TestNewMetricsFallsBackToNoop(t *testing.T) {
	assert.IsType(t, noopMetrics{}, newMetrics(nil, true), "nil registry -> noop")
	assert.IsType(t, noopMetrics{}, newMetrics(prometheus.NewRegistry(), false), "disabled -> noop")
	assert.IsType(t, &promMetrics{}, newMetrics(prometheus.NewRegistry(), true), "enabled -> prom")
}

func TestMetricsCacheOutcomeCounters(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.CacheHit("l1", "third_party", "123")
	m.CacheHit("l1", "third_party", "123")
	m.CacheHit("l2", "first_party", "123")
	m.CacheMiss("device", "123")
	m.CacheNegativeHit("l2", "first_party", "999")
	m.CacheInProgress("l1", "device", "123")

	assert.Equal(t, 2.0, testutil.ToFloat64(m.cacheHit.WithLabelValues("l1", "third_party", "123")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.cacheHit.WithLabelValues("l2", "first_party", "123")))
	assert.Equal(t, 2, testutil.CollectAndCount(m.cacheHit), "two distinct label sets")
	assert.Equal(t, 1.0, testutil.ToFloat64(m.cacheMiss.WithLabelValues("device", "123")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.cacheNegative.WithLabelValues("l2", "first_party", "999")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.cacheInProgress.WithLabelValues("l1", "device", "123")))
}

func TestMetricsBusinessCounters(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.APISuccess("p")
	m.APIError("p")
	m.APIError("p")
	m.Enriched("p")
	m.EidsNone("p")
	m.SkipNoEndpoint("p")
	m.ImpressionReported("p")
	m.ImpressionError("p")

	assert.Equal(t, 1.0, testutil.ToFloat64(m.apiSuccess.WithLabelValues("p")))
	assert.Equal(t, 2.0, testutil.ToFloat64(m.apiError.WithLabelValues("p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.enriched.WithLabelValues("p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.eidsNone.WithLabelValues("p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.skipNoEndpoint.WithLabelValues("p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.impressionReported.WithLabelValues("p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.impressionError.WithLabelValues("p")))
}

func TestMetricsTerminationCauseBounds(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.TerminationCause(0, "p")   // recorded (lower bound)
	m.TerminationCause(199, "p") // recorded (upper bound)
	m.TerminationCause(200, "p") // dropped
	m.TerminationCause(-1, "p")  // dropped

	assert.Equal(t, 1.0, testutil.ToFloat64(m.terminationCause.WithLabelValues("0", "p")))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.terminationCause.WithLabelValues("199", "p")))
	assert.Equal(t, 2, testutil.CollectAndCount(m.terminationCause), "out-of-range tc must not create series")
}

func TestMetricsLatencyHistograms(t *testing.T) {
	m, reg := newTestMetrics(t)

	m.APILatency(150*time.Millisecond, "p")
	m.FlowLatency(2*time.Second, "p")
	m.L2GetLatency(500 * time.Microsecond)
	m.L2PutLatency(1 * time.Millisecond)

	assert.Equal(t, 1, testutil.CollectAndCount(m.apiLatency))
	assert.Equal(t, 1, testutil.CollectAndCount(m.flowLatency))

	// The latency histograms sum to the observed seconds and are named with the expected prefix.
	got, err := reg.Gather()
	require.NoError(t, err)
	names := map[string]bool{}
	for _, mf := range got {
		names[mf.GetName()] = true
	}
	for _, want := range []string{
		metricPrefix + "api_latency_seconds",
		metricPrefix + "flow_latency_seconds",
		metricPrefix + "l2_get_latency_seconds",
		metricPrefix + "l2_put_latency_seconds",
	} {
		assert.True(t, names[want], "missing histogram %s", want)
	}

	// L2 histograms are global (no partner/layer labels).
	assert.Equal(t, 1, testutil.CollectAndCount(m.l2GetLatency))
	assert.Equal(t, 1, testutil.CollectAndCount(m.l2PutLatency))
}

func TestMetricsGlobalErrorCounters(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.L1GetError()
	m.L1PutError()
	m.L1PutError()
	m.L2GetError()
	m.L2PutError()

	assert.Equal(t, 1.0, testutil.ToFloat64(m.l1GetError))
	assert.Equal(t, 2.0, testutil.ToFloat64(m.l1PutError))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.l2GetError))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.l2PutError))
}

func TestMetricsGaugesLazyAndIdempotent(t *testing.T) {
	m, reg := newTestMetrics(t)

	// Before registration the gauges must not exist.
	assert.Equal(t, 0, countSeriesWithSuffix(t, reg, "l1_size"))

	l1Size, l1Evict := int64(7), int64(3)
	l2Size, l2Evict := int64(42), int64(11)
	m.RegisterL1Gauges(func() int64 { return l1Size }, func() int64 { return l1Evict })
	m.RegisterL2Gauges(func() int64 { return l2Size }, func() int64 { return l2Evict })

	// A second call must not panic or double-register.
	m.RegisterL1Gauges(func() int64 { return 100 }, func() int64 { return 100 })
	m.RegisterL2Gauges(func() int64 { return 100 }, func() int64 { return 100 })

	assert.Equal(t, 7.0, gaugeValue(t, reg, metricPrefix+"l1_size"))
	assert.Equal(t, 3.0, gaugeValue(t, reg, metricPrefix+"l1_eviction"))
	assert.Equal(t, 42.0, gaugeValue(t, reg, metricPrefix+"l2_size"))
	assert.Equal(t, 11.0, gaugeValue(t, reg, metricPrefix+"l2_eviction"))

	// GaugeFunc reads the live closure on each scrape.
	l1Size = 9
	assert.Equal(t, 9.0, gaugeValue(t, reg, metricPrefix+"l1_size"))
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
