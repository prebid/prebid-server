package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/privacy/ccpa"
	"github.com/prebid/prebid-server/v3/usersync"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// fakeTime implements the Time interface
type fakeTime struct {
	time time.Time
}

func (ft *fakeTime) Now() time.Time {
	return ft.time
}

func TestNewCookieSyncEndpoint(t *testing.T) {
	var (
		syncersByBidder  = map[string]usersync.Syncer{"a": &MockSyncer{}}
		gdprPermsBuilder = fakePermissionsBuilder{
			permissions: &fakePermissions{},
		}.Builder
		tcf2ConfigBuilder = fakeTCF2ConfigBuilder{
			cfg: gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}.Builder
		configUserSync    = config.UserSync{Cooperative: config.UserSyncCooperative{EnabledByDefault: true}}
		configHostCookie  = config.HostCookie{Family: "foo"}
		configGDPR        = config.GDPR{HostVendorID: 42}
		configCCPAEnforce = true
		metrics           = metrics.MetricsEngineMock{}
		analytics         = MockAnalyticsRunner{}
		fetcher           = FakeAccountsFetcher{}
		bidders           = map[string]openrtb_ext.BidderName{"bidderA": openrtb_ext.BidderName("bidderA"), "bidderB": openrtb_ext.BidderName("bidderB")}
		bidderInfo        = map[string]config.BidderInfo{"bidderA": {}, "bidderB": {}}
		biddersKnown      = map[string]struct{}{"bidderA": {}, "bidderB": {}}
	)

	endpoint := NewCookieSyncEndpoint(
		syncersByBidder,
		&config.Configuration{
			UserSync:    configUserSync,
			HostCookie:  configHostCookie,
			GDPR:        configGDPR,
			CCPA:        config.CCPA{Enforce: configCCPAEnforce},
			BidderInfos: bidderInfo,
		},
		gdprPermsBuilder,
		tcf2ConfigBuilder,
		&metrics,
		&analytics,
		&fetcher,
		bidders,
	)
	result := endpoint.(*cookieSyncEndpoint)

	expected := &cookieSyncEndpoint{
		chooser: usersync.NewChooser(syncersByBidder, biddersKnown, bidderInfo),
		config: &config.Configuration{
			UserSync:    configUserSync,
			HostCookie:  configHostCookie,
			GDPR:        configGDPR,
			CCPA:        config.CCPA{Enforce: configCCPAEnforce},
			BidderInfos: bidderInfo,
		},
		privacyConfig: usersyncPrivacyConfig{
			gdprConfig:             configGDPR,
			gdprPermissionsBuilder: gdprPermsBuilder,
			tcf2ConfigBuilder:      tcf2ConfigBuilder,
			ccpaEnforce:            configCCPAEnforce,
			bidderHashSet:          map[string]struct{}{"bidderA": {}, "bidderB": {}},
		},
		metrics:         &metrics,
		pbsAnalytics:    &analytics,
		accountsFetcher: &fetcher,
	}

	assert.IsType(t, &cookieSyncEndpoint{}, endpoint)

	assert.Equal(t, expected.config, result.config)
	assert.ObjectsAreEqualValues(expected.chooser, result.chooser)
	assert.Equal(t, expected.metrics, result.metrics)
	assert.Equal(t, expected.pbsAnalytics, result.pbsAnalytics)
	assert.Equal(t, expected.accountsFetcher, result.accountsFetcher)

	assert.Equal(t, expected.privacyConfig.gdprConfig, result.privacyConfig.gdprConfig)
	assert.Equal(t, expected.privacyConfig.ccpaEnforce, result.privacyConfig.ccpaEnforce)
	assert.Equal(t, expected.privacyConfig.bidderHashSet, result.privacyConfig.bidderHashSet)
}

