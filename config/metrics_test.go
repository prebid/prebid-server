package config

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/rcrowley/go-metrics"
)

// Start a simple test to insure we get valid MetricsEngines for various configurations
func TestDummyMetricsEngine(t *testing.T) {
	cfg := Configuration{}
	adapterList := make([]openrtb_ext.BidderName, 0, 2)
	testEngine := NewMetricsEngine(&cfg, adapterList)
	_, ok := testEngine.(*DummyMetricsEngine)
	if !ok {
		t.Error("Expected a DummyMetricsEngine, but didn't get it")
	}
}

func TestGoMetricsEngine(t *testing.T) {
	cfg := Configuration{}
	cfg.Metrics.Influxdb.Host = "localhost"
	adapterList := make([]openrtb_ext.BidderName, 0, 2)
	testEngine := NewMetricsEngine(&cfg, adapterList)
	_, ok := testEngine.(*pbsmetrics.Metrics)
	if !ok {
		t.Error("Expected a legacy Metrics as MetricsEngine, but didn't get it")
	}
}

// Test the multiengine
func TestMultiMetricsEngine(t *testing.T) {
	cfg := Configuration{}
	cfg.Metrics.Influxdb.Host = "localhost"
	adapterList := openrtb_ext.BidderList()
	goEngine := pbsmetrics.NewMetrics(metrics.NewPrefixedRegistry("prebidserver."), adapterList)
	engineList := make(MultiMetricsEngine, 2)
	engineList[0] = goEngine
	engineList[1] = &DummyMetricsEngine{}
	var metricsEngine pbsmetrics.MetricsEngine
	metricsEngine = &engineList
	labels := pbsmetrics.Labels{
		Source:        pbsmetrics.DemandWeb,
		RType:         pbsmetrics.ReqTypeORTB2,
		PubID:         "test1",
		Browser:       pbsmetrics.BrowserSafari,
		CookieFlag:    pbsmetrics.CookieFlagYes,
		RequestStatus: pbsmetrics.RequestStatusOK,
	}
	blabels := pbsmetrics.AdapterLabels{
		Source:        pbsmetrics.DemandWeb,
		RType:         pbsmetrics.ReqTypeORTB2,
		Adapter:       openrtb_ext.BidderPubmatic,
		PubID:         "test1",
		Browser:       pbsmetrics.BrowserSafari,
		CookieFlag:    pbsmetrics.CookieFlagYes,
		AdapterStatus: pbsmetrics.AdapterStatusOK,
	}
	for i := 0; i < 5; i++ {
		metricsEngine.RecordRequest(labels)
		metricsEngine.RecordImps(labels, 2)
		metricsEngine.RecordRequestTime(labels, time.Millisecond*20)
		metricsEngine.RecordAdapterRequest(blabels)
		metricsEngine.RecordAdapterPrice(blabels, 1.34)
		metricsEngine.RecordAdapterTime(blabels, time.Millisecond*20)
	}
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.OK", goEngine.RequestStatuses[pbsmetrics.ReqTypeORTB2][pbsmetrics.RequestStatusOK].Count(), 5)
	VerifyMetrics(t, "RequestStatuses.Legacy.OK", goEngine.RequestStatuses[pbsmetrics.ReqTypeLegacy][pbsmetrics.RequestStatusOK].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.AMP.OK", goEngine.RequestStatuses[pbsmetrics.ReqTypeAMP][pbsmetrics.RequestStatusOK].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.Error", goEngine.RequestStatuses[pbsmetrics.ReqTypeORTB2][pbsmetrics.RequestStatusErr].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.BadInput", goEngine.RequestStatuses[pbsmetrics.ReqTypeORTB2][pbsmetrics.RequestStatusBadInput].Count(), 0)
	VerifyMetrics(t, "Request", goEngine.RequestStatuses[pbsmetrics.ReqTypeORTB2][pbsmetrics.RequestStatusOK].Count(), 5)
	VerifyMetrics(t, "ImpMeter", goEngine.ImpMeter.Count(), 10)
	VerifyMetrics(t, "NoCookieMeter", goEngine.NoCookieMeter.Count(), 0)
	VerifyMetrics(t, "SafariRequestMeter", goEngine.SafariRequestMeter.Count(), 5)
	VerifyMetrics(t, "SafariNoCookieMeter", goEngine.SafariNoCookieMeter.Count(), 0)
	VerifyMetrics(t, "AdapterMetrics.Pubmatic.RequestMeter", goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].RequestMeter.Count(), 5)
	VerifyMetrics(t, "AdapterMetrics.Pubmatic.ErrorMeter", goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].ErrorMeter.Count(), 0)
	VerifyMetrics(t, "AdapterMetrics.AppNexus.RequestMeter", goEngine.AdapterMetrics[openrtb_ext.BidderAppnexus].RequestMeter.Count(), 0)
}

func VerifyMetrics(t *testing.T, name string, expected int64, actual int64) {
	if expected != actual {
		t.Errorf("Error in metric %s: expected %d, got %d.", name, expected, actual)
	}
}
