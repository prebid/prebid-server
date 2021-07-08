package config

import (
	"testing"
	"time"

	mainConfig "github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	gometrics "github.com/rcrowley/go-metrics"
)

// Start a simple test to insure we get valid MetricsEngines for various configurations
func TestDummyMetricsEngine(t *testing.T) {
	cfg := mainConfig.Configuration{}
	adapterList := make([]openrtb_ext.BidderName, 0, 2)
	testEngine := NewMetricsEngine(&cfg, adapterList)
	_, ok := testEngine.MetricsEngine.(*DummyMetricsEngine)
	if !ok {
		t.Error("Expected a DummyMetricsEngine, but didn't get it")
	}
}

func TestGoMetricsEngine(t *testing.T) {
	cfg := mainConfig.Configuration{}
	cfg.Metrics.Influxdb.Host = "localhost"
	adapterList := make([]openrtb_ext.BidderName, 0, 2)
	testEngine := NewMetricsEngine(&cfg, adapterList)
	_, ok := testEngine.MetricsEngine.(*metrics.Metrics)
	if !ok {
		t.Error("Expected a legacy Metrics as MetricsEngine, but didn't get it")
	}
}

// Test the multiengine
func TestMultiMetricsEngine(t *testing.T) {
	cfg := mainConfig.Configuration{}
	cfg.Metrics.Influxdb.Host = "localhost"
	adapterList := openrtb_ext.CoreBidderNames()
	goEngine := metrics.NewMetrics(gometrics.NewPrefixedRegistry("prebidserver."), adapterList, mainConfig.DisabledMetrics{})
	engineList := make(MultiMetricsEngine, 2)
	engineList[0] = goEngine
	engineList[1] = &DummyMetricsEngine{}
	var metricsEngine metrics.MetricsEngine
	metricsEngine = &engineList
	labels := metrics.Labels{
		Source:        metrics.DemandWeb,
		RType:         metrics.ReqTypeORTB2Web,
		PubID:         "test1",
		CookieFlag:    metrics.CookieFlagYes,
		RequestStatus: metrics.RequestStatusOK,
	}
	apnLabels := metrics.AdapterLabels{
		Source:      metrics.DemandWeb,
		RType:       metrics.ReqTypeORTB2Web,
		Adapter:     openrtb_ext.BidderAppnexus,
		PubID:       "test1",
		CookieFlag:  metrics.CookieFlagYes,
		AdapterBids: metrics.AdapterBidNone,
	}
	pubLabels := metrics.AdapterLabels{
		Source:      metrics.DemandWeb,
		RType:       metrics.ReqTypeORTB2Web,
		Adapter:     openrtb_ext.BidderPubmatic,
		PubID:       "test1",
		CookieFlag:  metrics.CookieFlagYes,
		AdapterBids: metrics.AdapterBidPresent,
	}
	impTypeLabels := metrics.ImpLabels{
		BannerImps: true,
		VideoImps:  false,
		AudioImps:  true,
		NativeImps: true,
	}
	for i := 0; i < 5; i++ {
		metricsEngine.RecordRequest(labels)
		metricsEngine.RecordImps(impTypeLabels)
		metricsEngine.RecordLegacyImps(labels, 2)
		metricsEngine.RecordRequestTime(labels, time.Millisecond*20)
		metricsEngine.RecordAdapterRequest(pubLabels)
		metricsEngine.RecordAdapterRequest(apnLabels)
		metricsEngine.RecordAdapterPrice(pubLabels, 1.34)
		metricsEngine.RecordAdapterBidReceived(pubLabels, openrtb_ext.BidTypeBanner, true)
		metricsEngine.RecordAdapterTime(pubLabels, time.Millisecond*20)
		metricsEngine.RecordPrebidCacheRequestTime(true, time.Millisecond*20)
	}
	labelsBlacklist := []metrics.Labels{
		{
			Source:        metrics.DemandWeb,
			RType:         metrics.ReqTypeAMP,
			PubID:         "test2",
			CookieFlag:    metrics.CookieFlagYes,
			RequestStatus: metrics.RequestStatusBlacklisted,
		},
		{
			Source:        metrics.DemandWeb,
			RType:         metrics.ReqTypeVideo,
			PubID:         "test2",
			CookieFlag:    metrics.CookieFlagYes,
			RequestStatus: metrics.RequestStatusBlacklisted,
		},
	}
	for _, label := range labelsBlacklist {
		metricsEngine.RecordRequest(label)
	}
	impTypeLabels.BannerImps = false
	impTypeLabels.VideoImps = true
	impTypeLabels.AudioImps = false
	impTypeLabels.NativeImps = false
	for i := 0; i < 3; i++ {
		metricsEngine.RecordImps(impTypeLabels)
	}

	metricsEngine.RecordStoredReqCacheResult(metrics.CacheMiss, 1)
	metricsEngine.RecordStoredImpCacheResult(metrics.CacheMiss, 2)
	metricsEngine.RecordAccountCacheResult(metrics.CacheMiss, 3)
	metricsEngine.RecordStoredReqCacheResult(metrics.CacheHit, 4)
	metricsEngine.RecordStoredImpCacheResult(metrics.CacheHit, 5)
	metricsEngine.RecordAccountCacheResult(metrics.CacheHit, 6)

	metricsEngine.RecordAdapterGDPRRequestBlocked(openrtb_ext.BidderAppnexus)

	metricsEngine.RecordRequestQueueTime(false, metrics.ReqTypeVideo, time.Duration(1))

	//Make the metrics engine, instantiated here with goEngine, fill its RequestStatuses[RequestType][metrics.RequestStatusXX] with the new boolean values added to metrics.Labels
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.OK", goEngine.RequestStatuses[metrics.ReqTypeORTB2Web][metrics.RequestStatusOK].Count(), 5)
	VerifyMetrics(t, "RequestStatuses.Legacy.OK", goEngine.RequestStatuses[metrics.ReqTypeLegacy][metrics.RequestStatusOK].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.AMP.OK", goEngine.RequestStatuses[metrics.ReqTypeAMP][metrics.RequestStatusOK].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.AMP.BlacklistedAcctOrApp", goEngine.RequestStatuses[metrics.ReqTypeAMP][metrics.RequestStatusBlacklisted].Count(), 1)
	VerifyMetrics(t, "RequestStatuses.Video.OK", goEngine.RequestStatuses[metrics.ReqTypeVideo][metrics.RequestStatusOK].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.Video.Error", goEngine.RequestStatuses[metrics.ReqTypeVideo][metrics.RequestStatusErr].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.Video.BadInput", goEngine.RequestStatuses[metrics.ReqTypeVideo][metrics.RequestStatusBadInput].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.Video.BlacklistedAcctOrApp", goEngine.RequestStatuses[metrics.ReqTypeVideo][metrics.RequestStatusBlacklisted].Count(), 1)
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.Error", goEngine.RequestStatuses[metrics.ReqTypeORTB2Web][metrics.RequestStatusErr].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.BadInput", goEngine.RequestStatuses[metrics.ReqTypeORTB2Web][metrics.RequestStatusBadInput].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.BlacklistedAcctOrApp", goEngine.RequestStatuses[metrics.ReqTypeORTB2Web][metrics.RequestStatusBlacklisted].Count(), 0)

	VerifyMetrics(t, "ImpsTypeBanner", goEngine.ImpsTypeBanner.Count(), 5)
	VerifyMetrics(t, "ImpsTypeVideo", goEngine.ImpsTypeVideo.Count(), 3)
	VerifyMetrics(t, "ImpsTypeAudio", goEngine.ImpsTypeAudio.Count(), 5)
	VerifyMetrics(t, "ImpsTypeNative", goEngine.ImpsTypeNative.Count(), 5)

	VerifyMetrics(t, "RecordPrebidCacheRequestTime", goEngine.PrebidCacheRequestTimerSuccess.Count(), 5)

	VerifyMetrics(t, "Request", goEngine.RequestStatuses[metrics.ReqTypeORTB2Web][metrics.RequestStatusOK].Count(), 5)
	VerifyMetrics(t, "ImpMeter", goEngine.ImpMeter.Count(), 8)
	VerifyMetrics(t, "LegacyImpMeter", goEngine.LegacyImpMeter.Count(), 10)
	VerifyMetrics(t, "NoCookieMeter", goEngine.NoCookieMeter.Count(), 0)
	VerifyMetrics(t, "AdapterMetrics.Pubmatic.GotBidsMeter", goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].GotBidsMeter.Count(), 5)
	VerifyMetrics(t, "AdapterMetrics.Pubmatic.NoBidMeter", goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].NoBidMeter.Count(), 0)
	for _, err := range metrics.AdapterErrors() {
		VerifyMetrics(t, "AdapterMetrics.Pubmatic.Request.ErrorMeter."+string(err), goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].ErrorMeters[err].Count(), 0)
	}
	VerifyMetrics(t, "AdapterMetrics.AppNexus.GotBidsMeter", goEngine.AdapterMetrics[openrtb_ext.BidderAppnexus].GotBidsMeter.Count(), 0)
	VerifyMetrics(t, "AdapterMetrics.AppNexus.NoBidMeter", goEngine.AdapterMetrics[openrtb_ext.BidderAppnexus].NoBidMeter.Count(), 5)

	VerifyMetrics(t, "RecordRequestQueueTime.Video.Rejected", goEngine.RequestsQueueTimer[metrics.ReqTypeVideo][false].Count(), 1)
	VerifyMetrics(t, "RecordRequestQueueTime.Video.Accepted", goEngine.RequestsQueueTimer[metrics.ReqTypeVideo][true].Count(), 0)

	VerifyMetrics(t, "StoredReqCache.Miss", goEngine.StoredReqCacheMeter[metrics.CacheMiss].Count(), 1)
	VerifyMetrics(t, "StoredImpCache.Miss", goEngine.StoredImpCacheMeter[metrics.CacheMiss].Count(), 2)
	VerifyMetrics(t, "AccountCache.Miss", goEngine.AccountCacheMeter[metrics.CacheMiss].Count(), 3)
	VerifyMetrics(t, "StoredReqCache.Hit", goEngine.StoredReqCacheMeter[metrics.CacheHit].Count(), 4)
	VerifyMetrics(t, "StoredImpCache.Hit", goEngine.StoredImpCacheMeter[metrics.CacheHit].Count(), 5)
	VerifyMetrics(t, "AccountCache.Hit", goEngine.AccountCacheMeter[metrics.CacheHit].Count(), 6)

	VerifyMetrics(t, "AdapterMetrics.AppNexus.GDPRRequestBlocked", goEngine.AdapterMetrics[openrtb_ext.BidderAppnexus].GDPRRequestBlocked.Count(), 1)
}

func VerifyMetrics(t *testing.T, name string, actual int64, expected int64) {
	if expected != actual {
		t.Errorf("Error in metric %s: got %d, expected %d.", name, actual, expected)
	}
}
