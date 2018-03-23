package pbsmetrics

import (
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/rcrowley/go-metrics"
	"testing"
	"time"
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
	cfg.Metrics.Influxdb.Host = "localhost"
	adapterList := make([]openrtb_ext.BidderName, 0, 2)
	testEngine := NewMetricsEngine(&cfg, adapterList)
	_, ok := testEngine.(*Metrics)
	if !ok {
		t.Error("Expected a legacy Metrics as MetricsEngine, but didn't get it")
	}
}

// Test the multiengine
func TestMultiMetricsEngine(t *testing.T) {
	cfg := config.Configuration{}
	cfg.Metrics.Influxdb.Host = "localhost"
	adapterList := openrtb_ext.BidderList()
	goEngine := NewMetrics(metrics.NewPrefixedRegistry("prebidserver."), adapterList)
	engineList := make(MultiMetricsEngine, 2)
	engineList[0] = goEngine
	engineList[1] = &DummyMetricsEngine{}
	var metricsEngine MetricsEngine
	metricsEngine = &engineList
	labels := Labels{
		Source:        DemandWeb,
		RType:         ReqTypeORTB2,
		PubID:         "test1",
		Browser:       BrowserSafari,
		CookieFlag:    CookieFlagYes,
		RequestStatus: RequestStatusOK,
	}
	blabels := AdapterLabels{
		Source:        DemandWeb,
		RType:         ReqTypeORTB2,
		Adapter:       openrtb_ext.BidderPubmatic,
		PubID:         "test1",
		Browser:       BrowserSafari,
		CookieFlag:    CookieFlagYes,
		AdapterStatus: AdapterStatusOK,
	}
	for i := 0; i < 5; i++ {
		metricsEngine.RecordRequest(labels)
		metricsEngine.RecordRequestTime(labels, time.Millisecond*20)
		metricsEngine.RecordAdapterRequest(blabels)
		metricsEngine.RecordAdapterPrice(blabels, 1.34)
		metricsEngine.RecordAdapterBidsReceived(blabels, 2)
		metricsEngine.RecordAdapterTime(blabels, time.Millisecond*20)
	}
	VerifyMetrics(t, "RequestMeter", goEngine.RequestMeter.Count(), 5)
	VerifyMetrics(t, "AppRequestMeter", goEngine.AmpRequestMeter.Count(), 0)
	VerifyMetrics(t, "NoCookieMeter", goEngine.NoCookieMeter.Count(), 0)
	VerifyMetrics(t, "SafariRequestMeter", goEngine.SafariRequestMeter.Count(), 5)
	VerifyMetrics(t, "SafariNoCookieMeter", goEngine.SafariNoCookieMeter.Count(), 0)
	VerifyMetrics(t, "ErrorMeter", goEngine.ErrorMeter.Count(), 0)
	VerifyMetrics(t, "ORTBRequestMeter", goEngine.ORTBRequestMeter.Count(), 5)
	VerifyMetrics(t, "AmpRequestMeter", goEngine.AmpRequestMeter.Count(), 0)
	VerifyMetrics(t, "AdapterMetrics.Pubmatic.RequestMeter", goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].RequestMeter.Count(), 5)
	VerifyMetrics(t, "AdapterMetrics.Pubmatic.ErrorMeter", goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].ErrorMeter.Count(), 0)
	VerifyMetrics(t, "AdapterMetrics.AppNexus.RequestMeter", goEngine.AdapterMetrics[openrtb_ext.BidderAppnexus].RequestMeter.Count(), 0)
	VerifyMetrics(t, "AccountMetrics.test1.RequestMeter", goEngine.accountMetrics["test1"].requestMeter.Count(), 5)
}

func VerifyMetrics(t *testing.T, name string, expected int64, actual int64) {
	if expected != actual {
		t.Errorf("Error in metric %s: expected %d, got %d.", name, expected, actual)
	}
}
