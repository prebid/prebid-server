package metrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	registry := metrics.NewRegistry()
	syncerKeys := []string{"foo"}
	moduleStageNames := map[string][]string{"foobar": {"entry", "raw"}, "another_module": {"raw", "auction"}}
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{}, syncerKeys, moduleStageNames)

	ensureContains(t, registry, "app_requests", m.AppRequestMeter)
	ensureContains(t, registry, "debug_requests", m.DebugRequestMeter)
	ensureContains(t, registry, "no_cookie_requests", m.NoCookieMeter)
	ensureContains(t, registry, "request_time", m.RequestTimer)
	ensureContains(t, registry, "amp_no_cookie_requests", m.AmpNoCookieMeter)
	ensureContainsAdapterMetrics(t, registry, "adapter.appnexus", m.AdapterMetrics["appnexus"])
	ensureContainsAdapterMetrics(t, registry, "adapter.rubicon", m.AdapterMetrics["rubicon"])
	ensureContains(t, registry, "cookie_sync_requests", m.CookieSyncMeter)
	ensureContains(t, registry, "cookie_sync_requests.ok", m.CookieSyncStatusMeter[CookieSyncOK])
	ensureContains(t, registry, "cookie_sync_requests.bad_request", m.CookieSyncStatusMeter[CookieSyncBadRequest])
	ensureContains(t, registry, "cookie_sync_requests.opt_out", m.CookieSyncStatusMeter[CookieSyncOptOut])
	ensureContains(t, registry, "cookie_sync_requests.gdpr_blocked_host_cookie", m.CookieSyncStatusMeter[CookieSyncGDPRHostCookieBlocked])
	ensureContains(t, registry, "setuid_requests", m.SetUidMeter)
	ensureContains(t, registry, "setuid_requests.ok", m.SetUidStatusMeter[SetUidOK])
	ensureContains(t, registry, "setuid_requests.bad_request", m.SetUidStatusMeter[SetUidBadRequest])
	ensureContains(t, registry, "setuid_requests.opt_out", m.SetUidStatusMeter[SetUidOptOut])
	ensureContains(t, registry, "setuid_requests.gdpr_blocked_host_cookie", m.SetUidStatusMeter[SetUidGDPRHostCookieBlocked])
	ensureContains(t, registry, "setuid_requests.syncer_unknown", m.SetUidStatusMeter[SetUidSyncerUnknown])
	ensureContains(t, registry, "stored_responses", m.StoredResponsesMeter)

	ensureContains(t, registry, "prebid_cache_request_time.ok", m.PrebidCacheRequestTimerSuccess)
	ensureContains(t, registry, "prebid_cache_request_time.err", m.PrebidCacheRequestTimerError)

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

	ensureContains(t, registry, "syncer.foo.request.ok", m.SyncerRequestsMeter["foo"][SyncerCookieSyncOK])
	ensureContains(t, registry, "syncer.foo.request.privacy_blocked", m.SyncerRequestsMeter["foo"][SyncerCookieSyncPrivacyBlocked])
	ensureContains(t, registry, "syncer.foo.request.already_synced", m.SyncerRequestsMeter["foo"][SyncerCookieSyncAlreadySynced])
	ensureContains(t, registry, "syncer.foo.request.type_not_supported", m.SyncerRequestsMeter["foo"][SyncerCookieSyncTypeNotSupported])
	ensureContains(t, registry, "syncer.foo.set.ok", m.SyncerSetsMeter["foo"][SyncerSetUidOK])
	ensureContains(t, registry, "syncer.foo.set.cleared", m.SyncerSetsMeter["foo"][SyncerSetUidCleared])

	ensureContains(t, registry, "ads_cert_requests.ok", m.AdsCertRequestsSuccess)
	ensureContains(t, registry, "ads_cert_requests.failed", m.AdsCertRequestsFailure)

	ensureContains(t, registry, "request_over_head_time.pre-bidder", m.OverheadTimer[PreBidder])
	ensureContains(t, registry, "request_over_head_time.make-auction-response", m.OverheadTimer[MakeAuctionResponse])
	ensureContains(t, registry, "request_over_head_time.make-bidder-requests", m.OverheadTimer[MakeBidderRequests])
	ensureContains(t, registry, "bidder_server_response_time_seconds", m.BidderServerResponseTimer)

	for module, stages := range moduleStageNames {
		for _, stage := range stages {
			ensureContainsModuleMetrics(t, registry, fmt.Sprintf("modules.module.%s.stage.%s", module, stage), m.ModuleMetrics[module][stage])
		}
	}
}

