package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	gdprPrivacy "github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/prebid/prebid-server/usersync"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewCookieSyncEndpoint(t *testing.T) {
	var (
		syncersByBidder   = map[string]usersync.Syncer{"a": &MockSyncer{}}
		gdprPerms         = MockGDPRPerms{}
		configUserSync    = config.UserSync{Cooperative: config.UserSyncCooperative{EnabledByDefault: true}}
		configHostCookie  = config.HostCookie{Family: "foo"}
		configGDPR        = config.GDPR{HostVendorID: 42}
		configCCPAEnforce = true
		metrics           = metrics.MetricsEngineMock{}
		analytics         = MockAnalytics{}
		fetcher           = FakeAccountsFetcher{}
		bidders           = map[string]openrtb_ext.BidderName{"bidderA": openrtb_ext.BidderName("bidderA"), "bidderB": openrtb_ext.BidderName("bidderB")}
	)

	endpoint := NewCookieSyncEndpoint(
		syncersByBidder,
		&config.Configuration{
			UserSync:   configUserSync,
			HostCookie: configHostCookie,
			GDPR:       configGDPR,
			CCPA:       config.CCPA{Enforce: configCCPAEnforce},
		},
		&gdprPerms,
		&metrics,
		&analytics,
		&fetcher,
		bidders,
	)

	expected := &cookieSyncEndpoint{
		chooser: usersync.NewChooser(syncersByBidder),
		config: &config.Configuration{
			UserSync:   configUserSync,
			HostCookie: configHostCookie,
			GDPR:       configGDPR,
			CCPA:       config.CCPA{Enforce: configCCPAEnforce},
		},
		privacyConfig: usersyncPrivacyConfig{
			gdprConfig:      configGDPR,
			gdprPermissions: &gdprPerms,
			ccpaEnforce:     configCCPAEnforce,
			bidderHashSet:   map[string]struct{}{"bidderA": {}, "bidderB": {}},
		},
		metrics:         &metrics,
		pbsAnalytics:    &analytics,
		accountsFetcher: &fetcher,
	}

	assert.Equal(t, expected, endpoint)
}

// usersyncPrivacy
func TestCookieSyncHandle(t *testing.T) {
	syncTypeExpected := []usersync.SyncType{usersync.SyncTypeIFrame, usersync.SyncTypeRedirect}
	sync := usersync.Sync{URL: "aURL", Type: usersync.SyncTypeRedirect, SupportCORS: true}
	syncer := MockSyncer{}
	syncer.On("GetSync", syncTypeExpected, privacy.Policies{}).Return(sync, nil).Maybe()

	cookieWithSyncs := usersync.NewCookie()
	cookieWithSyncs.TrySync("foo", "anyID")

	testCases := []struct {
		description              string
		givenCookie              *usersync.Cookie
		givenBody                io.Reader
		givenChooserResult       usersync.Result
		expectedStatusCode       int
		expectedBody             string
		setMetricsExpectations   func(*metrics.MetricsEngineMock)
		setAnalyticsExpectations func(*MockAnalytics)
	}{
		{
			description: "Request With Cookie",
			givenCookie: cookieWithSyncs,
			givenBody:   strings.NewReader(`{}`),
			givenChooserResult: usersync.Result{
				Status:           usersync.StatusOK,
				BiddersEvaluated: []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusOK}},
				SyncersChosen:    []usersync.SyncerChoice{{Bidder: "a", Syncer: &syncer}},
			},
			expectedStatusCode: 200,
			expectedBody: `{"status":"ok","bidder_status":[` +
				`{"bidder":"a","no_cookie":true,"usersync":{"url":"aURL","type":"redirect","supportCORS":true}}` +
				`]}` + "\n",
			setMetricsExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncOK).Once()
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncOK).Once()
			},
			setAnalyticsExpectations: func(a *MockAnalytics) {
				expected := analytics.CookieSyncObject{
					Status: 200,
					Errors: nil,
					BidderStatus: []*analytics.CookieSyncBidder{
						{
							BidderCode:   "a",
							NoCookie:     true,
							UsersyncInfo: &analytics.UsersyncInfo{URL: "aURL", Type: "redirect", SupportCORS: true},
						},
					},
				}
				a.On("LogCookieSyncObject", &expected).Once()
			},
		},
		{
			description: "Request Without Cookie",
			givenCookie: nil,
			givenBody:   strings.NewReader(`{}`),
			givenChooserResult: usersync.Result{
				Status:           usersync.StatusOK,
				BiddersEvaluated: []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusOK}},
				SyncersChosen:    []usersync.SyncerChoice{{Bidder: "a", Syncer: &syncer}},
			},
			expectedStatusCode: 200,
			expectedBody: `{"status":"no_cookie","bidder_status":[` +
				`{"bidder":"a","no_cookie":true,"usersync":{"url":"aURL","type":"redirect","supportCORS":true}}` +
				`]}` + "\n",
			setMetricsExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncOK).Once()
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncOK).Once()
			},
			setAnalyticsExpectations: func(a *MockAnalytics) {
				expected := analytics.CookieSyncObject{
					Status: 200,
					Errors: nil,
					BidderStatus: []*analytics.CookieSyncBidder{
						{
							BidderCode:   "a",
							NoCookie:     true,
							UsersyncInfo: &analytics.UsersyncInfo{URL: "aURL", Type: "redirect", SupportCORS: true},
						},
					},
				}
				a.On("LogCookieSyncObject", &expected).Once()
			},
		},
		{
			description: "Malformed Request",
			givenCookie: cookieWithSyncs,
			givenBody:   strings.NewReader(`malformed`),
			givenChooserResult: usersync.Result{
				Status:           usersync.StatusOK,
				BiddersEvaluated: []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusOK}},
				SyncersChosen:    []usersync.SyncerChoice{{Bidder: "a", Syncer: &syncer}},
			},
			expectedStatusCode: 400,
			expectedBody:       `JSON parsing failed: invalid character 'm' looking for beginning of value` + "\n",
			setMetricsExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncBadRequest).Once()
			},
			setAnalyticsExpectations: func(a *MockAnalytics) {
				expected := analytics.CookieSyncObject{
					Status:       400,
					Errors:       []error{errors.New("JSON parsing failed: invalid character 'm' looking for beginning of value")},
					BidderStatus: []*analytics.CookieSyncBidder{},
				}
				a.On("LogCookieSyncObject", &expected).Once()
			},
		},
		{
			description: "Request Blocked By Opt Out",
			givenCookie: cookieWithSyncs,
			givenBody:   strings.NewReader(`{}`),
			givenChooserResult: usersync.Result{
				Status:           usersync.StatusBlockedByUserOptOut,
				BiddersEvaluated: []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusOK}},
				SyncersChosen:    []usersync.SyncerChoice{{Bidder: "a", Syncer: &syncer}},
			},
			expectedStatusCode: 401,
			expectedBody:       `User has opted out` + "\n",
			setMetricsExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncOptOut).Once()
			},
			setAnalyticsExpectations: func(a *MockAnalytics) {
				expected := analytics.CookieSyncObject{
					Status:       401,
					Errors:       []error{errors.New("User has opted out")},
					BidderStatus: []*analytics.CookieSyncBidder{},
				}
				a.On("LogCookieSyncObject", &expected).Once()
			},
		},
		{
			description: "Request Blocked By GDPR Host Cookie Restriction",
			givenCookie: cookieWithSyncs,
			givenBody:   strings.NewReader(`{}`),
			givenChooserResult: usersync.Result{
				Status:           usersync.StatusBlockedByGDPR,
				BiddersEvaluated: []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusOK}},
				SyncersChosen:    []usersync.SyncerChoice{{Bidder: "a", Syncer: &syncer}},
			},
			expectedStatusCode: 200,
			expectedBody:       `{"status":"ok","bidder_status":[]}` + "\n",
			setMetricsExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncGDPRHostCookieBlocked).Once()
			},
			setAnalyticsExpectations: func(a *MockAnalytics) {
				expected := analytics.CookieSyncObject{
					Status:       200,
					Errors:       nil,
					BidderStatus: []*analytics.CookieSyncBidder{},
				}
				a.On("LogCookieSyncObject", &expected).Once()
			},
		},
	}

	for _, test := range testCases {
		mockMetrics := metrics.MetricsEngineMock{}
		test.setMetricsExpectations(&mockMetrics)

		mockAnalytics := MockAnalytics{}
		test.setAnalyticsExpectations(&mockAnalytics)

		fakeAccountFetcher := FakeAccountsFetcher{}

		request := httptest.NewRequest("POST", "/cookiesync", test.givenBody)
		if test.givenCookie != nil {
			request.AddCookie(test.givenCookie.ToHTTPCookie(24 * time.Hour))
		}

		writer := httptest.NewRecorder()

		endpoint := cookieSyncEndpoint{
			chooser: FakeChooser{Result: test.givenChooserResult},
			config:  &config.Configuration{},
			privacyConfig: usersyncPrivacyConfig{
				gdprConfig: config.GDPR{
					Enabled:      true,
					DefaultValue: "0",
				},
				ccpaEnforce: true,
			},
			metrics:         &mockMetrics,
			pbsAnalytics:    &mockAnalytics,
			accountsFetcher: &fakeAccountFetcher,
		}
		endpoint.Handle(writer, request, nil)

		assert.Equal(t, test.expectedStatusCode, writer.Code, test.description+":status_code")
		assert.Equal(t, test.expectedBody, writer.Body.String(), test.description+":body")
		mockMetrics.AssertExpectations(t)
		mockAnalytics.AssertExpectations(t)
	}
}