func TestCookieSyncHandle(t *testing.T) {
	syncTypeExpected := []usersync.SyncType{usersync.SyncTypeIFrame, usersync.SyncTypeRedirect}
	sync := usersync.Sync{URL: "aURL", Type: usersync.SyncTypeRedirect, SupportCORS: true}
	syncer := MockSyncer{}
	syncer.On("GetSync", syncTypeExpected, macros.UserSyncPrivacy{}).Return(sync, nil).Maybe()

	cookieWithSyncs := usersync.NewCookie()
	cookieWithSyncs.Sync("foo", "anyID")

	testCases := []struct {
		description                     string
		givenCookie                     *usersync.Cookie
		givenBody                       io.Reader
		givenChooserResult              usersync.Result
		givenAccountData                map[string]json.RawMessage
		expectedStatusCode              int
		expectedBody                    string
		setMetricsExpectations          func(*metrics.MetricsEngineMock)
		setAnalyticsExpectations        func(*MockAnalyticsRunner)
		expectedCookieDeprecationHeader bool
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
			setAnalyticsExpectations: func(a *MockAnalyticsRunner) {
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
			setAnalyticsExpectations: func(a *MockAnalyticsRunner) {
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
			expectedBody:       `JSON parsing failed: expect { or n, but found m` + "\n",
			setMetricsExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncBadRequest).Once()
			},
			setAnalyticsExpectations: func(a *MockAnalyticsRunner) {
				expected := analytics.CookieSyncObject{
					Status:       400,
					Errors:       []error{errors.New("JSON parsing failed: expect { or n, but found m")},
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
			setAnalyticsExpectations: func(a *MockAnalyticsRunner) {
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
				Status:           usersync.StatusBlockedByPrivacy,
				BiddersEvaluated: []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusOK}},
				SyncersChosen:    []usersync.SyncerChoice{{Bidder: "a", Syncer: &syncer}},
			},
			expectedStatusCode: 200,
			expectedBody:       `{"status":"ok","bidder_status":[]}` + "\n",
			setMetricsExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncGDPRHostCookieBlocked).Once()
			},
			setAnalyticsExpectations: func(a *MockAnalyticsRunner) {
				expected := analytics.CookieSyncObject{
					Status:       200,
					Errors:       nil,
					BidderStatus: []*analytics.CookieSyncBidder{},
				}
				a.On("LogCookieSyncObject", &expected).Once()
			},
		},
		{
			description: "Debug Check",
			givenCookie: cookieWithSyncs,
			givenBody:   strings.NewReader(`{"debug": true}`),
			givenChooserResult: usersync.Result{
				Status:           usersync.StatusOK,
				BiddersEvaluated: []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusAlreadySynced}},
				SyncersChosen:    []usersync.SyncerChoice{{Bidder: "a", Syncer: &syncer}},
			},
			expectedStatusCode: 200,
			expectedBody: `{"status":"ok","bidder_status":[` +
				`{"bidder":"a","no_cookie":true,"usersync":{"url":"aURL","type":"redirect","supportCORS":true}}` +
				`],"debug":[{"bidder":"a","error":"Already in sync"}]}` + "\n",
			setMetricsExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncOK).Once()
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncAlreadySynced).Once()
			},
			setAnalyticsExpectations: func(a *MockAnalyticsRunner) {
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
			description: "CookieDeprecation-Set",
			givenCookie: cookieWithSyncs,
			givenBody:   strings.NewReader(`{"account": "testAccount"}`),
			givenChooserResult: usersync.Result{
				Status:           usersync.StatusOK,
				BiddersEvaluated: []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusAlreadySynced}},
				SyncersChosen:    []usersync.SyncerChoice{{Bidder: "a", Syncer: &syncer}},
			},
			givenAccountData: map[string]json.RawMessage{
				"testAccount": json.RawMessage(`{"id":"1","privacy":{"privacysandbox":{"cookiedeprecation":{"enabled":true,"ttlsec":86400}}}}`),
			},
			expectedStatusCode:              200,
			expectedCookieDeprecationHeader: true,
			expectedBody: `{"status":"ok","bidder_status":[` +
				`{"bidder":"a","no_cookie":true,"usersync":{"url":"aURL","type":"redirect","supportCORS":true}}` +
				`]}` + "\n",
			setMetricsExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncOK).Once()
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncAlreadySynced).Once()
			},
			setAnalyticsExpectations: func(a *MockAnalyticsRunner) {
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
	}

	for _, test := range testCases {
		mockMetrics := metrics.MetricsEngineMock{}
		test.setMetricsExpectations(&mockMetrics)

		mockAnalytics := MockAnalyticsRunner{}
		test.setAnalyticsExpectations(&mockAnalytics)

		fakeAccountFetcher := FakeAccountsFetcher{
			AccountData: test.givenAccountData,
		}

		gdprPermsBuilder := fakePermissionsBuilder{
			permissions: &fakePermissions{},
		}.Builder
		tcf2ConfigBuilder := fakeTCF2ConfigBuilder{
			cfg: gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}.Builder

		request := httptest.NewRequest("POST", "/cookiesync", test.givenBody)
		if test.givenCookie != nil {
			httpCookie, err := ToHTTPCookie(test.givenCookie)
			assert.NoError(t, err)
			request.AddCookie(httpCookie)
		}

		writer := httptest.NewRecorder()

		endpoint := cookieSyncEndpoint{
			chooser: FakeChooser{Result: test.givenChooserResult},
			config: &config.Configuration{
				AccountDefaults: config.Account{Disabled: false},
			},
			privacyConfig: usersyncPrivacyConfig{
				gdprConfig: config.GDPR{
					Enabled:      true,
					DefaultValue: "0",
				},
				gdprPermissionsBuilder: gdprPermsBuilder,
				tcf2ConfigBuilder:      tcf2ConfigBuilder,
				ccpaEnforce:            true,
			},
			metrics:         &mockMetrics,
			pbsAnalytics:    &mockAnalytics,
			accountsFetcher: &fakeAccountFetcher,
			time:            &fakeTime{time: time.Date(2024, 2, 22, 9, 42, 4, 13, time.UTC)},
		}
		assert.NoError(t, endpoint.config.MarshalAccountDefaults())

		endpoint.Handle(writer, request, nil)

		assert.Equal(t, test.expectedStatusCode, writer.Code, test.description+":status_code")
		assert.Equal(t, test.expectedBody, writer.Body.String(), test.description+":body")

		gotCookie := writer.Header().Get("Set-Cookie")
		if test.expectedCookieDeprecationHeader {
			wantCookieTTL := endpoint.time.Now().Add(time.Second * time.Duration(86400)).UTC().Format(http.TimeFormat)
			wantCookie := fmt.Sprintf("receive-cookie-deprecation=1; Path=/; Expires=%v; HttpOnly; Secure; SameSite=None; Partitioned;", wantCookieTTL)
			assert.Equal(t, wantCookie, gotCookie, test.description)
		} else {
			assert.Empty(t, gotCookie, test.description)
		}

		mockMetrics.AssertExpectations(t)
		mockAnalytics.AssertExpectations(t)
	}
}