func TestRecordBidType(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{}, nil, nil)

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

	ensureContains(t, registry, name+".response.validation.size.err", adapterMetrics.BidValidationCreativeSizeErrorMeter)
	ensureContains(t, registry, name+".response.validation.size.warn", adapterMetrics.BidValidationCreativeSizeWarnMeter)
	ensureContains(t, registry, name+".response.validation.secure.err", adapterMetrics.BidValidationSecureMarkupErrorMeter)
	ensureContains(t, registry, name+".response.validation.secure.warn", adapterMetrics.BidValidationSecureMarkupWarnMeter)

}

func ensureContainsModuleMetrics(t *testing.T, registry metrics.Registry, name string, moduleMetrics *ModuleMetrics) {
	t.Helper()
	ensureContains(t, registry, name+".duration", moduleMetrics.DurationTimer)
	ensureContains(t, registry, name+".call", moduleMetrics.CallCounter)
	ensureContains(t, registry, name+".failure", moduleMetrics.FailureCounter)
	ensureContains(t, registry, name+".success.noop", moduleMetrics.SuccessNoopCounter)
	ensureContains(t, registry, name+".success.update", moduleMetrics.SuccessUpdateCounter)
	ensureContains(t, registry, name+".success.reject", moduleMetrics.SuccessRejectCounter)
	ensureContains(t, registry, name+".execution_error", moduleMetrics.ExecutionErrorCounter)
	ensureContains(t, registry, name+".timeout", moduleMetrics.TimeoutCounter)
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
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, test.DisabledMetrics, nil, nil)

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

func TestRecordDebugRequest(t *testing.T) {
	testCases := []struct {
		description               string
		givenDisabledMetrics      config.DisabledMetrics
		givenDebugEnabledFlag     bool
		givenPubID                string
		expectedAccountDebugCount int64
		expectedDebugCount        int64
	}{
		{
			description: "Debug is enabled and account debug is enabled, both metrics should be updated",
			givenDisabledMetrics: config.DisabledMetrics{
				AccountAdapterDetails: true,
				AccountDebug:          false,
			},
			givenDebugEnabledFlag:     true,
			givenPubID:                "acct-id",
			expectedAccountDebugCount: 1,
			expectedDebugCount:        1,
		},
		{
			description: "Debug and account debug are disabled, niether metrics should be updated",
			givenDisabledMetrics: config.DisabledMetrics{
				AccountAdapterDetails: true,
				AccountDebug:          true,
			},
			givenDebugEnabledFlag:     false,
			givenPubID:                "acct-id",
			expectedAccountDebugCount: 0,
			expectedDebugCount:        0,
		},
		{
			description: "Debug is enabled and account debug is enabled, but unknown PubID leads to account debug being 0",
			givenDisabledMetrics: config.DisabledMetrics{
				AccountAdapterDetails: true,
				AccountDebug:          false,
			},
			givenDebugEnabledFlag:     true,
			givenPubID:                PublisherUnknown,
			expectedAccountDebugCount: 0,
			expectedDebugCount:        1,
		},
		{
			description: "Debug is disabled, account debug is enabled, niether metric should update",
			givenDisabledMetrics: config.DisabledMetrics{
				AccountAdapterDetails: true,
				AccountDebug:          false,
			},
			givenDebugEnabledFlag:     false,
			givenPubID:                "acct-id",
			expectedAccountDebugCount: 0,
			expectedDebugCount:        0,
		},
	}
	for _, test := range testCases {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, test.givenDisabledMetrics, nil, nil)

		m.RecordDebugRequest(test.givenDebugEnabledFlag, test.givenPubID)
		am := m.getAccountMetrics(test.givenPubID)

		assert.Equal(t, test.expectedDebugCount, m.DebugRequestMeter.Count())
		assert.Equal(t, test.expectedAccountDebugCount, am.debugRequestMeter.Count())
	}
}

