package config

import (
	"testing"
	"time"

	mainConfig "github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/rcrowley/go-metrics"
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
	_, ok := testEngine.MetricsEngine.(*pbsmetrics.Metrics)
	if !ok {
		t.Error("Expected a legacy Metrics as MetricsEngine, but didn't get it")
	}
}

// Test the multiengine
func TestMultiMetricsEngine(t *testing.T) {
	cfg := mainConfig.Configuration{}
	cfg.Metrics.Influxdb.Host = "localhost"
	adapterList := openrtb_ext.BidderList()
	goEngine := pbsmetrics.NewMetrics(metrics.NewPrefixedRegistry("prebidserver."), adapterList, mainConfig.DisabledMetrics{})
	engineList := make(MultiMetricsEngine, 2)
	engineList[0] = goEngine
	engineList[1] = &DummyMetricsEngine{}
	var metricsEngine pbsmetrics.MetricsEngine
	metricsEngine = &engineList
	labels := pbsmetrics.Labels{
		Source:        pbsmetrics.DemandWeb,
		RType:         pbsmetrics.ReqTypeORTB2Web,
		PubID:         "test1",
		CookieFlag:    pbsmetrics.CookieFlagYes,
		RequestStatus: pbsmetrics.RequestStatusOK,
	}
	apnLabels := pbsmetrics.AdapterLabels{
		Source:      pbsmetrics.DemandWeb,
		RType:       pbsmetrics.ReqTypeORTB2Web,
		Adapter:     openrtb_ext.BidderAppnexus,
		PubID:       "test1",
		CookieFlag:  pbsmetrics.CookieFlagYes,
		AdapterBids: pbsmetrics.AdapterBidNone,
	}
	pubLabels := pbsmetrics.AdapterLabels{
		Source:      pbsmetrics.DemandWeb,
		RType:       pbsmetrics.ReqTypeORTB2Web,
		Adapter:     openrtb_ext.BidderPubmatic,
		PubID:       "test1",
		CookieFlag:  pbsmetrics.CookieFlagYes,
		AdapterBids: pbsmetrics.AdapterBidPresent,
	}
	impTypeLabels := pbsmetrics.ImpLabels{
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
	labelsBlacklist := []pbsmetrics.Labels{
		{
			Source:        pbsmetrics.DemandWeb,
			RType:         pbsmetrics.ReqTypeAMP,
			PubID:         "test2",
			CookieFlag:    pbsmetrics.CookieFlagYes,
			RequestStatus: pbsmetrics.RequestStatusBlacklisted,
		},
		{
			Source:        pbsmetrics.DemandWeb,
			RType:         pbsmetrics.ReqTypeVideo,
			PubID:         "test2",
			CookieFlag:    pbsmetrics.CookieFlagYes,
			RequestStatus: pbsmetrics.RequestStatusBlacklisted,
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

	metricsEngine.RecordStoredReqCacheResult(pbsmetrics.CacheMiss, 1)
	metricsEngine.RecordStoredImpCacheResult(pbsmetrics.CacheMiss, 2)
	metricsEngine.RecordAccountCacheResult(pbsmetrics.CacheMiss, 3)
	metricsEngine.RecordStoredReqCacheResult(pbsmetrics.CacheHit, 4)
	metricsEngine.RecordStoredImpCacheResult(pbsmetrics.CacheHit, 5)
	metricsEngine.RecordAccountCacheResult(pbsmetrics.CacheHit, 6)

	metricsEngine.RecordRequestQueueTime(false, pbsmetrics.ReqTypeVideo, time.Duration(1))

	//Make the metrics engine, instantiated here with goEngine, fill its RequestStatuses[RequestType][pbsmetrics.RequestStatusXX] with the new boolean values added to pbsmetrics.Labels
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.OK", goEngine.RequestStatuses[pbsmetrics.ReqTypeORTB2Web][pbsmetrics.RequestStatusOK].Count(), 5)
	VerifyMetrics(t, "RequestStatuses.Legacy.OK", goEngine.RequestStatuses[pbsmetrics.ReqTypeLegacy][pbsmetrics.RequestStatusOK].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.AMP.OK", goEngine.RequestStatuses[pbsmetrics.ReqTypeAMP][pbsmetrics.RequestStatusOK].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.AMP.BlacklistedAcctOrApp", goEngine.RequestStatuses[pbsmetrics.ReqTypeAMP][pbsmetrics.RequestStatusBlacklisted].Count(), 1)
	VerifyMetrics(t, "RequestStatuses.Video.OK", goEngine.RequestStatuses[pbsmetrics.ReqTypeVideo][pbsmetrics.RequestStatusOK].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.Video.Error", goEngine.RequestStatuses[pbsmetrics.ReqTypeVideo][pbsmetrics.RequestStatusErr].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.Video.BadInput", goEngine.RequestStatuses[pbsmetrics.ReqTypeVideo][pbsmetrics.RequestStatusBadInput].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.Video.BlacklistedAcctOrApp", goEngine.RequestStatuses[pbsmetrics.ReqTypeVideo][pbsmetrics.RequestStatusBlacklisted].Count(), 1)
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.Error", goEngine.RequestStatuses[pbsmetrics.ReqTypeORTB2Web][pbsmetrics.RequestStatusErr].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.BadInput", goEngine.RequestStatuses[pbsmetrics.ReqTypeORTB2Web][pbsmetrics.RequestStatusBadInput].Count(), 0)
	VerifyMetrics(t, "RequestStatuses.OpenRTB2.BlacklistedAcctOrApp", goEngine.RequestStatuses[pbsmetrics.ReqTypeORTB2Web][pbsmetrics.RequestStatusBlacklisted].Count(), 0)

	VerifyMetrics(t, "ImpsTypeBanner", goEngine.ImpsTypeBanner.Count(), 5)
	VerifyMetrics(t, "ImpsTypeVideo", goEngine.ImpsTypeVideo.Count(), 3)
	VerifyMetrics(t, "ImpsTypeAudio", goEngine.ImpsTypeAudio.Count(), 5)
	VerifyMetrics(t, "ImpsTypeNative", goEngine.ImpsTypeNative.Count(), 5)

	VerifyMetrics(t, "RecordPrebidCacheRequestTime", goEngine.PrebidCacheRequestTimerSuccess.Count(), 5)

	VerifyMetrics(t, "Request", goEngine.RequestStatuses[pbsmetrics.ReqTypeORTB2Web][pbsmetrics.RequestStatusOK].Count(), 5)
	VerifyMetrics(t, "ImpMeter", goEngine.ImpMeter.Count(), 8)
	VerifyMetrics(t, "LegacyImpMeter", goEngine.LegacyImpMeter.Count(), 10)
	VerifyMetrics(t, "NoCookieMeter", goEngine.NoCookieMeter.Count(), 0)
	VerifyMetrics(t, "AdapterMetrics.Pubmatic.GotBidsMeter", goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].GotBidsMeter.Count(), 5)
	VerifyMetrics(t, "AdapterMetrics.Pubmatic.NoBidMeter", goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].NoBidMeter.Count(), 0)
	for _, err := range pbsmetrics.AdapterErrors() {
		VerifyMetrics(t, "AdapterMetrics.Pubmatic.Request.ErrorMeter."+string(err), goEngine.AdapterMetrics[openrtb_ext.BidderPubmatic].ErrorMeters[err].Count(), 0)
	}
	VerifyMetrics(t, "AdapterMetrics.AppNexus.GotBidsMeter", goEngine.AdapterMetrics[openrtb_ext.BidderAppnexus].GotBidsMeter.Count(), 0)
	VerifyMetrics(t, "AdapterMetrics.AppNexus.NoBidMeter", goEngine.AdapterMetrics[openrtb_ext.BidderAppnexus].NoBidMeter.Count(), 5)

	VerifyMetrics(t, "RecordRequestQueueTime.Video.Rejected", goEngine.RequestsQueueTimer[pbsmetrics.ReqTypeVideo][false].Count(), 1)
	VerifyMetrics(t, "RecordRequestQueueTime.Video.Accepted", goEngine.RequestsQueueTimer[pbsmetrics.ReqTypeVideo][true].Count(), 0)

	VerifyMetrics(t, "StoredReqCache.Miss", goEngine.StoredReqCacheMeter[pbsmetrics.CacheMiss].Count(), 1)
	VerifyMetrics(t, "StoredImpCache.Miss", goEngine.StoredImpCacheMeter[pbsmetrics.CacheMiss].Count(), 2)
	VerifyMetrics(t, "AccountCache.Miss", goEngine.AccountCacheMeter[pbsmetrics.CacheMiss].Count(), 3)
	VerifyMetrics(t, "StoredReqCache.Hit", goEngine.StoredReqCacheMeter[pbsmetrics.CacheHit].Count(), 4)
	VerifyMetrics(t, "StoredImpCache.Hit", goEngine.StoredImpCacheMeter[pbsmetrics.CacheHit].Count(), 5)
	VerifyMetrics(t, "AccountCache.Hit", goEngine.AccountCacheMeter[pbsmetrics.CacheHit].Count(), 6)
}

func VerifyMetrics(t *testing.T, name string, actual int64, expected int64) {
	if expected != actual {
		t.Errorf("Error in metric %s: got %d, expected %d.", name, actual, expected)
	}
}
