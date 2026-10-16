package config

import (
	mainConfig "github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/metrics"
	prometheusmetrics "github.com/prebid/prebid-server/v4/metrics/prometheus"
	gometrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVastlintCountersReachEveryBackend(t *testing.T) {
	registry := gometrics.NewRegistry()
	goEngine := metrics.NewMetrics(registry, nil, mainConfig.DisabledMetrics{}, nil, nil)
	prom := prometheusmetrics.NewMetrics(mainConfig.PrometheusMetrics{}, mainConfig.DisabledMetrics{}, nil, nil)
	engines := MultiMetricsEngine{goEngine, prom, &NilMetricsEngine{}}
	engines.RecordVastlintEvent("checked", 3)
	engines.RecordVastlintEvent("creative-secret", 1)
	require.EqualValues(t, 3, goEngine.VastlintCounters["checked"].Count())
	require.Nil(t, registry.Get("vastlint.creative-secret"))
	families, err := prom.Gatherer.Gather()
	require.NoError(t, err)
	found := false
	for _, family := range families {
		if family.GetName() == "vastlint_collection_total" {
			require.Len(t, family.Metric, len(metrics.VastlintEvents()))
			for _, sample := range family.Metric {
				if sample.Label[0].GetValue() == "checked" {
					found = true
					require.Equal(t, float64(3), sample.Counter.GetValue())
				}
			}
		}
	}
	require.True(t, found)
}