func TestRecordBidValidationCreativeSize(t *testing.T) {
	testCases := []struct {
		description          string
		givenDisabledMetrics config.DisabledMetrics
		givenPubID           string
		expectedAccountCount int64
		expectedAdapterCount int64
	}{
		{
			description: "Account Metric isn't disabled, so both metrics should be incremented",
			givenDisabledMetrics: config.DisabledMetrics{
				AccountAdapterDetails: false,
			},
			givenPubID:           "acct-id",
			expectedAdapterCount: 1,
			expectedAccountCount: 1,
		},
		{
			description: "Account Metric is disabled, so only the adapter metric should increment",
			givenDisabledMetrics: config.DisabledMetrics{
				AccountAdapterDetails: true,
			},
			givenPubID:           "acct-id",
			expectedAdapterCount: 1,
			expectedAccountCount: 0,
		},
	}
	for _, test := range testCases {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, test.givenDisabledMetrics, nil, nil)

		m.RecordBidValidationCreativeSizeError(openrtb_ext.BidderAppnexus, test.givenPubID)
		m.RecordBidValidationCreativeSizeWarn(openrtb_ext.BidderAppnexus, test.givenPubID)
		am := m.getAccountMetrics(test.givenPubID)

		assert.Equal(t, test.expectedAdapterCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].BidValidationCreativeSizeErrorMeter.Count())
		assert.Equal(t, test.expectedAdapterCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].BidValidationCreativeSizeWarnMeter.Count())
		assert.Equal(t, test.expectedAccountCount, am.bidValidationCreativeSizeMeter.Count())
		assert.Equal(t, test.expectedAccountCount, am.bidValidationCreativeSizeWarnMeter.Count())
	}
}

func TestRecordBidValidationSecureMarkup(t *testing.T) {
	testCases := []struct {
		description          string
		givenDisabledMetrics config.DisabledMetrics
		givenPubID           string
		expectedAccountCount int64
		expectedAdapterCount int64
	}{
		{
			description: "Account Metric isn't disabled, so both metrics should be incremented",
			givenDisabledMetrics: config.DisabledMetrics{
				AccountAdapterDetails: false,
			},
			givenPubID:           "acct-id",
			expectedAdapterCount: 1,
			expectedAccountCount: 1,
		},
		{
			description: "Account Metric is disabled, so only the adapter metric should increment",
			givenDisabledMetrics: config.DisabledMetrics{
				AccountAdapterDetails: true,
			},
			givenPubID:           "acct-id",
			expectedAdapterCount: 1,
			expectedAccountCount: 0,
		},
	}
	for _, test := range testCases {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, test.givenDisabledMetrics, nil, nil)

		m.RecordBidValidationSecureMarkupError(openrtb_ext.BidderAppnexus, test.givenPubID)
		m.RecordBidValidationSecureMarkupWarn(openrtb_ext.BidderAppnexus, test.givenPubID)
		am := m.getAccountMetrics(test.givenPubID)

		assert.Equal(t, test.expectedAdapterCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].BidValidationSecureMarkupErrorMeter.Count())
		assert.Equal(t, test.expectedAdapterCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].BidValidationSecureMarkupWarnMeter.Count())
		assert.Equal(t, test.expectedAccountCount, am.bidValidationSecureMarkupMeter.Count())
		assert.Equal(t, test.expectedAccountCount, am.bidValidationSecureMarkupWarnMeter.Count())
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
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AccountAdapterDetails: true}, nil, nil)

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
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AccountAdapterDetails: true}, nil, nil)

		m.RecordTLSHandshakeTime(test.tLSHandshakeDuration)

		assert.Equal(t, test.expectedDuration.Nanoseconds(), m.TLSHandshakeTimer.Sum(), test.description)
	}
}

