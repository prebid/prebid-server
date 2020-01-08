package pbsmetrics

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{})

	ensureContains(t, registry, "app_requests", m.AppRequestMeter)
	ensureContains(t, registry, "no_cookie_requests", m.NoCookieMeter)
	ensureContains(t, registry, "safari_requests", m.SafariRequestMeter)
	ensureContains(t, registry, "safari_no_cookie_requests", m.SafariNoCookieMeter)
	ensureContains(t, registry, "request_time", m.RequestTimer)
	ensureContains(t, registry, "amp_no_cookie_requests", m.AmpNoCookieMeter)
	ensureContainsAdapterMetrics(t, registry, "adapter.appnexus", m.AdapterMetrics["appnexus"])
	ensureContainsAdapterMetrics(t, registry, "adapter.rubicon", m.AdapterMetrics["rubicon"])
	ensureContains(t, registry, "cookie_sync_requests", m.CookieSyncMeter)
	ensureContains(t, registry, "cookie_sync.appnexus.gen", m.CookieSyncGen["appnexus"])
	ensureContains(t, registry, "cookie_sync.appnexus.gdpr_prevent", m.CookieSyncGDPRPrevent["appnexus"])
	ensureContains(t, registry, "usersync.appnexus.gdpr_prevent", m.userSyncGDPRPrevent["appnexus"])
	ensureContains(t, registry, "usersync.rubicon.gdpr_prevent", m.userSyncGDPRPrevent["rubicon"])
	ensureContains(t, registry, "usersync.unknown.gdpr_prevent", m.userSyncGDPRPrevent["unknown"])
	ensureContains(t, registry, "prebid_cache_request_time.ok", m.PrebidCacheRequestTimerSuccess)
	ensureContains(t, registry, "prebid_cache_request_time.err", m.PrebidCacheRequestTimerError)

	ensureContains(t, registry, "requests.ok.legacy", m.RequestStatuses[ReqTypeLegacy][RequestStatusOK])
	ensureContains(t, registry, "requests.badinput.legacy", m.RequestStatuses[ReqTypeLegacy][RequestStatusBadInput])
	ensureContains(t, registry, "requests.err.legacy", m.RequestStatuses[ReqTypeLegacy][RequestStatusErr])
	ensureContains(t, registry, "requests.networkerr.legacy", m.RequestStatuses[ReqTypeLegacy][RequestStatusNetworkErr])
	ensureContains(t, registry, "requests.ok.openrtb2-web", m.RequestStatuses[ReqTypeORTB2Web][RequestStatusOK])
	ensureContains(t, registry, "requests.badinput.openrtb2-web", m.RequestStatuses[ReqTypeORTB2Web][RequestStatusBadInput])
	ensureContains(t, registry, "requests.err.openrtb2-web", m.RequestStatuses[ReqTypeORTB2Web][RequestStatusErr])
	ensureContains(t, registry, "requests.networkerr.openrtb2-web", m.RequestStatuses[ReqTypeORTB2Web][RequestStatusNetworkErr])
	ensureContains(t, registry, "requests.ok.openrtb2-app", m.RequestStatuses[ReqTypeORTB2App][RequestStatusOK])
	ensureContains(t, registry, "requests.badinput.openrtb2-app", m.RequestStatuses[ReqTypeORTB2App][RequestStatusBadInput])
	ensureContains(t, registry, "requests.err.openrtb2-app", m.RequestStatuses[ReqTypeORTB2App][RequestStatusErr])
	ensureContains(t, registry, "requests.networkerr.openrtb2-app", m.RequestStatuses[ReqTypeORTB2App][RequestStatusNetworkErr])
	ensureContains(t, registry, "requests.ok.amp", m.RequestStatuses[ReqTypeAMP][RequestStatusOK])
	ensureContains(t, registry, "requests.badinput.amp", m.RequestStatuses[ReqTypeAMP][RequestStatusBadInput])
	ensureContains(t, registry, "requests.err.amp", m.RequestStatuses[ReqTypeAMP][RequestStatusErr])
	ensureContains(t, registry, "requests.networkerr.amp", m.RequestStatuses[ReqTypeAMP][RequestStatusNetworkErr])
	ensureContains(t, registry, "requests.ok.video", m.RequestStatuses[ReqTypeVideo][RequestStatusOK])
	ensureContains(t, registry, "requests.badinput.video", m.RequestStatuses[ReqTypeVideo][RequestStatusBadInput])
	ensureContains(t, registry, "requests.err.video", m.RequestStatuses[ReqTypeVideo][RequestStatusErr])
	ensureContains(t, registry, "requests.networkerr.video", m.RequestStatuses[ReqTypeVideo][RequestStatusNetworkErr])
}

