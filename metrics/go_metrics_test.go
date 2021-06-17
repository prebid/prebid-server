package metrics

import (
	"testing"
	"time"

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

	ensureContains(t, registry, "queued_requests.video.rejected", m.RequestsQueueTimer[ReqTypeVideo][false])
	ensureContains(t, registry, "queued_requests.video.accepted", m.RequestsQueueTimer[ReqTypeVideo][true])

	ensureContains(t, registry, "timeout_notification.ok", m.TimeoutNotificationSuccess)
	ensureContains(t, registry, "timeout_notification.failed", m.TimeoutNotificationFailure)

	ensureContains(t, registry, "privacy.request.ccpa.specified", m.PrivacyCCPARequest)
	ensureContains(t, registry, "privacy.request.ccpa.opt-out", m.PrivacyCCPARequestOptOut)
	ensureContains(t, registry, "privacy.request.coppa", m.PrivacyCOPPARequest)
	ensureContains(t, registry, "privacy.request.lmt", m.PrivacyLMTRequest)
	ensureContains(t, registry, "privacy.request.tcf.v2", m.PrivacyTCFRequestVersion[TCFVersionV2])
	ensureContains(t, registry, "privacy.request.tcf.err", m.PrivacyTCFRequestVersion[TCFVersionErr])
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

	ensureContains(t, registry, name+".connections_created", adapterMetrics.ConnCreated)
	ensureContains(t, registry, name+".connections_reused", adapterMetrics.ConnReused)
	ensureContains(t, registry, name+".connection_wait_time", adapterMetrics.ConnWaitTime)
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

func TestRecordDNSTime(t *testing.T) {
	testCases := []struct {
		description         string
		inDnsLookupDuration time.Duration
		outExpDuration      time.Duration
	}{
		{
			description:         "Five second DNS lookup time",
			inDnsLookupDuration: time.Second * 5,
			outExpDuration:      time.Second * 5,
		},
		{
			description:         "Zero DNS lookup time",
			inDnsLookupDuration: time.Duration(0),
			outExpDuration:      time.Duration(0),
		},
	}
	for _, test := range testCases {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AccountAdapterDetails: true})

		m.RecordDNSTime(test.inDnsLookupDuration)

		assert.Equal(t, test.outExpDuration.Nanoseconds(), m.DNSLookupTimer.Sum(), test.description)
	}
}

func TestRecordTLSHandshakeTime(t *testing.T) {
	testCases := []struct {
		description          string
		tLSHandshakeDuration time.Duration
		expectedDuration     time.Duration
	}{
		{
			description:          "Five second TLS handshake time",
			tLSHandshakeDuration: time.Second * 5,
			expectedDuration:     time.Second * 5,
		},
		{
			description:          "Zero TLS handshake time",
			tLSHandshakeDuration: time.Duration(0),
			expectedDuration:     time.Duration(0),
		},
	}
	for _, test := range testCases {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AccountAdapterDetails: true})

		m.RecordTLSHandshakeTime(test.tLSHandshakeDuration)

		assert.Equal(t, test.expectedDuration.Nanoseconds(), m.TLSHandshakeTimer.Sum(), test.description)
	}
}