func TestRecordBidderServerResponseTime(t *testing.T) {
	testCases := []struct {
		name          string
		time          time.Duration
		expectedCount int64
		expectedSum   int64
	}{
		{
			name:          "record-bidder-server-response-time-1",
			time:          time.Duration(500),
			expectedCount: 1,
			expectedSum:   500,
		},
		{
			name:          "record-bidder-server-response-time-2",
			time:          time.Duration(500),
			expectedCount: 2,
			expectedSum:   1000,
		},
	}
	for _, test := range testCases {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AccountAdapterDetails: true}, nil, nil)

		m.RecordBidderServerResponseTime(test.time)

		assert.Equal(t, test.time.Nanoseconds(), m.BidderServerResponseTimer.Sum(), test.name)
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
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AdapterConnectionMetrics: test.in.connMetricsDisabled}, nil, nil)

		m.RecordAdapterConnections(test.in.adapterName, test.in.connWasReused, test.in.connWait)

		assert.Equal(t, test.out.expectedConnReusedCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].ConnReused.Count(), "Test [%d] incorrect number of reused connections to adapter", i)
		assert.Equal(t, test.out.expectedConnCreatedCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].ConnCreated.Count(), "Test [%d] incorrect number of new connections to adapter created", i)
		assert.Equal(t, test.out.expectedConnWaitTime.Nanoseconds(), m.AdapterMetrics[openrtb_ext.BidderAppnexus].ConnWaitTime.Sum(), "Test [%d] incorrect wait time in connection to adapter", i)
	}
}

func TestNewMetricsWithDisabledConfig(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{AccountAdapterDetails: true, AccountModulesMetrics: true}, nil, map[string][]string{"foobar": {"entry", "raw"}})

	assert.True(t, m.MetricsDisabled.AccountAdapterDetails, "Accound adapter metrics should be disabled")
	assert.True(t, m.MetricsDisabled.AccountModulesMetrics, "Accound modules metrics should be disabled")
}

func TestRecordPrebidCacheRequestTimeWithSuccess(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AccountAdapterDetails: true}, nil, nil)

	m.RecordPrebidCacheRequestTime(true, 42)

	assert.Equal(t, m.PrebidCacheRequestTimerSuccess.Count(), int64(1))
	assert.Equal(t, m.PrebidCacheRequestTimerError.Count(), int64(0))
}

func TestRecordPrebidCacheRequestTimeWithNotSuccess(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AccountAdapterDetails: true}, nil, nil)

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
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{AccountAdapterDetails: true}, nil, nil)
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
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{AccountAdapterDetails: true}, nil, nil)
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
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{AccountAdapterDetails: true}, nil, nil)

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
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AdapterGDPRRequestBlocked: tt.metricsDisabled}, nil, nil)

		m.RecordAdapterGDPRRequestBlocked(tt.adapterName)

		assert.Equal(t, tt.expectedCount, m.AdapterMetrics[openrtb_ext.BidderAppnexus].GDPRRequestBlocked.Count(), tt.description)
	}
}

func TestRecordCookieSync(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{}, nil, nil)

	// Known
	m.RecordCookieSync(CookieSyncBadRequest)

	// Unknown
	m.RecordCookieSync(CookieSyncStatus("unknown status"))

	assert.Equal(t, m.CookieSyncMeter.Count(), int64(2))
	assert.Equal(t, m.CookieSyncStatusMeter[CookieSyncOK].Count(), int64(0))
	assert.Equal(t, m.CookieSyncStatusMeter[CookieSyncBadRequest].Count(), int64(1))
	assert.Equal(t, m.CookieSyncStatusMeter[CookieSyncOptOut].Count(), int64(0))
	assert.Equal(t, m.CookieSyncStatusMeter[CookieSyncGDPRHostCookieBlocked].Count(), int64(0))
}

