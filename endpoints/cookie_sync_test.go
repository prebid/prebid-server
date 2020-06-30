package endpoints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters/appnexus"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/audienceNetwork"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/lifestreet"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pubmatic"
	analyticsConf "github.com/PubMatic-OpenWrap/prebid-server/analytics/config"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/gdpr"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
	metricsConf "github.com/PubMatic-OpenWrap/prebid-server/pbsmetrics/config"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
	"github.com/buger/jsonparser"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func TestCookieSyncNoCookies(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "audienceNetwork", "random"]}`, nil, true, syncersForTest(), false, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	syncs := parseSyncs(t, rr.Body.Bytes())
	assert.Contains(t, syncs, "appnexus")
	assert.Contains(t, syncs, "audienceNetwork")
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRPreventsCookie(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "pubmatic"]}`, nil, false, syncersForTest(), false, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRPreventsBidders(t *testing.T) {
	rr := doPost(`{"gdpr":1,"bidders":["appnexus", "pubmatic", "lifestreet"],"gdpr_consent":"BOONs2HOONs2HABABBENAGgAAAAPrABACGA"}`, nil, true, map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderLifestreet: lifestreet.NewLifestreetSyncer(template.Must(template.New("sync").Parse("someurl.com"))),
	}, false, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	syncs := parseSyncs(t, rr.Body.Bytes())
	assert.Contains(t, syncs, "lifestreet")
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRIgnoredIfZero(t *testing.T) {
	rr := doPost(`{"gdpr":0,"bidders":["appnexus", "pubmatic"]}`, nil, false, nil, false, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	syncs := parseSyncs(t, rr.Body.Bytes())
	assert.Contains(t, syncs, "appnexus")
	assert.Contains(t, syncs, "pubmatic")
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRConsentRequired(t *testing.T) {
	rr := doPost(`{"gdpr":1,"bidders":["appnexus", "pubmatic"]}`, nil, false, nil, false, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "text/plain; charset=utf-8")
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "gdpr_consent is required if gdpr=1\n", rr.Body.String())
}

func TestCCPA(t *testing.T) {
	testCases := []struct {
		description   string
		requestBody   string
		enforceCCPA   bool
		expectedSyncs []string
	}{
		{
			description:   "Feature Flag On & Opt-Out Yes",
			requestBody:   `{"bidders":["appnexus"], "us_privacy":"1-Y-"}`,
			enforceCCPA:   true,
			expectedSyncs: []string{},
		},
		{
			description:   "Feature Flag Off & Opt-Out Yes",
			requestBody:   `{"bidders":["appnexus"], "us_privacy":"1-Y-"}`,
			enforceCCPA:   false,
			expectedSyncs: []string{"appnexus"},
		},
		{
			description:   "Feature Flag On & Opt-Out No",
			requestBody:   `{"bidders":["appnexus"], "us_privacy":"1-N-"}`,
			enforceCCPA:   false,
			expectedSyncs: []string{"appnexus"},
		},
		{
			description:   "Feature Flag On & Opt-Out Unknown",
			requestBody:   `{"bidders":["appnexus"], "us_privacy":"1---"}`,
			enforceCCPA:   false,
			expectedSyncs: []string{"appnexus"},
		},
		{
			description:   "Feature Flag On & Opt-Out Invalid",
			requestBody:   `{"bidders":["appnexus"], "us_privacy":"invalid"}`,
			enforceCCPA:   false,
			expectedSyncs: []string{"appnexus"},
		},
		{
			description:   "Feature Flag On & Opt-Out Not Provided",
			requestBody:   `{"bidders":["appnexus"]}`,
			enforceCCPA:   false,
			expectedSyncs: []string{"appnexus"},
		},
	}

	for _, test := range testCases {
		gdpr := config.GDPR{UsersyncIfAmbiguous: true}
		ccpa := config.CCPA{Enforce: test.enforceCCPA}
		rr := doConfigurablePost(test.requestBody, nil, true, syncersForTest(), gdpr, ccpa, false, false, false)
		assert.Equal(t, http.StatusOK, rr.Code, test.description+":httpResponseCode")
		assert.ElementsMatch(t, test.expectedSyncs, parseSyncs(t, rr.Body.Bytes()), test.description+":syncs")
		assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()), test.description+":status")
	}
}

