package pbsmetrics

import (
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

// Start a simple test to insure we get valid MetricsEngines for various configurations
func TestDummyMetricsEngine(t *testing.T) {
	cfg := config.Configuration{}
	adapterList := make([]openrtb_ext.BidderName, 0, 2)
	testEngine := NewMetricsEngine(&cfg, adapterList)
	_, ok := testEngine.(*DummyMetricsEngine)
	if !ok {
		t.Error("Expected a DummyMetricsEngine, but didn't get it")
	}
}

func TestGoMetricsEngine(t *testing.T) {
	cfg := config.Configuration{}
	cfg.Metrics.GoMetrics.Enabled = "yes"
	adapterList := make([]openrtb_ext.BidderName, 0, 2)
	testEngine := NewMetricsEngine(&cfg, adapterList)
	_, ok := testEngine.(*Metrics)
	if !ok {
		t.Error("Expected a legacy Metrics as MetricsEngine, but didn't get it")
	}
}