func TestRecordSyncerRequest(t *testing.T) {
	registry := metrics.NewRegistry()
	syncerKeys := []string{"foo"}
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{}, syncerKeys, nil)

	// Known
	m.RecordSyncerRequest("foo", SyncerCookieSyncOK)

	// Unknown Bidder
	m.RecordSyncerRequest("bar", SyncerCookieSyncOK)

	// Unknown Status
	m.RecordSyncerRequest("foo", SyncerCookieSyncStatus("unknown status"))

	assert.Equal(t, m.SyncerRequestsMeter["foo"][SyncerCookieSyncOK].Count(), int64(1))
	assert.Equal(t, m.SyncerRequestsMeter["foo"][SyncerCookieSyncPrivacyBlocked].Count(), int64(0))
	assert.Equal(t, m.SyncerRequestsMeter["foo"][SyncerCookieSyncAlreadySynced].Count(), int64(0))
	assert.Equal(t, m.SyncerRequestsMeter["foo"][SyncerCookieSyncTypeNotSupported].Count(), int64(0))
}

func TestRecordSetUid(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{}, nil, nil)

	// Known
	m.RecordSetUid(SetUidOptOut)

	// Unknown
	m.RecordSetUid(SetUidStatus("unknown status"))

	assert.Equal(t, m.SetUidMeter.Count(), int64(2))
	assert.Equal(t, m.SetUidStatusMeter[SetUidOK].Count(), int64(0))
	assert.Equal(t, m.SetUidStatusMeter[SetUidBadRequest].Count(), int64(0))
	assert.Equal(t, m.SetUidStatusMeter[SetUidOptOut].Count(), int64(1))
	assert.Equal(t, m.SetUidStatusMeter[SetUidGDPRHostCookieBlocked].Count(), int64(0))
	assert.Equal(t, m.SetUidStatusMeter[SetUidSyncerUnknown].Count(), int64(0))
}

func TestRecordSyncerSet(t *testing.T) {
	registry := metrics.NewRegistry()
	syncerKeys := []string{"foo"}
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon}, config.DisabledMetrics{}, syncerKeys, nil)

	// Known
	m.RecordSyncerSet("foo", SyncerSetUidCleared)

	// Unknown Bidder
	m.RecordSyncerSet("bar", SyncerSetUidCleared)

	// Unknown Status
	m.RecordSyncerSet("foo", SyncerSetUidStatus("unknown status"))

	assert.Equal(t, m.SyncerSetsMeter["foo"][SyncerSetUidOK].Count(), int64(0))
	assert.Equal(t, m.SyncerSetsMeter["foo"][SyncerSetUidCleared].Count(), int64(1))
}

func TestStoredResponses(t *testing.T) {
	testCases := []struct {
		description                           string
		givenPubID                            string
		accountStoredResponsesMetricsDisabled bool
		expectedAccountStoredResponsesCount   int64
		expectedStoredResponsesCount          int64
	}{
		{
			description:                           "Publisher id is given, account stored responses disabled, both metrics should be updated",
			givenPubID:                            "acct-id",
			accountStoredResponsesMetricsDisabled: true,
			expectedAccountStoredResponsesCount:   0,
			expectedStoredResponsesCount:          1,
		},
		{
			description:                           "Publisher id is given, account stored responses enabled, both metrics should be updated",
			givenPubID:                            "acct-id",
			accountStoredResponsesMetricsDisabled: false,
			expectedAccountStoredResponsesCount:   1,
			expectedStoredResponsesCount:          1,
		},
		{
			description:                           "Publisher id is unknown, account stored responses enabled, only expectedStoredResponsesCount metric should be updated",
			givenPubID:                            PublisherUnknown,
			accountStoredResponsesMetricsDisabled: false,
			expectedAccountStoredResponsesCount:   0,
			expectedStoredResponsesCount:          1,
		},
		{
			description:                           "Publisher id is unknown, account stored responses disabled, only expectedStoredResponsesCount metric should be updated",
			givenPubID:                            PublisherUnknown,
			accountStoredResponsesMetricsDisabled: true,
			expectedAccountStoredResponsesCount:   0,
			expectedStoredResponsesCount:          1,
		},
	}
	for _, test := range testCases {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{AccountStoredResponses: test.accountStoredResponsesMetricsDisabled}, nil, nil)

		m.RecordStoredResponse(test.givenPubID)
		am := m.getAccountMetrics(test.givenPubID)

		assert.Equal(t, test.expectedStoredResponsesCount, m.StoredResponsesMeter.Count())
		assert.Equal(t, test.expectedAccountStoredResponsesCount, am.storedResponsesMeter.Count())
	}
}