func TestExtractGDPRSignal(t *testing.T) {
	type testInput struct {
		requestGDPR *int
		gppSID      []int8
	}
	type testOutput struct {
		gdprSignal gdpr.Signal
		gdprString string
		err        error
	}
	testCases := []struct {
		desc     string
		in       testInput
		expected testOutput
	}{
		{
			desc: "SectionTCFEU2 is listed in GPP_SID array, expect SignalYes and nil error",
			in: testInput{
				requestGDPR: nil,
				gppSID:      []int8{2},
			},
			expected: testOutput{
				gdprSignal: gdpr.SignalYes,
				gdprString: strconv.Itoa(int(gdpr.SignalYes)),
				err:        nil,
			},
		},
		{
			desc: "SectionTCFEU2 is not listed in GPP_SID array, expect SignalNo and nil error",
			in: testInput{
				requestGDPR: nil,
				gppSID:      []int8{6},
			},
			expected: testOutput{
				gdprSignal: gdpr.SignalNo,
				gdprString: strconv.Itoa(int(gdpr.SignalNo)),
				err:        nil,
			},
		},
		{
			desc: "Empty GPP_SID array and nil requestGDPR value, expect SignalAmbiguous and nil error",
			in: testInput{
				requestGDPR: nil,
				gppSID:      []int8{},
			},
			expected: testOutput{
				gdprSignal: gdpr.SignalAmbiguous,
				gdprString: "",
				err:        nil,
			},
		},
		{
			desc: "Empty GPP_SID array and non-nil requestGDPR value that could not be successfully parsed, expect SignalAmbiguous and parse error",
			in: testInput{
				requestGDPR: ptrutil.ToPtr(2),
				gppSID:      nil,
			},
			expected: testOutput{
				gdprSignal: gdpr.SignalAmbiguous,
				gdprString: "2",
				err:        &errortypes.BadInput{Message: "GDPR signal should be integer 0 or 1"},
			},
		},
		{
			desc: "Empty GPP_SID array and non-nil requestGDPR value that could be successfully parsed, expect SignalYes and nil error",
			in: testInput{
				requestGDPR: ptrutil.ToPtr(1),
				gppSID:      nil,
			},
			expected: testOutput{
				gdprSignal: gdpr.SignalYes,
				gdprString: "1",
				err:        nil,
			},
		},
	}
	for _, tc := range testCases {
		// run
		outSignal, outGdprStr, outErr := extractGDPRSignal(tc.in.requestGDPR, tc.in.gppSID)
		// assertions
		assert.Equal(t, tc.expected.gdprSignal, outSignal, tc.desc)
		assert.Equal(t, tc.expected.gdprString, outGdprStr, tc.desc)
		assert.Equal(t, tc.expected.err, outErr, tc.desc)
	}
}

func TestExtractPrivacyPolicies(t *testing.T) {
	type testInput struct {
		request                  cookieSyncRequest
		usersyncDefaultGDPRValue string
	}
	type testOutput struct {
		macros     macros.UserSyncPrivacy
		gdprSignal gdpr.Signal
		policies   privacy.Policies
		err        error
	}
	testCases := []struct {
		desc     string
		in       testInput
		expected testOutput
	}{
		{
			desc: "request GPP string is malformed, expect empty policies, signal No and error",
			in: testInput{
				request: cookieSyncRequest{GPP: "malformedGPPString"},
			},
			expected: testOutput{
				macros:     macros.UserSyncPrivacy{},
				gdprSignal: gdpr.SignalNo,
				policies:   privacy.Policies{},
				err:        errors.New("error parsing GPP header, header must have type=3"),
			},
		},
		{
			desc: "Malformed GPPSid string",
			in: testInput{
				request: cookieSyncRequest{
					GPP:       "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN",
					GPPSID:    "malformed",
					USPrivacy: "1YYY",
				},
			},
			expected: testOutput{
				macros:     macros.UserSyncPrivacy{},
				gdprSignal: gdpr.SignalNo,
				policies:   privacy.Policies{},
				err:        &strconv.NumError{Func: "ParseInt", Num: "malformed", Err: strconv.ErrSyntax},
			},
		},
		{
			desc: "request USPrivacy string is different from the one in the GPP string, expect empty policies, signalNo and error",
			in: testInput{
				request: cookieSyncRequest{
					GPP:       "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN",
					GPPSID:    "6",
					USPrivacy: "1YYY",
				},
			},
			expected: testOutput{
				macros:     macros.UserSyncPrivacy{},
				gdprSignal: gdpr.SignalNo,
				policies:   privacy.Policies{},
				err:        errors.New("request.us_privacy consent does not match uspv1"),
			},
		},
		{
			desc: "no issues extracting privacy policies from request GPP and request GPPSid strings",
			in: testInput{
				request: cookieSyncRequest{
					GPP:    "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN",
					GPPSID: "6",
				},
			},
			expected: testOutput{
				macros: macros.UserSyncPrivacy{
					GDPR:        "0",
					GDPRConsent: "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
					USPrivacy:   "1YNN",
					GPP:         "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN",
					GPPSID:      "6",
				},
				gdprSignal: gdpr.SignalNo,
				policies:   privacy.Policies{GPPSID: []int8{6}},
				err:        nil,
			},
		},
	}
	for _, tc := range testCases {
		outMacros, outSignal, outPolicies, outErr := extractPrivacyPolicies(tc.in.request, tc.in.usersyncDefaultGDPRValue)

		assert.Equal(t, tc.expected.macros, outMacros, tc.desc)
		assert.Equal(t, tc.expected.gdprSignal, outSignal, tc.desc)
		assert.Equal(t, tc.expected.policies, outPolicies, tc.desc)
		assert.Equal(t, tc.expected.err, outErr, tc.desc)
	}
}