func TestCookieSyncParseRequest(t *testing.T) {
	expectedCCPAParsedPolicy, _ := ccpa.Policy{Consent: "1NYN"}.Parse(map[string]struct{}{})

	testCases := []struct {
		description          string
		givenConfig          config.UserSync
		givenBody            io.Reader
		givenGDPRConfig      config.GDPR
		givenCCPAEnabled     bool
		givenAccountRequired bool
		expectedError        string
		expectedPrivacy      privacy.Policies
		expectedRequest      usersync.Request
	}{
		{
			description: "Complete Request",
			givenBody: strings.NewReader(`{` +
				`"bidders":["a", "b"],` +
				`"gdpr":1,` +
				`"gdpr_consent":"anyGDPRConsent",` +
				`"us_privacy":"1NYN",` +
				`"limit":42,` +
				`"coopSync":true,` +
				`"filterSettings":{"iframe":{"bidders":"*","filter":"include"}, "image":{"bidders":["b"],"filter":"exclude"}}` +
				`}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
					PriorityGroups:   [][]string{{"a", "b", "c"}},
				},
			},
			expectedPrivacy: privacy.Policies{
				GDPR: gdprPrivacy.Policy{
					Signal:  "1",
					Consent: "anyGDPRConsent",
				},
				CCPA: ccpa.Policy{
					Consent: "1NYN",
				},
			},
			expectedRequest: usersync.Request{
				Bidders: []string{"a", "b"},
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: 42,
				Privacy: usersyncPrivacy{
					gdprSignal:       gdpr.SignalYes,
					gdprConsent:      "anyGDPRConsent",
					ccpaParsedPolicy: expectedCCPAParsedPolicy,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewSpecificBidderFilter([]string{"b"}, usersync.BidderFilterModeExclude),
				},
			},
		},
		{
			description: "Complete Request - Legacy Fields Only",
			givenBody: strings.NewReader(`{` +
				`"bidders":["a", "b"],` +
				`"gdpr":1,` +
				`"gdpr_consent":"anyGDPRConsent",` +
				`"us_privacy":"1NYN",` +
				`"limit":42` +
				`}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
					PriorityGroups:   [][]string{{"a", "b", "c"}},
				},
			},
			expectedPrivacy: privacy.Policies{
				GDPR: gdprPrivacy.Policy{
					Signal:  "1",
					Consent: "anyGDPRConsent",
				},
				CCPA: ccpa.Policy{
					Consent: "1NYN",
				},
			},
			expectedRequest: usersync.Request{
				Bidders: []string{"a", "b"},
				Cooperative: usersync.Cooperative{
					Enabled:        false,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: 42,
				Privacy: usersyncPrivacy{
					gdprSignal:       gdpr.SignalYes,
					gdprConsent:      "anyGDPRConsent",
					ccpaParsedPolicy: expectedCCPAParsedPolicy,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "Empty Request",
			givenBody:        strings.NewReader(`{}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			expectedPrivacy:  privacy.Policies{},
			expectedRequest: usersync.Request{
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "Cooperative Unspecified - Default True",
			givenBody:        strings.NewReader(`{}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: true,
					PriorityGroups:   [][]string{{"a", "b", "c"}},
				},
			},
			expectedPrivacy: privacy.Policies{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "Cooperative Unspecified - Default False",
			givenBody:        strings.NewReader(`{}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
					PriorityGroups:   [][]string{{"a", "b", "c"}},
				},
			},
			expectedPrivacy: privacy.Policies{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        false,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "Cooperative False - Default True",
			givenBody:        strings.NewReader(`{"coopSync":false}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: true,
					PriorityGroups:   [][]string{{"a", "b", "c"}},
				},
			},
			expectedPrivacy: privacy.Policies{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        false,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "Cooperative False - Default False",
			givenBody:        strings.NewReader(`{"coopSync":false}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
					PriorityGroups:   [][]string{{"a", "b", "c"}},
				},
			},
			expectedPrivacy: privacy.Policies{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        false,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "Cooperative True - Default True",
			givenBody:        strings.NewReader(`{"coopSync":true}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: true,
					PriorityGroups:   [][]string{{"a", "b", "c"}},
				},
			},
			expectedPrivacy: privacy.Policies{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "Cooperative True - Default False",
			givenBody:        strings.NewReader(`{"coopSync":true}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
					PriorityGroups:   [][]string{{"a", "b", "c"}},
				},
			},
			expectedPrivacy: privacy.Policies{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "CCPA Consent Invalid",
			givenBody:        strings.NewReader(`{"us_privacy":"invalid"}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			expectedPrivacy:  privacy.Policies{},
			expectedRequest: usersync.Request{
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "CCPA Disabled",
			givenBody:        strings.NewReader(`{"us_privacy":"1NYN"}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: false,
			expectedPrivacy: privacy.Policies{
				CCPA: ccpa.Policy{
					Consent: "1NYN"},
			},
			expectedRequest: usersync.Request{
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "Invalid JSON",
			givenBody:        strings.NewReader(`malformed`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			expectedError:    "JSON parsing failed: invalid character 'm' looking for beginning of value",
		},
		{
			description:      "Invalid Type Filter",
			givenBody:        strings.NewReader(`{"filterSettings":{"iframe":{"bidders":"invalid","filter":"exclude"}}}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			expectedError:    "error parsing filtersettings.iframe: invalid bidders value `invalid`. must either be '*' or a string array",
		},
		{
			description:      "Invalid GDPR Signal",
			givenBody:        strings.NewReader(`{"gdpr":5}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			expectedError:    "GDPR signal should be integer 0 or 1",
		},
		{
			description:      "Missing GDPR Consent - Explicit Signal 0",
			givenBody:        strings.NewReader(`{"gdpr":0}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			expectedPrivacy: privacy.Policies{
				GDPR: gdprPrivacy.Policy{Signal: "0"},
			},
			expectedRequest: usersync.Request{
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalNo,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "Missing GDPR Consent - Explicit Signal 1",
			givenBody:        strings.NewReader(`{"gdpr":1}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			expectedError:    "gdpr_consent is required if gdpr=1",
		},
		{
			description:      "Missing GDPR Consent - Ambiguous Signal - Default Value 0",
			givenBody:        strings.NewReader(`{}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			expectedPrivacy: privacy.Policies{
				GDPR: gdprPrivacy.Policy{Signal: ""},
			},
			expectedRequest: usersync.Request{
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description:      "Missing GDPR Consent - Ambiguous Signal - Default Value 1",
			givenBody:        strings.NewReader(`{}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "1"},
			givenCCPAEnabled: true,
			expectedError:    "gdpr_consent is required. gdpr is not specified and is assumed to be 1 by the server. set gdpr=0 to exempt this request",
		},
		{
			description:      "HTTP Read Error",
			givenBody:        ErrReader(errors.New("anyError")),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			expectedError:    "Failed to read request body",
		},
		{
			description: "Account Defaults - Max Limit + Default Coop",
			givenBody: strings.NewReader(`{` +
				`"bidders":["a", "b"],` +
				`"limit":42,` +
				`"account":"TestAccount"` +
				`}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				Cooperative: config.UserSyncCooperative{
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
			},
			expectedPrivacy: privacy.Policies{},
			expectedRequest: usersync.Request{
				Bidders: []string{"a", "b"},
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: 30,
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description: "Account Defaults - DefaultLimit",
			givenBody: strings.NewReader(`{` +
				`"bidders":["a", "b"],` +
				`"account":"TestAccount"` +
				`}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
					PriorityGroups:   [][]string{{"a", "b", "c"}},
				},
			},
			expectedPrivacy: privacy.Policies{},
			expectedRequest: usersync.Request{
				Bidders: []string{"a", "b"},
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: 20,
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
		},
		{
			description: "Account Defaults - Error",
			givenBody: strings.NewReader(`{` +
				`"bidders":["a", "b"],` +
				`"account":"DisabledAccount"` +
				`}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
					PriorityGroups:   [][]string{{"a", "b", "c"}},
				},
			},
			expectedPrivacy: privacy.Policies{},
			expectedRequest: usersync.Request{
				Bidders: []string{"a", "b"},
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: 20,
				Privacy: usersyncPrivacy{
					gdprSignal: gdpr.SignalAmbiguous,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				},
			},
			expectedError:        errCookieSyncAccountBlocked.Error(),
			givenAccountRequired: true,
		},
	}

	for _, test := range testCases {
		httpRequest := httptest.NewRequest("POST", "/cookiesync", test.givenBody)

		endpoint := cookieSyncEndpoint{
			config: &config.Configuration{
				UserSync:        test.givenConfig,
				AccountRequired: test.givenAccountRequired,
			},
			privacyConfig: usersyncPrivacyConfig{
				gdprConfig:  test.givenGDPRConfig,
				ccpaEnforce: test.givenCCPAEnabled,
			},
			accountsFetcher: FakeAccountsFetcher{AccountData: map[string]json.RawMessage{
				"TestAccount":     json.RawMessage(`{"cookie_sync": {"default_limit": 20, "max_limit": 30, "default_coop_sync": true}}`),
				"DisabledAccount": json.RawMessage(`{"disabled":true}`),
			}},
		}
		assert.NoError(t, endpoint.config.MarshalAccountDefaults())
		request, privacyPolicies, err := endpoint.parseRequest(httpRequest)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
			assert.Equal(t, test.expectedRequest, request, test.description+":request")
			assert.Equal(t, test.expectedPrivacy, privacyPolicies, test.description+":privacy")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
			assert.Empty(t, request, test.description+":request")
			assert.Empty(t, privacyPolicies, test.description+":privacy")
		}
	}
}

func TestSetLimit(t *testing.T) {
	intNegative1 := -1
	int20 := 20
	int30 := 30
	int40 := 40

	testCases := []struct {
		description     string
		givenRequest    cookieSyncRequest
		givenAccount    *config.Account
		expectedRequest cookieSyncRequest
	}{
		{
			description: "Default Limit is Applied (request limit = 0)",
			givenRequest: cookieSyncRequest{
				Limit: 0,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultLimit: &int20,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: 20,
			},
		},
		{
			description: "Default Limit is Not Applied (default limit not set)",
			givenRequest: cookieSyncRequest{
				Limit: 0,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultLimit: nil,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: 0,
			},
		},
		{
			description: "Default Limit is Not Applied (request limit > 0)",
			givenRequest: cookieSyncRequest{
				Limit: 10,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultLimit: &int20,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: 10,
			},
		},
		{
			description: "Max Limit is Applied (request limit <= 0)",
			givenRequest: cookieSyncRequest{
				Limit: 0,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					MaxLimit: &int30,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: 30,
			},
		},
		{
			description: "Max Limit is Applied (0 < max < limit)",
			givenRequest: cookieSyncRequest{
				Limit: 40,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					MaxLimit: &int30,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: 30,
			},
		},
		{
			description: "Max Limit is Not Applied (max not set)",
			givenRequest: cookieSyncRequest{
				Limit: 10,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					MaxLimit: nil,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: 10,
			},
		},
		{
			description: "Max Limit is Not Applied (0 < limit < max)",
			givenRequest: cookieSyncRequest{
				Limit: 10,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					MaxLimit: &int30,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: 10,
			},
		},
		{
			description: "Max Limit is Applied After applying the default",
			givenRequest: cookieSyncRequest{
				Limit: 0,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultLimit: &int40,
					MaxLimit:     &int30,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: 30,
			},
		},
		{
			description: "Negative Value Check",
			givenRequest: cookieSyncRequest{
				Limit: 0,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultLimit: &intNegative1,
					MaxLimit:     &intNegative1,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: 0,
			},
		},
	}

	for _, test := range testCases {
		endpoint := cookieSyncEndpoint{}
		request := endpoint.setLimit(test.givenRequest, test.givenAccount.CookieSync)
		assert.Equal(t, test.expectedRequest, request, test.description)
	}
}

func TestSetCooperativeSync(t *testing.T) {
	coopSyncFalse := false
	coopSyncTrue := true

	testCases := []struct {
		description     string
		givenRequest    cookieSyncRequest
		givenAccount    *config.Account
		expectedRequest cookieSyncRequest
	}{
		{
			description: "Request coop sync unmodified - request sync nil & default sync nil",
			givenRequest: cookieSyncRequest{
				CooperativeSync: nil,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultCoopSync: nil,
				},
			},
			expectedRequest: cookieSyncRequest{
				CooperativeSync: nil,
			},
		},
		{
			description: "Request coop sync set to default - request sync nil & default sync not nil",
			givenRequest: cookieSyncRequest{
				CooperativeSync: nil,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultCoopSync: &coopSyncTrue,
				},
			},
			expectedRequest: cookieSyncRequest{
				CooperativeSync: &coopSyncTrue,
			},
		},
		{
			description: "Request coop sync unmodified - request sync not nil & default sync nil",
			givenRequest: cookieSyncRequest{
				CooperativeSync: &coopSyncTrue,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultCoopSync: nil,
				},
			},
			expectedRequest: cookieSyncRequest{
				CooperativeSync: &coopSyncTrue,
			},
		},
		{
			description: "Request coop sync unmodified - request sync not nil & default sync not nil",
			givenRequest: cookieSyncRequest{
				CooperativeSync: &coopSyncFalse,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultCoopSync: &coopSyncTrue,
				},
			},
			expectedRequest: cookieSyncRequest{
				CooperativeSync: &coopSyncFalse,
			},
		},
	}

	for _, test := range testCases {
		endpoint := cookieSyncEndpoint{}
		request := endpoint.setCooperativeSync(test.givenRequest, test.givenAccount.CookieSync)
		assert.Equal(t, test.expectedRequest, request, test.description)
	}
}

func TestWriteParseRequestErrorMetrics(t *testing.T) {
	err := errors.New("anyError")

	mockAnalytics := MockAnalytics{}
	mockAnalytics.On("LogCookieSyncObject", mock.Anything)
	writer := httptest.NewRecorder()

	endpoint := cookieSyncEndpoint{pbsAnalytics: &mockAnalytics}
	endpoint.handleError(writer, err, 418)

	assert.Equal(t, writer.Code, 418)
	assert.Equal(t, writer.Body.String(), "anyError\n")
	mockAnalytics.AssertCalled(t, "LogCookieSyncObject", &analytics.CookieSyncObject{
		Status:       418,
		Errors:       []error{err},
		BidderStatus: []*analytics.CookieSyncBidder{},
	})
}

func TestCookieSyncWriteParseRequestErrorMetrics(t *testing.T) {
	testCases := []struct {
		description     string
		err             error
		setExpectations func(*metrics.MetricsEngineMock)
	}{
		{
			description: "Account Blocked",
			err:         errCookieSyncAccountBlocked,
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncAccountBlocked).Once()
			},
		},
		{
			description: "Account Invalid",
			err:         errCookieSyncAccountInvalid,
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncAccountInvalid).Once()
			},
		},
		{
			description: "No Special Case",
			err:         errors.New("any error"),
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncBadRequest).Once()
			},
		},
	}

	for _, test := range testCases {
		mockMetrics := metrics.MetricsEngineMock{}
		test.setExpectations(&mockMetrics)

		endpoint := &cookieSyncEndpoint{metrics: &mockMetrics}
		endpoint.writeParseRequestErrorMetrics(test.err)

		mockMetrics.AssertExpectations(t)
	}
}