func TestRecordAdsCertSignTime(t *testing.T) {
	testCases := []struct {
		description           string
		inAdsCertSignDuration time.Duration
		outExpDuration        time.Duration
	}{
		{
			description:           "Five second AdsCertSign time",
			inAdsCertSignDuration: time.Second * 5,
			outExpDuration:        time.Second * 5,
		},
		{
			description:           "Five millisecond AdsCertSign time",
			inAdsCertSignDuration: time.Millisecond * 5,
			outExpDuration:        time.Millisecond * 5,
		},
		{
			description:           "Zero AdsCertSign time",
			inAdsCertSignDuration: time.Duration(0),
			outExpDuration:        time.Duration(0),
		},
	}
	for _, test := range testCases {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{}, nil, nil)

		m.RecordAdsCertSignTime(test.inAdsCertSignDuration)

		assert.Equal(t, test.outExpDuration.Nanoseconds(), m.adsCertSignTimer.Sum(), test.description)
	}
}

func TestRecordAdsCertReqMetric(t *testing.T) {
	testCases := []struct {
		description                  string
		requestSuccess               bool
		expectedSuccessRequestsCount int64
		expectedFailedRequestsCount  int64
	}{
		{
			description:                  "Record failed request, expected success request count is 0 and failed request count is 1",
			requestSuccess:               false,
			expectedSuccessRequestsCount: 0,
			expectedFailedRequestsCount:  1,
		},
		{
			description:                  "Record successful request, expected success request count is 1 and failed request count is 0",
			requestSuccess:               true,
			expectedSuccessRequestsCount: 1,
			expectedFailedRequestsCount:  0,
		},
	}

	for _, test := range testCases {
		registry := metrics.NewRegistry()
		m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{}, nil, nil)

		m.RecordAdsCertReq(test.requestSuccess)

		assert.Equal(t, test.expectedSuccessRequestsCount, m.AdsCertRequestsSuccess.Count(), test.description)
		assert.Equal(t, test.expectedFailedRequestsCount, m.AdsCertRequestsFailure.Count(), test.description)
	}
}

func TestRecordModuleAccountMetrics(t *testing.T) {
	registry := metrics.NewRegistry()
	module := "foobar"
	stage1 := "entrypoint"
	stage2 := "raw_auction"
	stage3 := "processed_auction"

	testCases := []struct {
		description                string
		givenModuleName            string
		givenStageName             string
		givenPubID                 string
		givenDisabledMetrics       config.DisabledMetrics
		expectedModuleMetricCount  int64
		expectedAccountMetricCount int64
	}{
		{
			description:                "Entrypoint stage should not record account metrics",
			givenModuleName:            module,
			givenStageName:             stage1,
			givenDisabledMetrics:       config.DisabledMetrics{AccountModulesMetrics: false},
			expectedModuleMetricCount:  1,
			expectedAccountMetricCount: 0,
		},
		{
			description:                "Rawauction stage should record both metrics",
			givenModuleName:            module,
			givenStageName:             stage2,
			givenPubID:                 "acc-1",
			givenDisabledMetrics:       config.DisabledMetrics{AccountModulesMetrics: false},
			expectedModuleMetricCount:  1,
			expectedAccountMetricCount: 1,
		},
		{
			description:                "Rawauction stage should not record account metrics because they are disabled",
			givenModuleName:            module,
			givenStageName:             stage3,
			givenPubID:                 "acc-1",
			givenDisabledMetrics:       config.DisabledMetrics{AccountModulesMetrics: true},
			expectedModuleMetricCount:  1,
			expectedAccountMetricCount: 0,
		},
	}
	for _, test := range testCases {
		m := NewMetrics(registry, nil, test.givenDisabledMetrics, nil, map[string][]string{module: {stage1, stage2, stage3}})

		m.RecordModuleCalled(ModuleLabels{
			Module:    test.givenModuleName,
			Stage:     test.givenStageName,
			AccountID: test.givenPubID,
		}, time.Microsecond)
		am := m.getAccountMetrics(test.givenPubID)

		assert.Equal(t, test.expectedModuleMetricCount, m.ModuleMetrics[test.givenModuleName][test.givenStageName].CallCounter.Count())
		if !test.givenDisabledMetrics.AccountModulesMetrics {
			assert.Equal(t, test.expectedAccountMetricCount, am.moduleMetrics[test.givenModuleName].CallCounter.Count())
			assert.Equal(t, test.expectedAccountMetricCount, am.moduleMetrics[test.givenModuleName].DurationTimer.Count())
		} else {
			assert.Len(t, am.moduleMetrics, 0, "Account modules metrics are disabled, they should not be collected. Actual result %d account metrics collected \n", len(am.moduleMetrics))
		}
	}
}