func TestCookieSyncParseRequest(t *testing.T) {
	expectedCCPAParsedPolicy, _ := ccpa.Policy{Consent: "1NYN"}.Parse(map[string]struct{}{})
	emptyActivityPoliciesRequest := privacy.NewRequestFromPolicies(privacy.Policies{})

	testCases := []struct {
		description          string
		givenConfig          config.UserSync
		givenBody            io.Reader
		givenGDPRConfig      config.GDPR
		givenCCPAEnabled     bool
		givenAccountRequired bool
		expectedError        string
		expectedPrivacy      macros.UserSyncPrivacy
		expectedRequest      usersync.Request
	}{

		{
			description: "Complete Request - includes GPP string with EU TCF V2",
			givenBody: strings.NewReader(`{` +
				`"bidders":["a", "b"],` +
				`"gdpr":1,` +
				`"gdpr_consent":"anyGDPRConsent",` +
				`"us_privacy":"1NYN",` +
				`"gpp":"DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",` +
				`"gpp_sid":"2",` +
				`"limit":42,` +
				`"coopSync":true,` +
				`"filterSettings":{"iframe":{"bidders":"*","filter":"include"}, "image":{"bidders":["b"],"filter":"exclude"}}` +
				`}`),
			givenGDPRConfig:  config.GDPR{Enabled: true, DefaultValue: "0"},
			givenCCPAEnabled: true,
			givenConfig: config.UserSync{
				PriorityGroups: [][]string{{"a", "b", "c"}},
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
				},
			},
			expectedPrivacy: macros.UserSyncPrivacy{
				GDPR:        "1",
				GDPRConsent: "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
				USPrivacy:   "1NYN",
				GPP:         "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
				GPPSID:      "2",
			},
			expectedRequest: usersync.Request{
				Bidders: []string{"a", "b"},
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: 42,
				Privacy: usersyncPrivacy{
					gdprPermissions:  &fakePermissions{},
					ccpaParsedPolicy: expectedCCPAParsedPolicy,
					activityRequest:  privacy.NewRequestFromPolicies(privacy.Policies{GPPSID: []int8{2}}),
					gdprSignal:       1,
				},
				SyncTypeFilter: usersync.SyncTypeFilter{
					IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
					Redirect: usersync.NewSpecificBidderFilter([]string{"b"}, usersync.BidderFilterModeExclude),
				},
				GPPSID: "2",
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
				PriorityGroups: [][]string{{"a", "b", "c"}},
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
				},
			},
			expectedPrivacy: macros.UserSyncPrivacy{
				GDPR:        "1",
				GDPRConsent: "anyGDPRConsent",
				USPrivacy:   "1NYN",
			},
			expectedRequest: usersync.Request{
				Bidders: []string{"a", "b"},
				Cooperative: usersync.Cooperative{
					Enabled:        false,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: 42,
				Privacy: usersyncPrivacy{
					gdprPermissions:  &fakePermissions{},
					ccpaParsedPolicy: expectedCCPAParsedPolicy,
					activityRequest:  emptyActivityPoliciesRequest,
					gdprSignal:       1,
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
			expectedPrivacy:  macros.UserSyncPrivacy{},
			expectedRequest: usersync.Request{
				Limit: math.MaxInt,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
				PriorityGroups: [][]string{{"a", "b", "c"}},
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: true,
				},
			},
			expectedPrivacy: macros.UserSyncPrivacy{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: math.MaxInt,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
				PriorityGroups: [][]string{{"a", "b", "c"}},
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
				},
			},
			expectedPrivacy: macros.UserSyncPrivacy{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        false,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: math.MaxInt,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
				PriorityGroups: [][]string{{"a", "b", "c"}},
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: true,
				},
			},
			expectedPrivacy: macros.UserSyncPrivacy{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        false,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: math.MaxInt,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
				PriorityGroups: [][]string{{"a", "b", "c"}},
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
				},
			},
			expectedPrivacy: macros.UserSyncPrivacy{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        false,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: math.MaxInt,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
				PriorityGroups: [][]string{{"a", "b", "c"}},
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: true,
				},
			},
			expectedPrivacy: macros.UserSyncPrivacy{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: math.MaxInt,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
				PriorityGroups: [][]string{{"a", "b", "c"}},
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
				},
			},
			expectedPrivacy: macros.UserSyncPrivacy{},
			expectedRequest: usersync.Request{
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: math.MaxInt,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
			expectedPrivacy:  macros.UserSyncPrivacy{},
			expectedRequest: usersync.Request{
				Limit: math.MaxInt,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
			expectedPrivacy: macros.UserSyncPrivacy{
				USPrivacy: "1NYN",
			},
			expectedRequest: usersync.Request{
				Limit: math.MaxInt,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
			expectedError:    "JSON parsing failed: expect { or n, but found m",
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
			expectedPrivacy: macros.UserSyncPrivacy{
				GDPR: "0",
			},
			expectedRequest: usersync.Request{
				Limit: math.MaxInt,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      0,
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
			expectedPrivacy: macros.UserSyncPrivacy{
				GDPR: "",
			},
			expectedRequest: usersync.Request{
				Limit: math.MaxInt,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
			givenBody:        iotest.ErrReader(errors.New("anyError")),
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
				PriorityGroups: [][]string{{"a", "b", "c"}},
				Cooperative:    config.UserSyncCooperative{},
			},
			expectedPrivacy: macros.UserSyncPrivacy{},
			expectedRequest: usersync.Request{
				Bidders: []string{"a", "b"},
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: 30,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
				PriorityGroups: [][]string{{"a", "b", "c"}},
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
				},
			},
			expectedPrivacy: macros.UserSyncPrivacy{},
			expectedRequest: usersync.Request{
				Bidders: []string{"a", "b"},
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: 20,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      -1,
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
				PriorityGroups: [][]string{{"a", "b", "c"}},
				Cooperative: config.UserSyncCooperative{
					EnabledByDefault: false,
				},
			},
			expectedPrivacy: macros.UserSyncPrivacy{},
			expectedRequest: usersync.Request{
				Bidders: []string{"a", "b"},
				Cooperative: usersync.Cooperative{
					Enabled:        true,
					PriorityGroups: [][]string{{"a", "b", "c"}},
				},
				Limit: 20,
				Privacy: usersyncPrivacy{
					gdprPermissions: &fakePermissions{},
					activityRequest: emptyActivityPoliciesRequest,
					gdprSignal:      0,
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

		gdprPermsBuilder := fakePermissionsBuilder{
			permissions: &fakePermissions{},
		}.Builder
		tcf2ConfigBuilder := fakeTCF2ConfigBuilder{
			cfg: gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}.Builder

		endpoint := cookieSyncEndpoint{
			config: &config.Configuration{
				UserSync:        test.givenConfig,
				AccountRequired: test.givenAccountRequired,
			},
			privacyConfig: usersyncPrivacyConfig{
				gdprConfig:             test.givenGDPRConfig,
				gdprPermissionsBuilder: gdprPermsBuilder,
				tcf2ConfigBuilder:      tcf2ConfigBuilder,
				ccpaEnforce:            test.givenCCPAEnabled,
			},
			accountsFetcher: FakeAccountsFetcher{AccountData: map[string]json.RawMessage{
				"TestAccount":                   json.RawMessage(`{"cookie_sync": {"default_limit": 20, "max_limit": 30, "default_coop_sync": true}}`),
				"DisabledAccount":               json.RawMessage(`{"disabled":true}`),
				"ValidAccountInvalidActivities": json.RawMessage(`{"privacy":{"allowactivities":{"syncUser":{"rules":[{"condition":{"componentName": ["bidderA.bidderB.bidderC"]}}]}}}}`),
			}},
		}
		assert.NoError(t, endpoint.config.MarshalAccountDefaults())
		request, privacyPolicies, _, err := endpoint.parseRequest(httpRequest)

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

func TestGetEffectiveLimit(t *testing.T) {
	intNegative := ptrutil.ToPtr(-1)
	int0 := ptrutil.ToPtr(0)
	int30 := ptrutil.ToPtr(30)
	int40 := ptrutil.ToPtr(40)
	intMax := ptrutil.ToPtr(math.MaxInt)

	tests := []struct {
		name          string
		reqLimit      *int
		defaultLimit  *int
		expectedLimit int
	}{
		{
			name:          "nil",
			reqLimit:      nil,
			defaultLimit:  nil,
			expectedLimit: math.MaxInt,
		},
		{
			name:          "req_limit_negative",
			reqLimit:      intNegative,
			defaultLimit:  nil,
			expectedLimit: math.MaxInt,
		},
		{
			name:          "req_limit_zero",
			reqLimit:      int0,
			defaultLimit:  nil,
			expectedLimit: math.MaxInt,
		},
		{
			name:          "req_limit_in_range",
			reqLimit:      int30,
			defaultLimit:  nil,
			expectedLimit: 30,
		},
		{
			name:          "req_limit_at_max",
			reqLimit:      intMax,
			defaultLimit:  nil,
			expectedLimit: math.MaxInt,
		},
		{
			name:          "default_limit_negative",
			reqLimit:      nil,
			defaultLimit:  intNegative,
			expectedLimit: math.MaxInt,
		},
		{
			name:          "default_limit_zero",
			reqLimit:      nil,
			defaultLimit:  intNegative,
			expectedLimit: math.MaxInt,
		},
		{
			name:          "default_limit_in_range",
			reqLimit:      nil,
			defaultLimit:  int30,
			expectedLimit: 30,
		},
		{
			name:          "default_limit_at_max",
			reqLimit:      nil,
			defaultLimit:  intMax,
			expectedLimit: math.MaxInt,
		},
		{
			name:          "both_in_range",
			reqLimit:      int30,
			defaultLimit:  int40,
			expectedLimit: 30,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := getEffectiveLimit(test.reqLimit, test.defaultLimit)
			assert.Equal(t, test.expectedLimit, result)
		})
	}
}

func TestGetEffectiveMaxLimit(t *testing.T) {
	intNegative := ptrutil.ToPtr(-1)
	int0 := ptrutil.ToPtr(0)
	int30 := ptrutil.ToPtr(30)
	intMax := ptrutil.ToPtr(math.MaxInt)

	tests := []struct {
		name          string
		maxLimit      *int
		expectedLimit int
	}{
		{
			name:          "nil",
			maxLimit:      nil,
			expectedLimit: math.MaxInt,
		},
		{
			name:          "req_limit_negative",
			maxLimit:      intNegative,
			expectedLimit: math.MaxInt,
		},
		{
			name:          "req_limit_zero",
			maxLimit:      int0,
			expectedLimit: math.MaxInt,
		},
		{
			name:          "req_limit_in_range",
			maxLimit:      int30,
			expectedLimit: 30,
		},
		{
			name:          "req_limit_too_large",
			maxLimit:      intMax,
			expectedLimit: math.MaxInt,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := getEffectiveMaxLimit(test.maxLimit)
			assert.Equal(t, test.expectedLimit, result)
		})
	}
}

func TestSetLimit(t *testing.T) {
	intNegative := ptrutil.ToPtr(-1)
	int0 := ptrutil.ToPtr(0)
	int10 := ptrutil.ToPtr(10)
	int20 := ptrutil.ToPtr(20)
	int30 := ptrutil.ToPtr(30)
	intMax := ptrutil.ToPtr(math.MaxInt)

	tests := []struct {
		name            string
		givenRequest    cookieSyncRequest
		givenAccount    *config.Account
		expectedRequest cookieSyncRequest
	}{
		{
			name: "nil_limits",
			givenRequest: cookieSyncRequest{
				Limit: nil,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultLimit: nil,
					MaxLimit:     nil,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: intMax,
			},
		},
		{
			name: "limit_negative",
			givenRequest: cookieSyncRequest{
				Limit: intNegative,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultLimit: int20,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: intMax,
			},
		},
		{
			name: "limit_zero",
			givenRequest: cookieSyncRequest{
				Limit: int0,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultLimit: int20,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: intMax,
			},
		},
		{
			name: "limit_less_than_max",
			givenRequest: cookieSyncRequest{
				Limit: int10,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultLimit: int20,
					MaxLimit:     int30,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: int10,
			},
		},
		{
			name: "limit_greater_than_max",
			givenRequest: cookieSyncRequest{
				Limit: int30,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{
					DefaultLimit: int20,
					MaxLimit:     int10,
				},
			},
			expectedRequest: cookieSyncRequest{
				Limit: int10,
			},
		},
		{
			name: "limit_at_max",
			givenRequest: cookieSyncRequest{
				Limit: intMax,
			},
			givenAccount: &config.Account{
				CookieSync: config.CookieSync{},
			},
			expectedRequest: cookieSyncRequest{
				Limit: intMax,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			endpoint := cookieSyncEndpoint{}
			request := endpoint.setLimit(test.givenRequest, test.givenAccount.CookieSync)
			assert.Equal(t, test.expectedRequest, request)
		})
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

	mockAnalytics := MockAnalyticsRunner{}
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
			description: "Account Malformed",
			err:         errCookieSyncAccountConfigMalformed,
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordCookieSync", metrics.CookieSyncAccountConfigMalformed).Once()
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

	mockAnalytics := MockAnalyticsRunner{}
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
			given:       []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusBlockedByPrivacy}},
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncPrivacyBlocked).Once()
			},
		},
		{
			description: "One - Blocked By CCPA",
			given:       []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusBlockedByPrivacy}},
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
			description: "One - Rejected By Filter",
			given:       []usersync.BidderEvaluation{{Bidder: "a", SyncerKey: "aSyncer", Status: usersync.StatusRejectedByFilter}},
			setExpectations: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSyncerRequest", "aSyncer", metrics.SyncerCookieSyncRejectedByFilter).Once()
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
		endpoint.writeSyncerMetrics(test.given)

		mockMetrics.AssertExpectations(t)
	}
}