func TestParseTypeFilter(t *testing.T) {
	testCases := []struct {
		description    string
		given          *cookieSyncRequestFilterSettings
		expectedError  string
		expectedFilter usersync.SyncTypeFilter
	}{
		{
			description: "Nil",
			given:       nil,
			expectedFilter: usersync.SyncTypeFilter{
				IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
			},
		},
		{
			description: "Nil Object",
			given:       &cookieSyncRequestFilterSettings{},
			expectedFilter: usersync.SyncTypeFilter{
				IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
			},
		},
		{
			description: "Given IFrame Only",
			given: &cookieSyncRequestFilterSettings{
				IFrame: &cookieSyncRequestFilter{Bidders: []interface{}{"a"}, Mode: "exclude"},
			},
			expectedFilter: usersync.SyncTypeFilter{
				IFrame:   usersync.NewSpecificBidderFilter([]string{"a"}, usersync.BidderFilterModeExclude),
				Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
			},
		},
		{
			description: "Given Redirect Only",
			given: &cookieSyncRequestFilterSettings{
				Redirect: &cookieSyncRequestFilter{Bidders: []interface{}{"b"}, Mode: "exclude"},
			},
			expectedFilter: usersync.SyncTypeFilter{
				IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
				Redirect: usersync.NewSpecificBidderFilter([]string{"b"}, usersync.BidderFilterModeExclude),
			},
		},
		{
			description: "Given Both",
			given: &cookieSyncRequestFilterSettings{
				IFrame:   &cookieSyncRequestFilter{Bidders: []interface{}{"a"}, Mode: "exclude"},
				Redirect: &cookieSyncRequestFilter{Bidders: []interface{}{"b"}, Mode: "exclude"},
			},
			expectedFilter: usersync.SyncTypeFilter{
				IFrame:   usersync.NewSpecificBidderFilter([]string{"a"}, usersync.BidderFilterModeExclude),
				Redirect: usersync.NewSpecificBidderFilter([]string{"b"}, usersync.BidderFilterModeExclude),
			},
		},
		{
			description: "IFrame Error",
			given: &cookieSyncRequestFilterSettings{
				IFrame:   &cookieSyncRequestFilter{Bidders: 42, Mode: "exclude"},
				Redirect: &cookieSyncRequestFilter{Bidders: []interface{}{"b"}, Mode: "exclude"},
			},
			expectedError: "error parsing filtersettings.iframe: invalid bidders type. must either be a string '*' or a string array of bidders",
		},
		{
			description: "Redirect Error",
			given: &cookieSyncRequestFilterSettings{
				IFrame:   &cookieSyncRequestFilter{Bidders: []interface{}{"a"}, Mode: "exclude"},
				Redirect: &cookieSyncRequestFilter{Bidders: 42, Mode: "exclude"},
			},
			expectedError: "error parsing filtersettings.image: invalid bidders type. must either be a string '*' or a string array of bidders",
		},
	}

	for _, test := range testCases {
		result, err := parseTypeFilter(test.given)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
			assert.Equal(t, test.expectedFilter, result, test.description+":result")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
			assert.Empty(t, result, test.description+":result")
		}
	}
}

