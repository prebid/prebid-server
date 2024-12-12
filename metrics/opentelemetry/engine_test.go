package opentelemetry

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstantiateMetrics(t *testing.T) {
	t.Run("no prefix", func(t *testing.T) {
		emptyPrefixEngine, err := NewEngine("", &config.DisabledMetrics{})
		require.NoError(t, err)
		assert.NotNil(t, emptyPrefixEngine)
		assert.Equal(t, "account_cache_performance", GetMetricName(emptyPrefixEngine.AccountCachePerformance))
	})
	t.Run("foobar prefix", func(t *testing.T) {
		emptyPrefixEngine, err := NewEngine("foobar", &config.DisabledMetrics{})
		require.NoError(t, err)
		assert.NotNil(t, emptyPrefixEngine)
		assert.Equal(t, "foobar.account_cache_performance", GetMetricName(emptyPrefixEngine.AccountCachePerformance))
	})
	t.Run("FooBar prefix", func(t *testing.T) {
		emptyPrefixEngine, err := NewEngine("FooBar", &config.DisabledMetrics{})
		require.NoError(t, err)
		assert.NotNil(t, emptyPrefixEngine)
		assert.Equal(t, "foo_bar.account_cache_performance", GetMetricName(emptyPrefixEngine.AccountCachePerformance))
	})
}

func TestRecordAdapterTime(t *testing.T) {
	emptyPrefixEngine, err := NewEngine("", &config.DisabledMetrics{})
	require.NoError(t, err)
	emptyPrefixEngine.RecordAdapterTime(metrics.AdapterLabels{}, time.Second)
}