func TestRecordAdapterConnections(t *testing.T) {
	var fakeBidder openrtb_ext.BidderName = "fooAdvertising"

	type testIn struct {
		adapterName         openrtb_ext.BidderName
		connWasReused       bool
		connWait            time.Duration
		connMetricsDisabled bool
	}

	type testOut struct {
		expectedConnReusedCount  int64
		expectedConnCreatedCount int64
		expectedConnWaitTime     time.Duration
	}

	testCases := []struct {
		description string
		in          testIn
		out         testOut
	}{
		{
			description: "Successful, new connection created, has connection wait",
			in: testIn{
				adapterName:         openrtb_ext.BidderAppnexus,
				connWasReused:       false,
				connWait:            time.Second * 5,
				connMetricsDisabled: false,
			},
			out: testOut{
				expectedConnReusedCount:  0,
				expectedConnCreatedCount: 1,
				expectedConnWaitTime:     time.Second * 5,
			},
		},
		{
			description: "Successful, new connection created, has connection wait",
			in: testIn{
				adapterName:         openrtb_ext.BidderAppnexus,
				connWasReused:       false,
				connWait:            time.Second * 4,
				connMetricsDisabled: false,
			},
			out: testOut{
				expectedConnCreatedCount: 1,
				expectedConnWaitTime:     time.Second * 4,
			},
		},
		{
			description: "Successful, was reused, no connection wait",
			in: testIn{
				adapterName:         openrtb_ext.BidderAppnexus,
				connWasReused:       true,
				connMetricsDisabled: false,
			},
			out: testOut{
				expectedConnReusedCount: 1,
				expectedConnWaitTime:    0,
			},
		},
		{
			description: "Successful, was reused, has connection wait",
			in: testIn{
				adapterName:         openrtb_ext.BidderAppnexus,
				connWasReused:       true,
				connWait:            time.Second * 5,
				connMetricsDisabled: false,
			},
			out: testOut{
				expectedConnReusedCount: 1,
				expectedConnWaitTime:    time.Second * 5,
			},
		},
		{
			description: "Fake bidder, nothing gets updated",
			in: testIn{
				adapterName:         fakeBidder,
				connWasReused:       false,
				connWait:            0,
				connMetricsDisabled: false,
			},
			out: testOut{},
		},
		{
			description: "Adapter connection metrics are disabled, nothing gets updated",
			in: testIn{
				adapterName:         openrtb_ext.BidderAppnexus,
				connWasReused:       false,
				connWait:            time.Second * 5,
				connMetricsDisabled: true,
			},
			out: testOut{},
		},
	}

	for i, test := range testCases {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AdapterConnectionMetrics: test.in.connMetricsDisabled})

		m.RecordAdapterConnections(test.in.adapterName, test.in.connWasReused, test.in.connWait)

		assert.Equal(t, test.out.expectedConnReusedCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].ConnReused.Count(), "Test [%d] incorrect number of reused connections to adapter", i)
		assert.Equal(t, test.out.expectedConnCreatedCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].ConnCreated.Count(), "Test [%d] incorrect number of new connections to adapter created", i)
		assert.Equal(t, test.out.expectedConnWaitTime.Nanoseconds(), m.AdapterMetrics[openrtb_ext.BidderAppnexus].ConnWaitTime.Sum(), "Test [%d] incorrect wait time in connection to adapter", i)
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

func TestRecordStoredDataFetchTime(t *testing.T) {
	tests := []struct {
		description string
		dataType    StoredDataType
		fetchType   StoredDataFetchType
	}{
		{
			description: "Update stored_account_fetch_time.all timer",
			dataType:    AccountDataType,
			fetchType:   FetchAll,
		},
		{
			description: "Update stored_amp_fetch_time.all timer",
			dataType:    AMPDataType,
			fetchType:   FetchAll,
		},
		{
			description: "Update stored_category_fetch_time.all timer",
			dataType:    CategoryDataType,
			fetchType:   FetchAll,
		},
		{
			description: "Update stored_request_fetch_time.all timer",
			dataType:    RequestDataType,
			fetchType:   FetchAll,
		},
		{
			description: "Update stored_video_fetch_time.all timer",
			dataType:    VideoDataType,
			fetchType:   FetchAll,
		},
		{
			description: "Update stored_account_fetch_time.delta timer",
			dataType:    AccountDataType,
			fetchType:   FetchDelta,
		},
		{
			description: "Update stored_amp_fetch_time.delta timer",
			dataType:    AMPDataType,
			fetchType:   FetchDelta,
		},
		{
			description: "Update stored_category_fetch_time.delta timer",
			dataType:    CategoryDataType,
			fetchType:   FetchDelta,
		},
		{
			description: "Update stored_request_fetch_time.delta timer",
			dataType:    RequestDataType,
			fetchType:   FetchDelta,
		},
		{
			description: "Update stored_video_fetch_time.delta timer",
			dataType:    VideoDataType,
			fetchType:   FetchDelta,
		},
	}

	for _, tt := range tests {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{AccountAdapterDetails: true})
		m.RecordStoredDataFetchTime(StoredDataLabels{
			DataType:      tt.dataType,
			DataFetchType: tt.fetchType,
		}, time.Duration(500))

		actualCount := m.StoredDataFetchTimer[tt.dataType][tt.fetchType].Count()
		assert.Equal(t, int64(1), actualCount, tt.description)

		actualDuration := m.StoredDataFetchTimer[tt.dataType][tt.fetchType].Sum()
		assert.Equal(t, int64(500), actualDuration, tt.description)
	}
}