func TestParseBidderFilter(t *testing.T) {
	testCases := []struct {
		description    string
		given          *cookieSyncRequestFilter
		expectedError  string
		expectedFilter usersync.BidderFilter
	}{
		{
			description:    "Nil",
			given:          nil,
			expectedFilter: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
		},
		{
			description:    "All Bidders - Include",
			given:          &cookieSyncRequestFilter{Bidders: "*", Mode: "include"},
			expectedFilter: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
		},
		{
			description:    "All Bidders - Exclude",
			given:          &cookieSyncRequestFilter{Bidders: "*", Mode: "exclude"},
			expectedFilter: usersync.NewUniformBidderFilter(usersync.BidderFilterModeExclude),
		},
		{
			description:   "All Bidders - Invalid Mode",
			given:         &cookieSyncRequestFilter{Bidders: "*", Mode: "invalid"},
			expectedError: "invalid filter value 'invalid'. must be either 'include' or 'exclude'",
		},
		{
			description:   "All Bidders - Unexpected Bidders Value",
			given:         &cookieSyncRequestFilter{Bidders: "invalid", Mode: "include"},
			expectedError: "invalid bidders value `invalid`. must either be '*' or a string array",
		},
		{
			description:    "Specific Bidders - Include",
			given:          &cookieSyncRequestFilter{Bidders: []interface{}{"a", "b"}, Mode: "include"},
			expectedFilter: usersync.NewSpecificBidderFilter([]string{"a", "b"}, usersync.BidderFilterModeInclude),
		},
		{
			description:    "Specific Bidders - Exclude",
			given:          &cookieSyncRequestFilter{Bidders: []interface{}{"a", "b"}, Mode: "exclude"},
			expectedFilter: usersync.NewSpecificBidderFilter([]string{"a", "b"}, usersync.BidderFilterModeExclude),
		},
		{
			description:   "Specific Bidders - Invalid Mode",
			given:         &cookieSyncRequestFilter{Bidders: []interface{}{"a", "b"}, Mode: "invalid"},
			expectedError: "invalid filter value 'invalid'. must be either 'include' or 'exclude'",
		},
		{
			description:   "Invalid Bidders Type",
			given:         &cookieSyncRequestFilter{Bidders: 42, Mode: "include"},
			expectedError: "invalid bidders type. must either be a string '*' or a string array of bidders",
		},
		{
			description:   "Invalid Bidders Type Of Array Element",
			given:         &cookieSyncRequestFilter{Bidders: []interface{}{"a", 42}, Mode: "include"},
			expectedError: "invalid bidders type. must either be a string '*' or a string array of bidders",
		},
	}

	for _, test := range testCases {
		result, err := parseBidderFilter(test.given)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
			assert.Equal(t, test.expectedFilter, result, test.description+":result")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
			assert.Nil(t, result, test.description+":result")
		}
	}
}