func TestCookieSyncHandleResponse(t *testing.T) {
	syncTypeFilter := usersync.SyncTypeFilter{
		IFrame:   usersync.NewUniformBidderFilter(usersync.BidderFilterModeExclude),
		Redirect: usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude),
	}
	syncTypeExpected := []usersync.SyncType{usersync.SyncTypeRedirect}
	privacyMacros := macros.UserSyncPrivacy{USPrivacy: "anyConsent"}

	// The & in the URL is necessary to test proper JSON encoding.
	syncA := usersync.Sync{URL: "https://syncA.com/sync?a=1&b=2", Type: usersync.SyncTypeRedirect, SupportCORS: true}
	syncerA := MockSyncer{}
	syncerA.On("GetSync", syncTypeExpected, privacyMacros).Return(syncA, nil).Maybe()

	// The & in the URL is necessary to test proper JSON encoding.
	syncB := usersync.Sync{URL: "https://syncB.com/sync?a=1&b=2", Type: usersync.SyncTypeRedirect, SupportCORS: false}
	syncerB := MockSyncer{}
	syncerB.On("GetSync", syncTypeExpected, privacyMacros).Return(syncB, nil).Maybe()

	syncWithError := usersync.Sync{}
	syncerWithError := MockSyncer{}
	syncerWithError.On("GetSync", syncTypeExpected, privacyMacros).Return(syncWithError, errors.New("anyError")).Maybe()

	bidderEvalForDebug := []usersync.BidderEvaluation{
		{Bidder: "Bidder1", Status: usersync.StatusAlreadySynced},
		{Bidder: "Bidder2", Status: usersync.StatusUnknownBidder},
		{Bidder: "Bidder3", Status: usersync.StatusUnconfiguredBidder},
		{Bidder: "Bidder4", Status: usersync.StatusBlockedByPrivacy},
		{Bidder: "Bidder5", Status: usersync.StatusRejectedByFilter},
		{Bidder: "Bidder6", Status: usersync.StatusBlockedByUserOptOut},
		{Bidder: "Bidder7", Status: usersync.StatusBlockedByDisabledUsersync},
		{Bidder: "BidderA", Status: usersync.StatusDuplicate, SyncerKey: "syncerB"},
	}

	testCases := []struct {
		description         string
		givenCookieHasSyncs bool
		givenSyncersChosen  []usersync.SyncerChoice
		givenDebug          bool
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
		{
			description:         "Debug is true, should see all rejected bidder eval statuses in response",
			givenCookieHasSyncs: true,
			givenDebug:          true,
			givenSyncersChosen:  []usersync.SyncerChoice{},
			expectedJSON:        `{"status":"ok","bidder_status":[],"debug":[{"bidder":"Bidder1","error":"Already in sync"},{"bidder":"Bidder2","error":"Unsupported bidder"},{"bidder":"Bidder3","error":"No sync config"},{"bidder":"Bidder4","error":"Rejected by privacy"},{"bidder":"Bidder5","error":"Rejected by request filter"},{"bidder":"Bidder6","error":"Status blocked by user opt out"},{"bidder":"Bidder7","error":"Sync disabled by config"},{"bidder":"BidderA","error":"Duplicate bidder synced as syncerB"}]}` + "\n",
			expectedAnalytics:   analytics.CookieSyncObject{Status: 200, BidderStatus: []*analytics.CookieSyncBidder{}},
		},
	}

	for _, test := range testCases {
		mockAnalytics := MockAnalyticsRunner{}
		mockAnalytics.On("LogCookieSyncObject", &test.expectedAnalytics).Once()

		cookie := usersync.NewCookie()
		if test.givenCookieHasSyncs {
			if err := cookie.Sync("foo", "anyID"); err != nil {
				assert.FailNow(t, test.description+":set_cookie")
			}
		}

		writer := httptest.NewRecorder()
		endpoint := cookieSyncEndpoint{pbsAnalytics: &mockAnalytics}

		var bidderEval []usersync.BidderEvaluation
		if test.givenDebug {
			bidderEval = bidderEvalForDebug
		} else {
			bidderEval = []usersync.BidderEvaluation{}
		}
		endpoint.handleResponse(writer, syncTypeFilter, cookie, privacyMacros, test.givenSyncersChosen, bidderEval, test.givenDebug)

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
		mockPerms.On("HostCookiesAllowed", mock.Anything).Return(test.givenResponse, test.givenError)

		privacy := usersyncPrivacy{
			gdprPermissions: &mockPerms,
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
		mockPerms.On("BidderSyncAllowed", mock.Anything, openrtb_ext.BidderName("foo")).Return(test.givenResponse, test.givenError)

		privacy := usersyncPrivacy{
			gdprPermissions: &mockPerms,
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

func TestCookieSyncActivityControlIntegration(t *testing.T) {
	testCases := []struct {
		name           string
		bidderName     string
		accountPrivacy *config.AccountPrivacy
		expectedResult bool
	}{
		{
			name:           "activity_is_allowed",
			bidderName:     "bidderA",
			accountPrivacy: getDefaultActivityConfig("bidderA", true),
			expectedResult: true,
		},
		{
			name:           "activity_is_denied",
			bidderName:     "bidderA",
			accountPrivacy: getDefaultActivityConfig("bidderA", false),
			expectedResult: false,
		},
		{
			name:           "activity_is_abstain",
			bidderName:     "bidderA",
			accountPrivacy: nil,
			expectedResult: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			activities := privacy.NewActivityControl(test.accountPrivacy)
			up := usersyncPrivacy{
				activityControl: activities,
			}
			actualResult := up.ActivityAllowsUserSync(test.bidderName)
			assert.Equal(t, test.expectedResult, actualResult)
		})
	}
}

func TestUsersyncPrivacyGDPRInScope(t *testing.T) {
	testCases := []struct {
		description     string
		givenGdprSignal gdpr.Signal
		expected        bool
	}{
		{
			description:     "GDPR Signal Yes",
			givenGdprSignal: gdpr.SignalYes,
			expected:        true,
		},
		{
			description:     "GDPR Signal No",
			givenGdprSignal: gdpr.SignalNo,
			expected:        false,
		},
		{
			description:     "GDPR Signal Ambigious",
			givenGdprSignal: gdpr.SignalAmbiguous,
			expected:        false,
		},
	}

	for _, test := range testCases {
		privacy := usersyncPrivacy{
			gdprSignal: test.givenGdprSignal,
		}

		result := privacy.GDPRInScope()
		assert.Equal(t, test.expected, result, test.description)
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
			givenErrorList: []error{&errortypes.AccountDisabled{}},
			expectedError:  errCookieSyncAccountBlocked,
		},
		{
			description:    "Special Case: invalid (rejected via allow list)",
			givenErrorList: []error{&errortypes.AcctRequired{}},
			expectedError:  errCookieSyncAccountInvalid,
		},
		{
			description:    "Special Case: malformed account config",
			givenErrorList: []error{&errortypes.MalformedAcct{}},
			expectedError:  errCookieSyncAccountConfigMalformed,
		},
		{
			description:    "Special Case: multiple special cases, first one wins",
			givenErrorList: []error{&errortypes.AccountDisabled{}, &errortypes.AcctRequired{}, &errortypes.MalformedAcct{}},
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

func (m *MockSyncer) DefaultResponseFormat() usersync.SyncType {
	args := m.Called()
	return args.Get(0).(usersync.SyncType)
}

func (m *MockSyncer) SupportsType(syncTypes []usersync.SyncType) bool {
	args := m.Called(syncTypes)
	return args.Bool(0)
}

func (m *MockSyncer) GetSync(syncTypes []usersync.SyncType, privacyMacros macros.UserSyncPrivacy) (usersync.Sync, error) {
	args := m.Called(syncTypes, privacyMacros)
	return args.Get(0).(usersync.Sync), args.Error(1)
}

type MockAnalyticsRunner struct {
	mock.Mock
}

func (m *MockAnalyticsRunner) LogAuctionObject(obj *analytics.AuctionObject, ac privacy.ActivityControl) {
	m.Called(obj, ac)
}

func (m *MockAnalyticsRunner) LogVideoObject(obj *analytics.VideoObject, ac privacy.ActivityControl) {
	m.Called(obj, ac)
}

func (m *MockAnalyticsRunner) LogCookieSyncObject(obj *analytics.CookieSyncObject) {
	m.Called(obj)
}

func (m *MockAnalyticsRunner) LogSetUIDObject(obj *analytics.SetUIDObject) {
	m.Called(obj)
}

func (m *MockAnalyticsRunner) LogAmpObject(obj *analytics.AmpObject, ac privacy.ActivityControl) {
	m.Called(obj, ac)
}

func (m *MockAnalyticsRunner) LogNotificationEventObject(obj *analytics.NotificationEvent, ac privacy.ActivityControl) {
	m.Called(obj, ac)
}

func (m *MockAnalyticsRunner) Shutdown() {
	m.Called()
}

type MockGDPRPerms struct {
	mock.Mock
}

func (m *MockGDPRPerms) HostCookiesAllowed(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *MockGDPRPerms) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName) (bool, error) {
	args := m.Called(ctx, bidder)
	return args.Bool(0), args.Error(1)
}

func (m *MockGDPRPerms) AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) gdpr.AuctionPermissions {
	args := m.Called(ctx, bidderCoreName, bidder)
	return args.Get(0).(gdpr.AuctionPermissions)
}

type FakeAccountsFetcher struct {
	AccountData map[string]json.RawMessage
}

func (f FakeAccountsFetcher) FetchAccount(ctx context.Context, _ json.RawMessage, accountID string) (json.RawMessage, []error) {
	defaultAccountJSON := json.RawMessage(`{"disabled":false}`)

	if accountID == metrics.PublisherUnknown {
		return defaultAccountJSON, nil
	}
	if account, ok := f.AccountData[accountID]; ok {
		return account, nil
	}
	return nil, []error{errors.New("Account not found")}
}

type fakePermissions struct {
}

func (p *fakePermissions) HostCookiesAllowed(ctx context.Context) (bool, error) {
	return true, nil
}

func (p *fakePermissions) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName) (bool, error) {
	return true, nil
}

func (p *fakePermissions) AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) gdpr.AuctionPermissions {
	return gdpr.AuctionPermissions{
		AllowBidRequest: true,
	}
}