func TestRecordAccountGDPRPurposeWarningMetrics(t *testing.T) {
	testCases := []struct {
		name                   string
		givenPurposeName       string
		expectedP1MetricCount  int64
		expectedP2MetricCount  int64
		expectedP3MetricCount  int64
		expectedP4MetricCount  int64
		expectedP5MetricCount  int64
		expectedP6MetricCount  int64
		expectedP7MetricCount  int64
		expectedP8MetricCount  int64
		expectedP9MetricCount  int64
		expectedP10MetricCount int64
	}{
		{
			name:                  "Purpose1MetricIncremented",
			givenPurposeName:      "purpose1",
			expectedP1MetricCount: 1,
		},
		{
			name:                  "Purpose2MetricIncremented",
			givenPurposeName:      "purpose2",
			expectedP2MetricCount: 1,
		},
		{
			name:                  "Purpose3MetricIncremented",
			givenPurposeName:      "purpose3",
			expectedP3MetricCount: 1,
		},
		{
			name:                  "Purpose4MetricIncremented",
			givenPurposeName:      "purpose4",
			expectedP4MetricCount: 1,
		},
		{
			name:                  "Purpose5MetricIncremented",
			givenPurposeName:      "purpose5",
			expectedP5MetricCount: 1,
		},
		{
			name:                  "Purpose6MetricIncremented",
			givenPurposeName:      "purpose6",
			expectedP6MetricCount: 1,
		},
		{
			name:                  "Purpose7MetricIncremented",
			givenPurposeName:      "purpose7",
			expectedP7MetricCount: 1,
		},
		{
			name:                  "Purpose8MetricIncremented",
			givenPurposeName:      "purpose8",
			expectedP8MetricCount: 1,
		},
		{
			name:                  "Purpose9MetricIncremented",
			givenPurposeName:      "purpose9",
			expectedP9MetricCount: 1,
		},
		{
			name:                   "Purpose10MetricIncremented",
			givenPurposeName:       "purpose10",
			expectedP10MetricCount: 1,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			registry := metrics.NewRegistry()
			m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{}, nil, nil)

			m.RecordAccountGDPRPurposeWarning("acct-id", test.givenPurposeName)
			am := m.getAccountMetrics("acct-id")

			assert.Equal(t, test.expectedP1MetricCount, am.accountDeprecationWarningsPurpose1Meter.Count())
			assert.Equal(t, test.expectedP2MetricCount, am.accountDeprecationWarningsPurpose2Meter.Count())
			assert.Equal(t, test.expectedP3MetricCount, am.accountDeprecationWarningsPurpose3Meter.Count())
			assert.Equal(t, test.expectedP4MetricCount, am.accountDeprecationWarningsPurpose4Meter.Count())
			assert.Equal(t, test.expectedP5MetricCount, am.accountDeprecationWarningsPurpose5Meter.Count())
			assert.Equal(t, test.expectedP6MetricCount, am.accountDeprecationWarningsPurpose6Meter.Count())
			assert.Equal(t, test.expectedP7MetricCount, am.accountDeprecationWarningsPurpose7Meter.Count())
			assert.Equal(t, test.expectedP8MetricCount, am.accountDeprecationWarningsPurpose8Meter.Count())
			assert.Equal(t, test.expectedP9MetricCount, am.accountDeprecationWarningsPurpose9Meter.Count())
			assert.Equal(t, test.expectedP10MetricCount, am.accountDeprecationWarningsPurpose10Meter.Count())
		})
	}
}