func TestCookieSyncHandleError(t *testing.T) {
	err := errors.New("anyError")

	mockAnalytics := MockAnalytics{}
	mockAnalytics.On("LogCookieSyncObject", mock.Anything)
	writer := httptest.NewRecorder()

	endpoint := cookieSyncEndpoint{pbsAnalytics: &mockAnalytics}
	endpoint.handleError(writer, err, 418)

	assert.Equal(t, writer.Code, 418)
	assert.Equal(t, writer.Body.String(), "anyError\n")
	mockAnalytics.AssertCalled(t, "LogCookieSyncObject", &analytics.CookieSyncObject{
		Status:       418,
		Errors:       []error{err},
		BidderStatus: []*analytics.CookieSyncBidder{},
	})
}

func TestCookieSyncWriteBidderMetrics(t *testing.T) {
	testCases := []struct {
		description     string
		given           []usersync.BidderEvaluation
		setExpectations func(*metrics.MetricsEngineMock)
	}{
		{
			description: "None",
			given:       []usersync.BidderEvaluation{},
			setExpectations: func(m *metrics.MetricsEngineMock) {
			},
		},
		{
			description: "One - OK",
			given:       []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusOK}},
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncOK).Once()
			},
		},
		{
			description: "One - Blocked By GDPR",
			given:       []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusBlockedByGDPR}},
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncPrivacyBlocked).Once()
			},
		},
		{
			description: "One - Blocked By CCPA",
			given:       []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusBlockedByCCPA}},
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncPrivacyBlocked).Once()
			},
		},
		{
			description: "One - Already Synced",
			given:       []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusAlreadySynced}},
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncAlreadySynced).Once()
			},
		},
		{
			description: "One - Type Not Supported",
			given:       []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusTypeNotSupported}},
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncTypeNotSupported).Once()
			},
		},
		{
			description: "Many",
			given: []usersync.BidderEvaluation{
				{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusOK},
				{Bidder: "b", SyncerKey: "bSyncer", Status: usersync.StatusAlreadySynced},
			},
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncOK).Once()
				m.On("RecordSyncerRequest", "bSyncer", metrics.SyncerCookieSyncAlreadySynced).Once()
			},
		},
	}

	for _, test := range testCases {
		mockMetrics := metrics.MetricsEngineMock{}
		test.setExpectations(&mockMetrics)

		endpoint := &cookieSyncEndpoint{metrics: &mockMetrics}
		endpoint.writeBidderMetrics(test.given)

		mockMetrics.AssertExpectations(t)
	}
}

