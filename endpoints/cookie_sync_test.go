package endpoints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/audienceNetwork"
	"github.com/prebid/prebid-server/adapters/pubmatic"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/gdpr"
	metricsConf "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
	"github.com/stretchr/testify/assert"
)

func TestCookieSyncNoCookies(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "audienceNetwork", "random"]}`, nil, true, syncersForTest())
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.ElementsMatch(t, []string{"appnexus", "audienceNetwork"}, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRPreventsCookie(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "pubmatic"]}`, nil, false, syncersForTest())
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRPreventsBidders(t *testing.T) {
	rr := doPost(`{"gdpr":1,"bidders":["appnexus", "pubmatic"],"gdpr_consent":"BOONs2HOONs2HABABBENAGgAAAAPrABACGA"}`, nil, true, map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderPubmatic: pubmatic.NewPubmaticSyncer(template.Must(template.New("sync").Parse("someurl.com"))),
	})
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.ElementsMatch(t, []string{"pubmatic"}, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRIgnoredIfZero(t *testing.T) {
	rr := doPost(`{"gdpr":0,"bidders":["appnexus", "pubmatic"]}`, nil, false, nil)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.ElementsMatch(t, []string{"appnexus", "pubmatic"}, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRConsentRequired(t *testing.T) {
	rr := doPost(`{"gdpr":1,"bidders":["appnexus", "pubmatic"]}`, nil, false, nil)
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
		gdpr := config.GDPR{DefaultValue: "0"}
		ccpa := config.CCPA{Enforce: test.enforceCCPA}
		rr := doConfigurablePost(test.requestBody, nil, true, syncersForTest(), gdpr, ccpa)
		assert.Equal(t, http.StatusOK, rr.Code, test.description+":httpResponseCode")
		assert.ElementsMatch(t, test.expectedSyncs, parseSyncs(t, rr.Body.Bytes()), test.description+":syncs")
		assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()), test.description+":status")
	}
}

func TestCookieSyncHasCookies(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "audienceNetwork", "random"]}`, map[string]string{
		"adnxs":           "1234",
		"audienceNetwork": "2345",
	}, true, syncersForTest())
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "ok", parseStatus(t, rr.Body.Bytes()))
}

// Make sure that an empty bidders array returns no syncs
func TestCookieSyncEmptyBidders(t *testing.T) {
	rr := doPost(`{"bidders": []}`, nil, true, syncersForTest())
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

// Make sure that all syncs are returned if "bidders" isn't a key
func TestCookieSyncNoBidders(t *testing.T) {
	rr := doPost("{}", nil, true, syncersForTest())
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.ElementsMatch(t, []string{"appnexus", "audienceNetwork", "pubmatic"}, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestCookieSyncNoCookiesBrokenGDPR(t *testing.T) {
	rr := doConfigurablePost(`{"bidders":["appnexus", "audienceNetwork", "random"],"gdpr_consent":"GLKHGKGKKGK"}`, nil, true, map[openrtb_ext.BidderName]usersync.Usersyncer{}, config.GDPR{DefaultValue: "0"}, config.CCPA{})
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.ElementsMatch(t, []string{"appnexus", "audienceNetwork"}, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestCookieSyncWithLimit(t *testing.T) {
	rr := doPost(`{"limit":2}`, nil, true, syncersForTest())
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Len(t, parseSyncs(t, rr.Body.Bytes()), 2, "usersyncs")
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestCookieSyncWithLargeLimit(t *testing.T) {
	syncers := syncersForTest()
	rr := doPost(`{"limit":1000}`, nil, true, syncers)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Len(t, parseSyncs(t, rr.Body.Bytes()), len(syncers), "usersyncs")
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func doPost(body string, existingSyncs map[string]string, gdprHostConsent bool, gdprBidders map[openrtb_ext.BidderName]usersync.Usersyncer) *httptest.ResponseRecorder {
	return doConfigurablePost(body, existingSyncs, gdprHostConsent, gdprBidders, config.GDPR{}, config.CCPA{})
}

func doConfigurablePost(body string, existingSyncs map[string]string, gdprHostConsent bool, gdprBidders map[openrtb_ext.BidderName]usersync.Usersyncer, cfgGDPR config.GDPR, cfgCCPA config.CCPA) *httptest.ResponseRecorder {
	endpoint := testableEndpoint(mockPermissions(gdprHostConsent, gdprBidders), cfgGDPR, cfgCCPA)
	router := httprouter.New()
	router.POST("/cookie_sync", endpoint)
	req, _ := http.NewRequest("POST", "/cookie_sync", strings.NewReader(body))
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
	return NewCookieSyncEndpoint(syncersForTest(), &config.Configuration{GDPR: cfgGDPR, CCPA: cfgCCPA}, perms, &metricsConf.DummyMetricsEngine{}, analyticsConf.NewPBSAnalytics(&config.Analytics{}), openrtb_ext.BuildBidderMap())
}

func syncersForTest() map[openrtb_ext.BidderName]usersync.Usersyncer {
	return map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderAppnexus:        appnexus.NewAppnexusSyncer(template.Must(template.New("sync").Parse("someurl.com"))),
		openrtb_ext.BidderAudienceNetwork: audienceNetwork.NewFacebookSyncer(template.Must(template.New("sync").Parse("https://www.facebook.com/audiencenetwork/idsync/?partner=partnerId&callback=localhost%2Fsetuid%3Fbidder%3DaudienceNetwork%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"))),
		openrtb_ext.BidderPubmatic:        pubmatic.NewPubmaticSyncer(template.Must(template.New("sync").Parse("thaturl.com"))),
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

func (g *gdprPerms) HostCookiesAllowed(ctx context.Context, gdprSignal gdpr.Signal, consent string) (bool, error) {
	return g.allowHost, nil
}

func (g *gdprPerms) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, gdprSignal gdpr.Signal, consent string) (bool, error) {
	_, ok := g.allowedBidders[bidder]
	return ok, nil
}

func (g *gdprPerms) AuctionActivitiesAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, gdprSignal gdpr.Signal, consent string, weakVendorEnforcement bool) (allowBidRequest, passGeo bool, passID bool, err error) {
	return true, true, true, nil
}