func TestRecordAccountGDPRChannelEnabledWarningMetrics(t *testing.T) {
	testCases := []struct {
		name                string
		givenPubID          string
		expectedMetricCount int64
	}{
		{
			name:                "GdprChannelMetricIncremented",
			givenPubID:          "acct-id",
			expectedMetricCount: 1,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			registry := metrics.NewRegistry()
			m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{}, nil, nil)

			m.RecordAccountGDPRChannelEnabledWarning(test.givenPubID)
			am := m.getAccountMetrics(test.givenPubID)

			assert.Equal(t, test.expectedMetricCount, am.channelEnabledGDPRMeter.Count())
		})
	}
}

func TestRecordAccountCCPAChannelEnabledWarningMetrics(t *testing.T) {
	testCases := []struct {
		name                string
		givenPubID          string
		expectedMetricCount int64
	}{
		{
			name:                "CcpaChannelMetricIncremented",
			givenPubID:          "acct-id",
			expectedMetricCount: 1,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			registry := metrics.NewRegistry()
			m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{}, nil, nil)

			m.RecordAccountCCPAChannelEnabledWarning(test.givenPubID)
			am := m.getAccountMetrics(test.givenPubID)

			assert.Equal(t, test.expectedMetricCount, am.channelEnabledCCPAMeter.Count())
		})
	}
}

func TestRecordAccountUpgradeStatusMetrics(t *testing.T) {
	testCases := []struct {
		name                string
		givenPubID          string
		expectedMetricCount int64
	}{
		{
			name:                "AccountDeprecationMeterIncremented",
			givenPubID:          "acct-id",
			expectedMetricCount: 1,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			registry := metrics.NewRegistry()
			m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}, config.DisabledMetrics{}, nil, nil)

			m.RecordAccountUpgradeStatus(test.givenPubID)
			am := m.getAccountMetrics(test.givenPubID)

			assert.Equal(t, test.expectedMetricCount, am.accountDeprecationSummaryMeter.Count())
		})
	}
}

func TestRecordOverheadTime(t *testing.T) {
	testCases := []struct {
		name          string
		time          time.Duration
		overheadType  OverheadType
		expectedCount int64
		expectedSum   int64
	}{
		{
			name:          "record-pre-bidder-overhead-time-1",
			time:          time.Duration(500),
			overheadType:  PreBidder,
			expectedCount: 1,
			expectedSum:   500,
		},
		{
			name:          "record-pre-bidder-overhead-time-2",
			time:          time.Duration(500),
			overheadType:  PreBidder,
			expectedCount: 2,
			expectedSum:   1000,
		},
		{
			name:          "record-auction-response-overhead-time",
			time:          time.Duration(500),
			overheadType:  MakeAuctionResponse,
			expectedCount: 1,
			expectedSum:   500,
		},
		{
			name:          "record-make-bidder-requests-overhead-time",
			time:          time.Duration(500),
			overheadType:  MakeBidderRequests,
			expectedCount: 1,
			expectedSum:   500,
		},
	}
	registry := metrics.NewRegistry()
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			m := NewMetrics(registry, []openrtb_ext.BidderName{}, config.DisabledMetrics{}, nil, nil)
			m.RecordOverheadTime(test.overheadType, test.time)
			overheadMetrics := m.OverheadTimer[test.overheadType]
			assert.Equal(t, test.expectedCount, overheadMetrics.Count())
			assert.Equal(t, test.expectedSum, overheadMetrics.Sum())
		})
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