func TestCookieSyncHandleResponse(t *testing.T) {
	syncTypeFilter := usersync.SyncTypeFilter{
		IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeExclude),
		Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
	}
	syncTypeExpected := []usersync.SyncType{usersync.SyncTypeRedirect}
	privacyPolicies := privacy.Policies{CCPA: ccpa.Policy{Consent: "anyConsent"}}

	// The & in the URL is necessary to test proper JSON encoding.
	syncA := usersync.Sync{URL: "https://syncA.com/sync?a=1&b=2", Type: usersync.SyncTypeRedirect, SupportCORS: true}
	syncerA := MockSyncer{}
	syncerA.On("GetSync", syncTypeExpected, privacyPolicies).Return(syncA, nil).Maybe()

	// The & in the URL is necessary to test proper JSON encoding.
	syncB := usersync.Sync{URL: "https://syncB.com/sync?a=1&b=2", Type: usersync.SyncTypeRedirect, SupportCORS: false}
	syncerB := MockSyncer{}
	syncerB.On("GetSync", syncTypeExpected, privacyPolicies).Return(syncB, nil).Maybe()

	syncWithError := usersync.Sync{}
	syncerWithError := MockSyncer{}
	syncerWithError.On("GetSync", syncTypeExpected, privacyPolicies).Return(syncWithError, errors.New("anyError")).Maybe()

	testCases := []struct {
		description         string
		givenCookieHasSyncs bool
		givenSyncersChosen  []usersync.SyncerChoice
		expectedJSON        string
		expectedAnalytics   analytics.CookieSyncObject
	}{
		{
			description:         "None",
			givenCookieHasSyncs: true,
			givenSyncersChosen:  []usersync.SyncerChoice{},
			expectedJSON:        `{"status":"ok","bidder_status":[]}` + "\n",
			expectedAnalytics:   analytics.CookieSyncObject{Status: 200, BidderStatus: []*analytics.CookieSyncBidder{}},
		},
		{
			description:         "One",
			givenCookieHasSyncs: true,
			givenSyncersChosen:  []usersync.SyncerChoice{{Bidder: "foo", Syncer: &syncerA}},
			expectedJSON: `{"status":"ok","bidder_status":[` +
				`{"bidder":"foo","no_cookie":true,"usersync":{"url":"https://syncA.com/sync?a=1&b=2","type":"redirect","supportCORS":true}}` +
				`]}` + "\n",
			expectedAnalytics: analytics.CookieSyncObject{
				Status: 200,
				BidderStatus: []*analytics.CookieSyncBidder{
					{
						BidderCode:   "foo",
						NoCookie:     true,
						UsersyncInfo: &analytics.UsersyncInfo{URL: "https://syncA.com/sync?a=1&b=2", Type: "redirect", SupportCORS: true},
					},
				},
			},
		},
		{
			description:         "Many",
			givenCookieHasSyncs: true,
			givenSyncersChosen:  []usersync.SyncerChoice{{Bidder: "foo", Syncer: &syncerA}, {Bidder: "bar", Syncer: &syncerB}},
			expectedJSON: `{"status":"ok","bidder_status":[` +
				`{"bidder":"foo","no_cookie":true,"usersync":{"url":"https://syncA.com/sync?a=1&b=2","type":"redirect","supportCORS":true}},` +
				`{"bidder":"bar","no_cookie":true,"usersync":{"url":"https://syncB.com/sync?a=1&b=2","type":"redirect"}}` +
				`]}` + "\n",
			expectedAnalytics: analytics.CookieSyncObject{
				Status: 200,
				BidderStatus: []*analytics.CookieSyncBidder{
					{
						BidderCode:   "foo",
						NoCookie:     true,
						UsersyncInfo: &analytics.UsersyncInfo{URL: "https://syncA.com/sync?a=1&b=2", Type: "redirect", SupportCORS: true},
					},
					{
						BidderCode:   "bar",
						NoCookie:     true,
						UsersyncInfo: &analytics.UsersyncInfo{URL: "https://syncB.com/sync?a=1&b=2", Type: "redirect", SupportCORS: false},
					},
				},
			},
		},
		{
			description:         "Many With One GetSync Error",
			givenCookieHasSyncs: true,
			givenSyncersChosen:  []usersync.SyncerChoice{{Bidder: "foo", Syncer: &syncerWithError}, {Bidder: "bar", Syncer: &syncerB}},
			expectedJSON: `{"status":"ok","bidder_status":[` +
				`{"bidder":"bar","no_cookie":true,"usersync":{"url":"https://syncB.com/sync?a=1&b=2","type":"redirect"}}` +
				`]}` + "\n",
			expectedAnalytics: analytics.CookieSyncObject{
				Status: 200,
				BidderStatus: []*analytics.CookieSyncBidder{
					{
						BidderCode:   "bar",
						NoCookie:     true,
						UsersyncInfo: &analytics.UsersyncInfo{URL: "https://syncB.com/sync?a=1&b=2", Type: "redirect", SupportCORS: false},
					},
				},
			},
		},
		{
			description:         "No Existing Syncs",
			givenCookieHasSyncs: false,
			givenSyncersChosen:  []usersync.SyncerChoice{},
			expectedJSON:        `{"status":"no_cookie","bidder_status":[]}` + "\n",
			expectedAnalytics:   analytics.CookieSyncObject{Status: 200, BidderStatus: []*analytics.CookieSyncBidder{}},
		},
	}

	for _, test := range testCases {
		mockAnalytics := MockAnalytics{}
		mockAnalytics.On("LogCookieSyncObject", &test.expectedAnalytics).Once()

		cookie := usersync.NewCookie()
		if test.givenCookieHasSyncs {
			if err := cookie.TrySync("foo", "anyID"); err != nil {
				assert.FailNow(t, test.description+":set_cookie")
			}
		}

		writer := httptest.NewRecorder()
		endpoint := cookieSyncEndpoint{pbsAnalytics: &mockAnalytics}
		endpoint.handleResponse(writer, syncTypeFilter, cookie, privacyPolicies, test.givenSyncersChosen)

		if assert.Equal(t, writer.Code, http.StatusOK, test.description+":http_status") {
			assert.Equal(t, writer.Header().Get("Content-Type"), "application/json; charset=utf-8", test.description+":http_header")
			assert.Equal(t, test.expectedJSON, writer.Body.String(), test.description+":http_response")
		}
		mockAnalytics.AssertExpectations(t)
	}
}