func TestRecordStoredDataError(t *testing.T) {
	tests := []struct {
		description string
		dataType    StoredDataType
		errorType   StoredDataError
	}{
		{
			description: "Increment stored_account_error.network meter",
			dataType:    AccountDataType,
			errorType:   StoredDataErrorNetwork,
		},
		{
			description: "Increment stored_amp_error.network meter",
			dataType:    AMPDataType,
			errorType:   StoredDataErrorNetwork,
		},
		{
			description: "Increment stored_category_error.network meter",
			dataType:    CategoryDataType,
			errorType:   StoredDataErrorNetwork,
		},
		{
			description: "Increment stored_request_error.network meter",
			dataType:    RequestDataType,
			errorType:   StoredDataErrorNetwork,
		},
		{
			description: "Increment stored_video_error.network meter",
			dataType:    VideoDataType,
			errorType:   StoredDataErrorNetwork,
		},
		{
			description: "Increment stored_account_error.undefined meter",
			dataType:    AccountDataType,
			errorType:   StoredDataErrorUndefined,
		},
		{
			description: "Increment stored_amp_error.undefined meter",
			dataType:    AMPDataType,
			errorType:   StoredDataErrorUndefined,
		},
		{
			description: "Increment stored_category_error.undefined meter",
			dataType:    CategoryDataType,
			errorType:   StoredDataErrorUndefined,
		},
		{
			description: "Increment stored_request_error.undefined meter",
			dataType:    RequestDataType,
			errorType:   StoredDataErrorUndefined,
		},
		{
			description: "Increment stored_video_error.undefined meter",
			dataType:    VideoDataType,
			errorType:   StoredDataErrorUndefined,
		},
	}

	for _, tt := range tests {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{AccountAdapterDetails: true})
		m.RecordStoredDataError(StoredDataLabels{
			DataType: tt.dataType,
			Error:    tt.errorType,
		})

		actualCount := m.StoredDataErrorMeter[tt.dataType][tt.errorType].Count()
		assert.Equal(t, int64(1), actualCount, tt.description)
	}
}

func TestRecordRequestPrivacy(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{AccountAdapterDetails: true})

	// CCPA
	m.RecordRequestPrivacy(PrivacyLabels{
		CCPAEnforced: true,
		CCPAProvided: true,
	})
	m.RecordRequestPrivacy(PrivacyLabels{
		CCPAEnforced: true,
		CCPAProvided: false,
	})
	m.RecordRequestPrivacy(PrivacyLabels{
		CCPAEnforced: false,
		CCPAProvided: true,
	})

	// COPPA
	m.RecordRequestPrivacy(PrivacyLabels{
		COPPAEnforced: true,
	})

	// LMT
	m.RecordRequestPrivacy(PrivacyLabels{
		LMTEnforced: true,
	})

	// GDPR
	m.RecordRequestPrivacy(PrivacyLabels{
		GDPREnforced:   true,
		GDPRTCFVersion: TCFVersionErr,
	})
	m.RecordRequestPrivacy(PrivacyLabels{
		GDPREnforced:   true,
		GDPRTCFVersion: TCFVersionV2,
	})

	assert.Equal(t, m.PrivacyCCPARequest.Count(), int64(2), "CCPA")
	assert.Equal(t, m.PrivacyCCPARequestOptOut.Count(), int64(1), "CCPA Opt Out")
	assert.Equal(t, m.PrivacyCOPPARequest.Count(), int64(1), "COPPA")
	assert.Equal(t, m.PrivacyLMTRequest.Count(), int64(1), "LMT")
	assert.Equal(t, m.PrivacyTCFRequestVersion[TCFVersionErr].Count(), int64(1), "TCF Err")
	assert.Equal(t, m.PrivacyTCFRequestVersion[TCFVersionV2].Count(), int64(1), "TCF V2")
}

func TestRecordAdapterGDPRRequestBlocked(t *testing.T) {
	var fakeBidder openrtb_ext.BidderName = "fooAdvertising"

	tests := []struct {
		description     string
		metricsDisabled bool
		adapterName     openrtb_ext.BidderName
		expectedCount   int64
	}{
		{
			description:     "",
			metricsDisabled: false,
			adapterName:     openrtb_ext.BidderAppnexus,
			expectedCount:   1,
		},
		{
			description:     "",
			metricsDisabled: false,
			adapterName:     fakeBidder,
			expectedCount:   0,
		},
		{
			description:     "",
			metricsDisabled: true,
			adapterName:     openrtb_ext.BidderAppnexus,
			expectedCount:   0,
		},
	}

	for _, tt := range tests {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AdapterGDPRRequestBlocked: tt.metricsDisabled})

		m.RecordAdapterGDPRRequestBlocked(tt.adapterName)

		assert.Equal(t, tt.expectedCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].GDPRRequestBlocked.Count(), tt.description)
	}
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