func TestRecordBidType(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{})

	m.RecordAdapterBidReceived(AdapterLabels{
		Adapter: openrtb_ext.BidderAppnexus,
	}, openrtb_ext.BidTypeBanner, true)
	VerifyMetrics(t, "Appnexus Banner Adm Bids", m.AdapterMetrics[openrtb_ext.BidderAppnexus].MarkupMetrics[openrtb_ext.BidTypeBanner].AdmMeter.Count(), 1)
	VerifyMetrics(t, "Appnexus Banner Nurl Bids", m.AdapterMetrics[openrtb_ext.BidderAppnexus].MarkupMetrics[openrtb_ext.BidTypeBanner].NurlMeter.Count(), 0)

	m.RecordAdapterBidReceived(AdapterLabels{
		Adapter: openrtb_ext.BidderAppnexus,
	}, openrtb_ext.BidTypeVideo, false)
	VerifyMetrics(t, "Appnexus Video Adm Bids", m.AdapterMetrics[openrtb_ext.BidderAppnexus].MarkupMetrics[openrtb_ext.BidTypeVideo].AdmMeter.Count(), 0)
	VerifyMetrics(t, "Appnexus Video Nurl Bids", m.AdapterMetrics[openrtb_ext.BidderAppnexus].MarkupMetrics[openrtb_ext.BidTypeVideo].NurlMeter.Count(), 1)
}

func TestRecordGDPRRejection(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{})
	m.RecordUserIDSet(UserLabels{
		Action: RequestActionGDPR,
		Bidder: openrtb_ext.BidderAppnexus,
	})
	VerifyMetrics(t, "GDPR sync rejects", m.userSyncGDPRPrevent[openrtb_ext.BidderAppnexus].Count(), 1)
}

func ensureContains(t *testing.T, registry metrics.Registry, name string, metric interface{}) {
	t.Helper()
	if inRegistry := registry.Get(name); inRegistry == nil {
		t.Errorf("No metric in registry at %s.", name)
	} else if inRegistry != metric {
		t.Errorf("Bad value stored at metric %s.", name)
	}
}

func ensureContainsAdapterMetrics(t *testing.T, registry metrics.Registry, name string, adapterMetrics *AdapterMetrics) {
	t.Helper()
	ensureContains(t, registry, name+".no_cookie_requests", adapterMetrics.NoCookieMeter)
	ensureContains(t, registry, name+".requests.gotbids", adapterMetrics.GotBidsMeter)
	ensureContains(t, registry, name+".requests.nobid", adapterMetrics.NoBidMeter)
	ensureContains(t, registry, name+".requests.badinput", adapterMetrics.ErrorMeters[AdapterErrorBadInput])
	ensureContains(t, registry, name+".requests.badserverresponse", adapterMetrics.ErrorMeters[AdapterErrorBadServerResponse])
	ensureContains(t, registry, name+".requests.timeout", adapterMetrics.ErrorMeters[AdapterErrorTimeout])
	ensureContains(t, registry, name+".requests.unknown_error", adapterMetrics.ErrorMeters[AdapterErrorUnknown])

	ensureContains(t, registry, name+".request_time", adapterMetrics.RequestTimer)
	ensureContains(t, registry, name+".prices", adapterMetrics.PriceHistogram)
	ensureContainsBidTypeMetrics(t, registry, name, adapterMetrics.MarkupMetrics)
}