func TestMapBidderStatusToAnalytics(t *testing.T) {
	testCases := []struct {
		description string
		given       []cookieSyncResponseBidder
		expected    []*analytics.CookieSyncBidder
	}{
		{
			description: "None",
			given:       []cookieSyncResponseBidder{},
			expected:    []*analytics.CookieSyncBidder{},
		},
		{
			description: "One",
			given: []cookieSyncResponseBidder{
				{
					BidderCode:   "a",
					NoCookie:     true,
					UsersyncInfo: cookieSyncResponseSync{URL: "aURL", Type: "aType", SupportCORS: false},
				},
			},
			expected: []*analytics.CookieSyncBidder{
				{
					BidderCode:   "a",
					NoCookie:     true,
					UsersyncInfo: &analytics.UsersyncInfo{URL: "aURL", Type: "aType", SupportCORS: false},
				},
			},
		},
		{
			description: "Many",
			given: []cookieSyncResponseBidder{
				{
					BidderCode:   "a",
					NoCookie:     true,
					UsersyncInfo: cookieSyncResponseSync{URL: "aURL", Type: "aType", SupportCORS: false},
				},
				{
					BidderCode:   "b",
					NoCookie:     false,
					UsersyncInfo: cookieSyncResponseSync{URL: "bURL", Type: "bType", SupportCORS: true},
				},
			},
			expected: []*analytics.CookieSyncBidder{
				{
					BidderCode:   "a",
					NoCookie:     true,
					UsersyncInfo: &analytics.UsersyncInfo{URL: "aURL", Type: "aType", SupportCORS: false},
				},
				{
					BidderCode:   "b",
					NoCookie:     false,
					UsersyncInfo: &analytics.UsersyncInfo{URL: "bURL", Type: "bType", SupportCORS: true},
				},
			},
		},
	}

	for _, test := range testCases {
		result := mapBidderStatusToAnalytics(test.given)
		assert.ElementsMatch(t, test.expected, result, test.description)
	}
}