func TestCookieSyncHasCookies(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "audienceNetwork", "random"]}`, map[string]string{
		"adnxs":           "1234",
		"audienceNetwork": "2345",
	}, true, syncersForTest(), false, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "ok", parseStatus(t, rr.Body.Bytes()))
}

// Make sure that an empty bidders array returns no syncs
func TestCookieSyncEmptyBidders(t *testing.T) {
	rr := doPost(`{"bidders": []}`, nil, true, syncersForTest(), false, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

// Make sure that all syncs are returned if "bidders" isn't a key
func TestCookieSyncNoBidders(t *testing.T) {
	rr := doPost("{}", nil, true, syncersForTest(), false, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	syncs := parseSyncs(t, rr.Body.Bytes())
	assert.Contains(t, syncs, "appnexus")
	assert.Contains(t, syncs, "audienceNetwork")
	assert.Contains(t, syncs, "lifestreet")
	assert.Contains(t, syncs, "pubmatic")
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestCookieSyncNoCookiesBrokenGDPR(t *testing.T) {
	rr := doConfigurablePost(`{"bidders":["appnexus", "audienceNetwork", "random"],"gdpr_consent":"GLKHGKGKKGK"}`, nil, true, map[openrtb_ext.BidderName]usersync.Usersyncer{}, config.GDPR{UsersyncIfAmbiguous: true}, config.CCPA{}, false, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	syncs := parseSyncs(t, rr.Body.Bytes())
	assert.Contains(t, syncs, "appnexus")
	assert.Contains(t, syncs, "audienceNetwork")
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestCookieSyncWithLimit(t *testing.T) {
	rr := doPost(`{"limit":2}`, nil, true, syncersForTest(), false, false, false)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Len(t, parseSyncs(t, rr.Body.Bytes()), 2, "usersyncs")
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestCookieSyncWithLargeLimit(t *testing.T) {
	syncers := syncersForTest()
	rr := doPost(`{"limit":1000}`, nil, true, syncers, false, false, false)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Len(t, parseSyncs(t, rr.Body.Bytes()), len(syncers), "usersyncs")
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestCookieSyncWithSecureParam(t *testing.T) {
	rr := doPost(`{"bidders":["pubmatic", "random"]}`, nil, true, syncersForTest(),
		true, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	syncs := parseSyncsForSecureFlag(t, rr.Body.Bytes())
	assert.Contains(t, syncs, "pubmatic")
	assert.True(t, isSetSecParam(syncs["pubmatic"]))
}

func TestCookieSyncWithoutSecureParam(t *testing.T) {
	rr := doPost(`{"bidders":["pubmatic", "random"]}`, nil, true, syncersForTest(),
		false, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	syncs := parseSyncsForSecureFlag(t, rr.Body.Bytes())
	assert.Contains(t, syncs, "pubmatic")
	assert.False(t, isSetSecParam(syncs["pubmatic"]))
}

func TestRefererHeader(t *testing.T) {
	rr := doPost(`{"bidders":["pubmatic", "random"]}`, nil, true, syncersForTest(),
		false, true, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	syncs := parseSyncsForSecureFlag(t, rr.Body.Bytes())
	assert.Contains(t, syncs, "pubmatic")
	assert.False(t, isSetSecParam(syncs["pubmatic"]))
}

func TestNoRefererHeader(t *testing.T) {
	rr := doPost(`{"bidders":["pubmatic", "random"]}`, nil, true, syncersForTest(),
		false, false, false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	syncs := parseSyncsForSecureFlag(t, rr.Body.Bytes())
	assert.Contains(t, syncs, "pubmatic")
	assert.False(t, isSetSecParam(syncs["pubmatic"]))
}

func TestSecureRefererHeader(t *testing.T) {
	rr := doPost(`{"bidders":["pubmatic", "random"]}`, nil, true, syncersForTest(),
		false, false, true)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	syncs := parseSyncsForSecureFlag(t, rr.Body.Bytes())
	assert.Contains(t, syncs, "pubmatic")
	assert.True(t, isSetSecParam(syncs["pubmatic"]))
}

//Test that secure flag is getting set for all bidders
func TestCookieSyncWithSecureParamForBidders(t *testing.T) {
	rr := doConfigurablePost(`{"bidders":["appnexus", "audienceNetwork", "random"],"gdpr_consent":"GLKHGKGKKGK"}`,
		nil, true, map[openrtb_ext.BidderName]usersync.Usersyncer{},
		config.GDPR{UsersyncIfAmbiguous: true}, config.CCPA{}, true, false,
		false)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	syncs := parseSyncsForSecureFlag(t, rr.Body.Bytes())
	assert.Contains(t, syncs, "appnexus")
	assert.True(t, isSetSecParam(syncs["appnexus"]))
}

func doPost(body string, existingSyncs map[string]string, gdprHostConsent bool, gdprBidders map[openrtb_ext.BidderName]usersync.Usersyncer, addSecParam bool, addHttpRefererHeader bool, addHttpsRefererHeader bool) *httptest.ResponseRecorder {
	return doConfigurablePost(body, existingSyncs, gdprHostConsent, gdprBidders, config.GDPR{}, config.CCPA{}, addSecParam,
		addHttpRefererHeader, addHttpsRefererHeader)
}

func doConfigurablePost(body string, existingSyncs map[string]string, gdprHostConsent bool, gdprBidders map[openrtb_ext.BidderName]usersync.Usersyncer, cfgGDPR config.GDPR, cfgCCPA config.CCPA, addSecParam bool, addHttpRefererHeader bool, addHttpsRefererHeader bool) *httptest.ResponseRecorder {
	endpoint := testableEndpoint(mockPermissions(gdprHostConsent, gdprBidders), cfgGDPR, cfgCCPA)
	router := httprouter.New()
	router.POST("/cookie_sync", endpoint)
	req, _ := http.NewRequest("POST", "/cookie_sync", strings.NewReader(body))
	if addSecParam {
		q := req.URL.Query()
		q.Add("sec", "1")
		req.URL.RawQuery = q.Encode()
	}
	if addHttpRefererHeader {
		req.Header.Set("Referer", "http://unit-test.com")
	} else if addHttpsRefererHeader {
		req.Header.Set("Referer", "https://unit-test.com")
	}
	if len(existingSyncs) > 0 {

		pcs := usersync.NewPBSCookie()
		for bidder, uid := range existingSyncs {
			pcs.TrySync(bidder, uid)
		}
		req.AddCookie(pcs.ToHTTPCookie(90 * 24 * time.Hour))
	}

	rr := httptest.NewRecorder()
	endpoint(rr, req, nil)
	return rr
}

func testableEndpoint(perms gdpr.Permissions, cfgGDPR config.GDPR, cfgCCPA config.CCPA) httprouter.Handle {
	return NewCookieSyncEndpoint(syncersForTest(), &config.Configuration{GDPR: cfgGDPR, CCPA: cfgCCPA}, perms, &metricsConf.DummyMetricsEngine{}, analyticsConf.NewPBSAnalytics(&config.Analytics{}))
}

func syncersForTest() map[openrtb_ext.BidderName]usersync.Usersyncer {
	return map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderAppnexus:   appnexus.NewAppnexusSyncer(template.Must(template.New("sync").Parse("someurl.com?sec={SecParam}"))),
		openrtb_ext.BidderFacebook:   audienceNetwork.NewFacebookSyncer(template.Must(template.New("sync").Parse("https://www.facebook.com/audiencenetwork/idsync/?partner=partnerId&callback=localhost%2Fsetuid%3Fbidder%3DaudienceNetworksec%3Dsec={SecParam}%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"))),
		openrtb_ext.BidderLifestreet: lifestreet.NewLifestreetSyncer(template.Must(template.New("sync").Parse("anotherurl.com?sec%3D{SecParam}"))),
		openrtb_ext.BidderPubmatic:   pubmatic.NewPubmaticSyncer(template.Must(template.New("sync").Parse("thaturl.com?sec={SecParam}"))),
	}
}

func parseStatus(t *testing.T, responseBody []byte) string {
	t.Helper()
	val, err := jsonparser.GetString(responseBody, "status")
	if err != nil {
		t.Fatalf("response.status was not a string. Error was %v", err)
	}
	return val
}

func parseSyncs(t *testing.T, response []byte) []string {
	t.Helper()
	var syncs []string
	jsonparser.ArrayEach(response, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if dataType != jsonparser.Object {
			t.Errorf("response.bidder_status contained unexpected element of type %v.", dataType)
		}
		if val, err := jsonparser.GetString(value, "bidder"); err != nil {
			t.Errorf("response.bidder_status[?].bidder was not a string. Value was %s", string(value))
		} else {
			syncs = append(syncs, val)
		}
	}, "bidder_status")
	return syncs
}

func parseSyncsForSecureFlag(t *testing.T, response []byte) map[string]string {
	t.Helper()
	var syncs map[string]string = make(map[string]string)
	jsonparser.ArrayEach(response, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if dataType != jsonparser.Object {
			t.Errorf("response.bidder_status contained unexpected element of type %v.", dataType)
		}
		if val, err := jsonparser.GetString(value, "bidder"); err != nil {
			t.Errorf("response.bidder_status[?].bidder was not a string. Value was %s", string(value))
		} else {
			usersyncObj, _, _, err := jsonparser.Get(value, "usersync")
			if err != nil {
				syncs[val] = ""
			} else {
				usrsync_url, err := jsonparser.GetString(usersyncObj, "url")
				if err != nil {
					syncs[val] = ""
				} else {
					syncs[val] = usrsync_url
				}
			}
			//syncs = append(syncs, val)
		}
	}, "bidder_status")
	return syncs
}

func isSetSecParam(syncUrl string) bool {
	u, err := url.Parse(syncUrl)
	if err != nil {
		return false
	}
	q := u.Query()
	isSet := q.Get("sec") == "1"
	return isSet
}

func mockPermissions(allowHost bool, allowedBidders map[openrtb_ext.BidderName]usersync.Usersyncer) gdpr.Permissions {
	return &gdprPerms{
		allowHost:      allowHost,
		allowedBidders: allowedBidders,
	}
}

type gdprPerms struct {
	allowHost      bool
	allowedBidders map[openrtb_ext.BidderName]usersync.Usersyncer
}

func (g *gdprPerms) HostCookiesAllowed(ctx context.Context, consent string) (bool, error) {
	return g.allowHost, nil
}

func (g *gdprPerms) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	_, ok := g.allowedBidders[bidder]
	return ok, nil
}

func (g *gdprPerms) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, consent string) (bool, bool, error) {
	return true, true, nil
}

func (g *gdprPerms) AMPException() bool {
	return false
}

func TestSetSecureParam(t *testing.T) {
	type args struct {
		userSyncUrl string
		isSecure    bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test unescaped with secure = false",
			args: args{"http://testurl.com?sec={SecParam}", false},
			want: "http://testurl.com?sec=0",
		},
		{
			name: "Test unescaped with secure = true",
			args: args{"http://testurl.com?sec={SecParam}", true},
			want: "http://testurl.com?sec=1",
		},
		{
			name: "Test escaped with secure = false",
			args: args{"http://testurl.com?sec%2f%7BSecParam%7D", false},
			want: "http://testurl.com?sec%2f0",
		},
		{
			name: "Test escaped with secure = true",
			args: args{"http://testurl.com?sec%2f%7BSecParam%7D", true},
			want: "http://testurl.com?sec%2f1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setSecureParam(tt.args.userSyncUrl, tt.args.isSecure); got != tt.want {
				t.Errorf("Got: %s, want: %s", got, tt.want)
			}
		})
	}
}