func getDefaultActivityConfig(componentName string, allow bool) *config.AccountPrivacy {
	return &config.AccountPrivacy{
		AllowActivities: &config.AllowActivities{
			SyncUser: config.Activity{
				Default: ptrutil.ToPtr(true),
				Rules: []config.ActivityRule{
					{
						Allow: allow,
						Condition: config.ActivityCondition{
							ComponentName: []string{componentName},
							ComponentType: []string{"bidder"},
						},
					},
				},
			},
		},
	}
}

func TestSetCookieDeprecationHeader(t *testing.T) {
	getTestRequest := func(addCookie bool) *http.Request {
		r := httptest.NewRequest("POST", "/cookie_sync", nil)
		if addCookie {
			r.AddCookie(&http.Cookie{Name: receiveCookieDeprecation, Value: "1"})
		}
		return r
	}

	tests := []struct {
		name                            string
		responseWriter                  http.ResponseWriter
		request                         *http.Request
		account                         *config.Account
		expectedCookieDeprecationHeader bool
	}{
		{
			name:                            "not-present-account-nil",
			request:                         getTestRequest(false),
			responseWriter:                  httptest.NewRecorder(),
			account:                         nil,
			expectedCookieDeprecationHeader: false,
		},
		{
			name:           "not-present-cookiedeprecation-disabled",
			request:        getTestRequest(false),
			responseWriter: httptest.NewRecorder(),
			account: &config.Account{
				Privacy: config.AccountPrivacy{
					PrivacySandbox: config.PrivacySandbox{
						CookieDeprecation: config.CookieDeprecation{
							Enabled: false,
						},
					},
				},
			},
			expectedCookieDeprecationHeader: false,
		},
		{
			name:           "present-cookiedeprecation-disabled",
			request:        getTestRequest(true),
			responseWriter: httptest.NewRecorder(),
			account: &config.Account{
				Privacy: config.AccountPrivacy{
					PrivacySandbox: config.PrivacySandbox{
						CookieDeprecation: config.CookieDeprecation{
							Enabled: false,
						},
					},
				},
			},
			expectedCookieDeprecationHeader: false,
		},
		{
			name:           "present-cookiedeprecation-enabled",
			request:        getTestRequest(true),
			responseWriter: httptest.NewRecorder(),
			account: &config.Account{
				Privacy: config.AccountPrivacy{
					PrivacySandbox: config.PrivacySandbox{
						CookieDeprecation: config.CookieDeprecation{
							Enabled: true,
							TTLSec:  86400,
						},
					},
				},
			},

			expectedCookieDeprecationHeader: false,
		},
		{
			name:                            "present-account-nil",
			request:                         getTestRequest(true),
			responseWriter:                  httptest.NewRecorder(),
			account:                         nil,
			expectedCookieDeprecationHeader: false,
		},
		{
			name:           "not-present-cookiedeprecation-enabled",
			request:        getTestRequest(false),
			responseWriter: httptest.NewRecorder(),
			account: &config.Account{
				Privacy: config.AccountPrivacy{
					PrivacySandbox: config.PrivacySandbox{
						CookieDeprecation: config.CookieDeprecation{
							Enabled: true,
							TTLSec:  86400,
						},
					},
				},
			},
			expectedCookieDeprecationHeader: true,
		},
		{
			name:           "failed-to-read-cookiedeprecation-enabled",
			request:        &http.Request{}, // nil cookie. error: http: named cookie not present
			responseWriter: httptest.NewRecorder(),
			account: &config.Account{
				Privacy: config.AccountPrivacy{
					PrivacySandbox: config.PrivacySandbox{
						CookieDeprecation: config.CookieDeprecation{
							Enabled: true,
							TTLSec:  86400,
						},
					},
				},
			},
			expectedCookieDeprecationHeader: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cookieSyncEndpoint{
				time: &fakeTime{time: time.Date(2024, 2, 22, 9, 42, 4, 13, time.UTC)},
			}
			c.setCookieDeprecationHeader(tt.responseWriter, tt.request, tt.account)
			gotCookie := tt.responseWriter.Header().Get("Set-Cookie")
			if tt.expectedCookieDeprecationHeader {
				wantCookieTTL := c.time.Now().Add(time.Second * time.Duration(86400)).UTC().Format(http.TimeFormat)
				wantCookie := fmt.Sprintf("receive-cookie-deprecation=1; Path=/; Expires=%v; HttpOnly; Secure; SameSite=None; Partitioned;", wantCookieTTL)
				assert.Equal(t, wantCookie, gotCookie, ":set_cookie_deprecation_header")
			} else {
				assert.Empty(t, gotCookie, ":set_cookie_deprecation_header")
			}
		})
	}
}