func TestUsersyncPrivacyGDPRAllowsHostCookie(t *testing.T) {
	testCases := []struct {
		description   string
		givenResponse bool
		givenError    error
		expected      bool
	}{
		{
			description:   "Allowed - No Error",
			givenResponse: true,
			givenError:    nil,
			expected:      true,
		},
		{
			description:   "Allowed - Error",
			givenResponse: true,
			givenError:    errors.New("anyError"),
			expected:      false,
		},
		{
			description:   "Not Allowed - No Error",
			givenResponse: false,
			givenError:    nil,
			expected:      false,
		},
		{
			description:   "Not Allowed - Error",
			givenResponse: false,
			givenError:    errors.New("anyError"),
			expected:      false,
		},
	}

	for _, test := range testCases {
		mockPerms := MockGDPRPerms{}
		mockPerms.On("HostCookiesAllowed", mock.Anything, gdpr.SignalYes, "anyConsent").Return(test.givenResponse, test.givenError)

		privacy := usersyncPrivacy{
			gdprPermissions: &mockPerms,
			gdprSignal:      gdpr.SignalYes,
			gdprConsent:     "anyConsent",
		}

		result := privacy.GDPRAllowsHostCookie()
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestUsersyncPrivacyGDPRAllowsBidderSync(t *testing.T) {
	testCases := []struct {
		description   string
		givenResponse bool
		givenError    error
		expected      bool
	}{
		{
			description:   "Allowed - No Error",
			givenResponse: true,
			givenError:    nil,
			expected:      true,
		},
		{
			description:   "Allowed - Error",
			givenResponse: true,
			givenError:    errors.New("anyError"),
			expected:      false,
		},
		{
			description:   "Not Allowed - No Error",
			givenResponse: false,
			givenError:    nil,
			expected:      false,
		},
		{
			description:   "Not Allowed - Error",
			givenResponse: false,
			givenError:    errors.New("anyError"),
			expected:      false,
		},
	}

	for _, test := range testCases {
		mockPerms := MockGDPRPerms{}
		mockPerms.On("BidderSyncAllowed", mock.Anything, openrtb_ext.BidderName("foo"), gdpr.SignalYes, "anyConsent").Return(test.givenResponse, test.givenError)

		privacy := usersyncPrivacy{
			gdprPermissions: &mockPerms,
			gdprSignal:      gdpr.SignalYes,
			gdprConsent:     "anyConsent",
		}

		result := privacy.GDPRAllowsBidderSync("foo")
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestUsersyncPrivacyCCPAAllowsBidderSync(t *testing.T) {
	testCases := []struct {
		description  string
		givenConsent string
		expected     bool
	}{
		{
			description:  "Allowed - No Opt-Out",
			givenConsent: "1NNN",
			expected:     true,
		},
		{
			description:  "Not Allowed - Opt-Out",
			givenConsent: "1NYN",
			expected:     false,
		},
		{
			description:  "Not Specified",
			givenConsent: "",
			expected:     true,
		},
	}

	for _, test := range testCases {
		validBidders := map[string]struct{}{"foo": {}}
		parsedPolicy, err := ccpa.Policy{Consent: test.givenConsent}.Parse(validBidders)

		if assert.NoError(t, err) {
			privacy := usersyncPrivacy{ccpaParsedPolicy: parsedPolicy}
			result := privacy.CCPAAllowsBidderSync("foo")
			assert.Equal(t, test.expected, result, test.description)
		}
	}
}

func TestCombineErrors(t *testing.T) {
	testCases := []struct {
		description    string
		givenErrorList []error
		expectedError  error
	}{
		{
			description:    "No errors given",
			givenErrorList: []error{},
			expectedError:  errors.New(""),
		},
		{
			description:    "One error given",
			givenErrorList: []error{errors.New("Error #1")},
			expectedError:  errors.New("Error #1"),
		},
		{
			description:    "Multiple errors given",
			givenErrorList: []error{errors.New("Error #1"), errors.New("Error #2")},
			expectedError:  errors.New("Error #1 Error #2"),
		},
		{
			description:    "Special Case: blocked (rejected via block list)",
			givenErrorList: []error{&errortypes.BlacklistedAcct{}},
			expectedError:  errCookieSyncAccountBlocked,
		},
		{
			description:    "Special Case: invalid (rejected via allow list)",
			givenErrorList: []error{&errortypes.AcctRequired{}},
			expectedError:  errCookieSyncAccountInvalid,
		},
		{
			description:    "Special Case: multiple special cases, first one wins",
			givenErrorList: []error{&errortypes.BlacklistedAcct{}, &errortypes.AcctRequired{}},
			expectedError:  errCookieSyncAccountBlocked,
		},
	}

	for _, test := range testCases {
		combinedErrors := combineErrors(test.givenErrorList)
		assert.Equal(t, test.expectedError, combinedErrors, test.description)
	}
}

type FakeChooser struct {
	Result usersync.Result
}

func (c FakeChooser) Choose(request usersync.Request, cookie *usersync.Cookie) usersync.Result {
	return c.Result
}

type MockSyncer struct {
	mock.Mock
}

func (m *MockSyncer) Key() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSyncer) DefaultSyncType() usersync.SyncType {
	args := m.Called()
	return args.Get(0).(usersync.SyncType)
}

func (m *MockSyncer) SupportsType(syncTypes []usersync.SyncType) bool {
	args := m.Called(syncTypes)
	return args.Bool(0)
}

func (m *MockSyncer) GetSync(syncTypes []usersync.SyncType, privacyPolicies privacy.Policies) (usersync.Sync, error) {
	args := m.Called(syncTypes, privacyPolicies)
	return args.Get(0).(usersync.Sync), args.Error(1)
}

type MockAnalytics struct {
	mock.Mock
}

func (m *MockAnalytics) LogAuctionObject(obj *analytics.AuctionObject) {
	m.Called(obj)
}

func (m *MockAnalytics) LogVideoObject(obj *analytics.VideoObject) {
	m.Called(obj)
}

func (m *MockAnalytics) LogCookieSyncObject(obj *analytics.CookieSyncObject) {
	m.Called(obj)
}

func (m *MockAnalytics) LogSetUIDObject(obj *analytics.SetUIDObject) {
	m.Called(obj)
}

func (m *MockAnalytics) LogAmpObject(obj *analytics.AmpObject) {
	m.Called(obj)
}

func (m *MockAnalytics) LogNotificationEventObject(obj *analytics.NotificationEvent) {
	m.Called(obj)
}

type MockGDPRPerms struct {
	mock.Mock
}

func (m *MockGDPRPerms) HostCookiesAllowed(ctx context.Context, gdprSignal gdpr.Signal, consent string) (bool, error) {
	args := m.Called(ctx, gdprSignal, consent)
	return args.Bool(0), args.Error(1)
}

func (m *MockGDPRPerms) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, gdprSignal gdpr.Signal, consent string) (bool, error) {
	args := m.Called(ctx, bidder, gdprSignal, consent)
	return args.Bool(0), args.Error(1)
}

func (m *MockGDPRPerms) AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName, PublisherID string, gdprSignal gdpr.Signal, consent string, aliasGVLIDs map[string]uint16) (permissions gdpr.AuctionPermissions, err error) {
	args := m.Called(ctx, bidderCoreName, bidder, PublisherID, gdprSignal, consent, aliasGVLIDs)
	return args.Get(0).(gdpr.AuctionPermissions), args.Error(1)
}

type FakeAccountsFetcher struct {
	AccountData map[string]json.RawMessage
}

func (f FakeAccountsFetcher) FetchAccount(ctx context.Context, accountID string) (json.RawMessage, []error) {
	if account, ok := f.AccountData[accountID]; ok {
		return account, nil
	}
	return nil, []error{errors.New("Account not found")}
}

// ErrReader returns an io.Reader that returns 0, err from all Read calls. This is added in
// Go 1.16. Copied here for now until we switch over.
func ErrReader(err error) io.Reader {
	return &errReader{err: err}
}

type errReader struct {
	err error
}

func (r *errReader) Read(p []byte) (int, error) {
	return 0, r.err
}
