package pbsmetrics

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/rcrowley/go-metrics"
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
		RType:         ReqTypeORTB2Web,
		PubID:         "test1",
		Browser:       BrowserSafari,
		CookieFlag:    CookieFlagYes,
		RequestStatus: RequestStatusOK,
	}
	apnLabels := AdapterLabels{
		Source:      DemandWeb,
		RType:       ReqTypeORTB2Web,
		Adapter:     openrtb_ext.BidderAppnexus,
		PubID:       "test1",
		Browser:     BrowserSafari,
		CookieFlag:  CookieFlagYes,
		AdapterBids: AdapterBidNone,
	}
	pubLabels := AdapterLabels{
		Source:      DemandWeb,
		RType:       ReqTypeORTB2Web,
		Adapter:     openrtb_ext.BidderPubmatic,
		PubID:       "test1",
		Browser:     BrowserSafari,
		CookieFlag:  CookieFlagYes,
		AdapterBids: AdapterBidPresent,
	}
	for i := 0; i < 5; i++ {
		metricsEngine.RecordRequest(labels)
		metricsEngine.RecordImps(labels, 2)
		metricsEngine.RecordRequestTime(labels, time.Millisecond*20)
		metricsEngine.RecordAdapterRequest(pubLabels)
		metricsEngine.RecordAdapterRequest(apnLabels)
		metricsEngine.RecordAdapterPrice(pubLabels, 1.34)
		metricsEngine.RecordAdapterBidsReceived(pubLabels, 2)
		metricsEngine.RecordAdapterTime(pubLabels, time.Millisecond*20)
	}
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.OK", goEngine.RequestStatuses[ReqTypeORTB2Web][RequestStatusOK].Count(), 5)
	VerifyMetrics(t, "RequestStatuses.Legacy.OK", goEngine.RequestStatuses[ReqTypeLegacy][RequestStatusOK].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.AMP.OK", goEngine.RequestStatuses[ReqTypeAMP][RequestStatusOK].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.Error", goEngine.RequestStatuses[ReqTypeORTB2Web][RequestStatusErr].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.BadInput", goEngine.RequestStatuses[ReqTypeORTB2Web][RequestStatusBadInput].Count(), 0)
	VerifyMetrics(t, "Request", goEngine.RequestStatuses[ReqTypeORTB2Web][RequestStatusOK].Count(), 5)
	VerifyMetrics(t, "ImpMeter", goEngine.ImpMeter.Count(), 10)
	VerifyMetrics(t, "NoCookieMeter", goEngine.NoCookieMeter.Count(), 0)
	VerifyMetrics(t, "SafariRequestMeter", goEngine.SafariRequestMeter.Count(), 5)
	VerifyMetrics(t, "SafariNoCookieMeter", goEngine.SafariNoCookieMeter.Count(), 0)
	VerifyMetrics(t, "AdapterMetrics.Pubmatic.GotBidsMeter", goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].GotBidsMeter.Count(), 5)
	VerifyMetrics(t, "AdapterMetrics.Pubmatic.NoBidMeter", goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].NoBidMeter.Count(), 0)
	for _, err := range AdapterErrors() {
		VerifyMetrics(t, "AdapterMetrics.Pubmatic.Request.ErrorMeter."+string(err), goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].ErrorMeters[err].Count(), 0)
	}
	VerifyMetrics(t, "AdapterMetrics.AppNexus.GotBidsMeter", goEngine.AdapterMetrics[openrtb_ext.BidderAppnexus].GotBidsMeter.Count(), 0)
	VerifyMetrics(t, "AdapterMetrics.AppNexus.NoBidMeter", goEngine.AdapterMetrics[openrtb_ext.BidderAppnexus].NoBidMeter.Count(), 5)
	VerifyMetrics(t, "AccountMetrics.test1.RequestMeter", goEngine.accountMetrics["test1"].requestMeter.Count(), 5)
}

func VerifyMetrics(t *testing.T, name string, expected int64, actual int64) {
	if expected != actual {
		t.Errorf("Error in metric %s: expected %d, got %d.", name, expected, actual)
	}
}