func TestRecordBidTypeDisabledConfig(t *testing.T) {
	testCases := []struct {
		hasAdm                 bool
		DisabledMetrics        config.DisabledMetrics
		ExpectedAdmMeterCount  int64
		ExpectedNurlMeterCount int64
		BidType                openrtb_ext.BidType
		PubID                  string
	}{
		{
			hasAdm:                 true,
			DisabledMetrics:        config.DisabledMetrics{},
			ExpectedAdmMeterCount:  1,
			ExpectedNurlMeterCount: 0,
			BidType:                openrtb_ext.BidTypeBanner,
			PubID:                  "acct-id",
		},
		{
			hasAdm:                 false,
			DisabledMetrics:        config.DisabledMetrics{},
			ExpectedAdmMeterCount:  0,
			ExpectedNurlMeterCount: 1,
			BidType:                openrtb_ext.BidTypeVideo,
			PubID:                  "acct-id",
		},
		{
			hasAdm:                 false,
			DisabledMetrics:        config.DisabledMetrics{AccountAdapterDetails: true},
			ExpectedAdmMeterCount:  0,
			ExpectedNurlMeterCount: 1,
			BidType:                openrtb_ext.BidTypeVideo,
			PubID:                  "acct-id",
		},
		{
			hasAdm:                 true,
			DisabledMetrics:        config.DisabledMetrics{AccountAdapterDetails: true},
			ExpectedAdmMeterCount:  1,
			ExpectedNurlMeterCount: 0,
			BidType:                openrtb_ext.BidTypeBanner,
			PubID:                  "acct-id",
		},
	}

	for _, test := range testCases {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, test.DisabledMetrics)

		m.RecordAdapterBidReceived(AdapterLabels{
			Adapter: openrtb_ext.BidderAppnexus,
			PubID:   test.PubID,
		}, test.BidType, test.hasAdm)
		assert.Equal(t, test.ExpectedAdmMeterCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].MarkupMetrics[test.BidType].AdmMeter.Count(), "Appnexus Banner Adm Bids")
		assert.Equal(t, test.ExpectedNurlMeterCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].MarkupMetrics[test.BidType].NurlMeter.Count(), "Appnexus Banner Nurl Bids")

		if test.DisabledMetrics.AccountAdapterDetails {
			assert.Len(t, m.accountMetrics[test.PubID].adapterMetrics, 0, "Test failed. Account metrics that contain adapter information are disabled, therefore we expect no entries in m.accountMetrics[accountId].adapterMetrics, we have %d \n", len(m.accountMetrics[test.PubID].adapterMetrics))
		} else {
			assert.NotEqual(t, 0, len(m.accountMetrics[test.PubID].adapterMetrics), "Test failed. Account metrics that contain adapter information are disabled, therefore we expect no entries in m.accountMetrics[accountId].adapterMetrics, we have %d \n", len(m.accountMetrics[test.PubID].adapterMetrics))
		}
	}
}

func TestNewMetricsWithDisabledConfig(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{AccountAdapterDetails: true})

	assert.True(t, m.MetricsDisabled.AccountAdapterDetails, "Accound adapter metrics should be disabled")
}

func TestRecordPrebidCacheRequestTimeWithSuccess(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AccountAdapterDetails: true})

	m.RecordPrebidCacheRequestTime(true, 42)

	assert.Equal(t, m.PrebidCacheRequestTimerSuccess.Count(), int64(1))
	assert.Equal(t, m.PrebidCacheRequestTimerError.Count(), int64(0))
}

func TestRecordPrebidCacheRequestTimeWithNotSuccess(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AccountAdapterDetails: true})

	m.RecordPrebidCacheRequestTime(false, 42)

	assert.Equal(t, m.PrebidCacheRequestTimerSuccess.Count(), int64(0))
	assert.Equal(t, m.PrebidCacheRequestTimerError.Count(), int64(1))
}

func ensureContainsBidTypeMetrics(t *testing.T, registry metrics.Registry, prefix string, mdm map[openrtb_ext.BidType]*MarkupDeliveryMetrics) {
	ensureContains(t, registry, prefix+".banner.adm_bids_received", mdm[openrtb_ext.BidTypeBanner].AdmMeter)
	ensureContains(t, registry, prefix+".banner.nurl_bids_received", mdm[openrtb_ext.BidTypeBanner].NurlMeter)
	ensureContains(t, registry, prefix+".video.adm_bids_received", mdm[openrtb_ext.BidTypeVideo].AdmMeter)
	ensureContains(t, registry, prefix+".video.nurl_bids_received", mdm[openrtb_ext.BidTypeVideo].NurlMeter)
	ensureContains(t, registry, prefix+".audio.adm_bids_received", mdm[openrtb_ext.BidTypeAudio].AdmMeter)
	ensureContains(t, registry, prefix+".audio.nurl_bids_received", mdm[openrtb_ext.BidTypeAudio].NurlMeter)
	ensureContains(t, registry, prefix+".native.adm_bids_received", mdm[openrtb_ext.BidTypeNative].AdmMeter)
	ensureContains(t, registry, prefix+".native.nurl_bids_received", mdm[openrtb_ext.BidTypeNative].NurlMeter)
}

func VerifyMetrics(t *testing.T, name string, expected int64, actual int64) {
	if expected != actual {
		t.Errorf("Error in metric %s: expected %d, got %d.", name, expected, actual)
	}
}
