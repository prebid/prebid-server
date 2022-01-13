package router

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

const adapterDirectory = "../adapters"

type testValidator struct{}

func (validator *testValidator) Validate(name openrtb_ext.BidderName, ext json.RawMessage) error {
	return nil
}

func (validator *testValidator) Schema(name openrtb_ext.BidderName) string {
	if name == openrtb_ext.BidderAppnexus {
		return "{\"appnexus\":true}"
	} else {
		return "{\"appnexus\":false}"
	}
}

func ensureHasKey(t *testing.T, data map[string]json.RawMessage, key string) {
	t.Helper()
	if _, ok := data[key]; !ok {
		t.Errorf("Expected map to produce a schema for adapter: %s", key)
	}
}

func TestNewJsonDirectoryServer(t *testing.T) {
	alias := map[string]string{"aliastest": "appnexus"}
	handler := NewJsonDirectoryServer("../static/bidder-params", &testValidator{}, alias)
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/whatever", nil)
	handler(recorder, request, nil)

	var data map[string]json.RawMessage
	json.Unmarshal(recorder.Body.Bytes(), &data)

	// Make sure that every adapter has a json schema by the same name associated with it.
	adapterFiles, err := ioutil.ReadDir(adapterDirectory)
	if err != nil {
		t.Fatalf("Failed to open the adapters directory: %v", err)
	}

	for _, adapterFile := range adapterFiles {
		if adapterFile.IsDir() && adapterFile.Name() != "adapterstest" {
			ensureHasKey(t, data, adapterFile.Name())
		}
	}

	ensureHasKey(t, data, "aliastest")
}

func TestApplyBidderInfoConfigOverrides(t *testing.T) {
	var testCases = []struct {
		description         string
		givenBidderInfos    config.BidderInfos
		givenAdaptersCfg    map[string]config.Adapter
		expectedError       string
		expectedBidderInfos config.BidderInfos
	}{
		{
			description:         "Syncer Override",
			givenBidderInfos:    config.BidderInfos{"a": {Syncer: &config.Syncer{Key: "original"}}},
			givenAdaptersCfg:    map[string]config.Adapter{"a": {Syncer: &config.Syncer{Key: "override"}}},
			expectedBidderInfos: config.BidderInfos{"a": {Syncer: &config.Syncer{Key: "override"}}},
		},
		{
			// Adapter Configs use a lower case bidder name, but the Bidder Infos follow the official
			// bidder name casing.
			description:         "Syncer Override - Case Sensitivity",
			givenBidderInfos:    config.BidderInfos{"A": {Syncer: &config.Syncer{Key: "original"}}},
			givenAdaptersCfg:    map[string]config.Adapter{"a": {Syncer: &config.Syncer{Key: "override"}}},
			expectedBidderInfos: config.BidderInfos{"A": {Syncer: &config.Syncer{Key: "override"}}},
		},
		{
			description:         "UserSyncURL Override IFrame",
			givenBidderInfos:    config.BidderInfos{"a": {Syncer: &config.Syncer{IFrame: &config.SyncerEndpoint{URL: "original"}}}},
			givenAdaptersCfg:    map[string]config.Adapter{"a": {UserSyncURL: "override"}},
			expectedBidderInfos: config.BidderInfos{"a": {Syncer: &config.Syncer{IFrame: &config.SyncerEndpoint{URL: "override"}}}},
		},
		{
			description:         "UserSyncURL Supports IFrame",
			givenBidderInfos:    config.BidderInfos{"a": {Syncer: &config.Syncer{Supports: []string{"iframe"}}}},
			givenAdaptersCfg:    map[string]config.Adapter{"a": {UserSyncURL: "override"}},
			expectedBidderInfos: config.BidderInfos{"a": {Syncer: &config.Syncer{Supports: []string{"iframe"}, IFrame: &config.SyncerEndpoint{URL: "override"}}}},
		},
		{
			description:         "UserSyncURL Override Redirect",
			givenBidderInfos:    config.BidderInfos{"a": {Syncer: &config.Syncer{Supports: []string{"redirect"}}}},
			givenAdaptersCfg:    map[string]config.Adapter{"a": {UserSyncURL: "override"}},
			expectedBidderInfos: config.BidderInfos{"a": {Syncer: &config.Syncer{Supports: []string{"redirect"}, Redirect: &config.SyncerEndpoint{URL: "override"}}}},
		},
		{
			description:         "UserSyncURL Supports Redirect",
			givenBidderInfos:    config.BidderInfos{"a": {Syncer: &config.Syncer{Redirect: &config.SyncerEndpoint{URL: "original"}}}},
			givenAdaptersCfg:    map[string]config.Adapter{"a": {UserSyncURL: "override"}},
			expectedBidderInfos: config.BidderInfos{"a": {Syncer: &config.Syncer{Redirect: &config.SyncerEndpoint{URL: "override"}}}},
		},
		{
			description:      "UserSyncURL Override Syncer Not Defined",
			givenBidderInfos: config.BidderInfos{"a": {}},
			givenAdaptersCfg: map[string]config.Adapter{"a": {UserSyncURL: "override"}},
			expectedError:    "adapters.a.usersync_url cannot be applied, bidder does not define a user sync",
		},
		{
			description:      "UserSyncURL Override Syncer Endpoints Not Defined",
			givenBidderInfos: config.BidderInfos{"a": {Syncer: &config.Syncer{}}},
			givenAdaptersCfg: map[string]config.Adapter{"a": {UserSyncURL: "override"}},
			expectedError:    "adapters.a.usersync_url cannot be applied, bidder does not define user sync endpoints and does not define supported endpoints",
		},
		{
			description:      "UserSyncURL Override Ambiguous",
			givenBidderInfos: config.BidderInfos{"a": {Syncer: &config.Syncer{IFrame: &config.SyncerEndpoint{URL: "originalIFrame"}, Redirect: &config.SyncerEndpoint{URL: "originalRedirect"}}}},
			givenAdaptersCfg: map[string]config.Adapter{"a": {UserSyncURL: "override"}},
			expectedError:    "adapters.a.usersync_url cannot be applied, bidder defines multiple user sync endpoints or supports multiple endpoints",
		},
		{
			description:      "UserSyncURL Supports Ambiguous",
			givenBidderInfos: config.BidderInfos{"a": {Syncer: &config.Syncer{Supports: []string{"iframe", "redirect"}}}},
			givenAdaptersCfg: map[string]config.Adapter{"a": {UserSyncURL: "override"}},
			expectedError:    "adapters.a.usersync_url cannot be applied, bidder defines multiple user sync endpoints or supports multiple endpoints",
		},
	}

	for _, test := range testCases {
		resultErr := applyBidderInfoConfigOverrides(test.givenBidderInfos, test.givenAdaptersCfg)
		if test.expectedError == "" {
			assert.NoError(t, resultErr, test.description+":err")
			assert.Equal(t, test.expectedBidderInfos, test.givenBidderInfos, test.description+":result")
		} else {
			assert.EqualError(t, resultErr, test.expectedError, test.description+":err")
		}
	}
}

func TestCheckSupportedUserSyncEndpoints(t *testing.T) {
	anyEndpoint := &config.SyncerEndpoint{URL: "anyURL"}

	var testCases = []struct {
		description      string
		givenBidderInfos config.BidderInfos
		expectedError    string
	}{
		{
			description:      "None",
			givenBidderInfos: config.BidderInfos{},
			expectedError:    "",
		},
		{
			description: "One - No Syncer",
			givenBidderInfos: config.BidderInfos{
				"a": config.BidderInfo{Syncer: nil},
			},
			expectedError: "",
		},
		{
			description: "One - Invalid Supported Endpoint",
			givenBidderInfos: config.BidderInfos{
				"a": config.BidderInfo{Syncer: &config.Syncer{Supports: []string{"invalid"}}},
			},
			expectedError: "failed to load bidder info for a, user sync supported endpoint 'invalid' is unrecognized",
		},
		{
			description: "One - IFrame Supported - Not Specified",
			givenBidderInfos: config.BidderInfos{
				"a": config.BidderInfo{Syncer: &config.Syncer{Supports: []string{"iframe"}, IFrame: nil}},
			},
			expectedError: "",
		},
		{
			description: "One - IFrame Supported - Specified",
			givenBidderInfos: config.BidderInfos{
				"a": config.BidderInfo{Syncer: &config.Syncer{Supports: []string{"iframe"}, IFrame: anyEndpoint}},
			},
			expectedError: "",
		},
		{
			description: "One - Redirect Supported - Not Specified",
			givenBidderInfos: config.BidderInfos{
				"a": config.BidderInfo{Syncer: &config.Syncer{Supports: []string{"redirect"}, Redirect: nil}},
			},
			expectedError: "",
		},
		{
			description: "One - IFrame Supported - Specified",
			givenBidderInfos: config.BidderInfos{
				"a": config.BidderInfo{Syncer: &config.Syncer{Supports: []string{"redirect"}, Redirect: anyEndpoint}},
			},
			expectedError: "",
		},
		{
			description: "One - IFrame + Redirect Supported - Not Specified",
			givenBidderInfos: config.BidderInfos{
				"a": config.BidderInfo{Syncer: &config.Syncer{Supports: []string{"iframe", "redirect"}, IFrame: nil, Redirect: nil}},
			},
			expectedError: "",
		},
		{
			description: "One - IFrame + Redirect Supported - Specified",
			givenBidderInfos: config.BidderInfos{
				"a": config.BidderInfo{Syncer: &config.Syncer{Supports: []string{"iframe", "redirect"}, IFrame: anyEndpoint, Redirect: anyEndpoint}},
			},
			expectedError: "",
		},
		{
			description: "Many - With Invalid Supported Endpoint",
			givenBidderInfos: config.BidderInfos{
				"a": config.BidderInfo{},
				"b": config.BidderInfo{Syncer: &config.Syncer{Supports: []string{"invalid"}}},
			},
			expectedError: "failed to load bidder info for b, user sync supported endpoint 'invalid' is unrecognized",
		},
		{
			description: "Many - Specified + Not Specified",
			givenBidderInfos: config.BidderInfos{
				"a": config.BidderInfo{Syncer: &config.Syncer{Supports: []string{"iframe"}, IFrame: anyEndpoint}},
				"b": config.BidderInfo{Syncer: &config.Syncer{Supports: []string{"redirect"}, Redirect: nil}},
			},
			expectedError: "",
		},
	}

	for _, test := range testCases {
		resultErr := checkSupportedUserSyncEndpoints(test.givenBidderInfos)
		if test.expectedError == "" {
			assert.NoError(t, resultErr, test.description)
		} else {
			assert.EqualError(t, resultErr, test.expectedError, test.description)
		}
	}
}

// Prevents #648
func TestCORSSupport(t *testing.T) {
	const origin = "https://publisher-domain.com"
	handler := func(w http.ResponseWriter, r *http.Request) {}
	cors := SupportCORS(http.HandlerFunc(handler))
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("OPTIONS", "http://some-domain.com/openrtb2/auction", nil)
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "origin")
	req.Header.Set("Origin", origin)

	if !assert.NoError(t, err) {
		return
	}
	cors.ServeHTTP(rr, req)
	assert.Equal(t, origin, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestNoCache(t *testing.T) {
	nc := NoCache{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://localhost/nocache", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("ETag", "abcdef")
	nc.ServeHTTP(rw, req)
	h := rw.Header()
	if expected := "no-cache, no-store, must-revalidate"; expected != h.Get("Cache-Control") {
		t.Errorf("invalid cache-control header: expected: %s got: %s", expected, h.Get("Cache-Control"))
	}
	if expected := "no-cache"; expected != h.Get("Pragma") {
		t.Errorf("invalid pragma header: expected: %s got: %s", expected, h.Get("Pragma"))
	}
	if expected := "0"; expected != h.Get("Expires") {
		t.Errorf("invalid expires header: expected: %s got: %s", expected, h.Get("Expires"))
	}
	if expected := ""; expected != h.Get("ETag") {
		t.Errorf("invalid etag header: expected: %s got: %s", expected, h.Get("ETag"))
	}
}

var testDefReqConfig = config.DefReqConfig{
	Type: "file",
	FileSystem: config.DefReqFiles{
		FileName: "test_aliases.json",
	},
	AliasInfo: true,
}

func TestLoadDefaultAliases(t *testing.T) {
	defAliases, aliasJSON := readDefaultRequest(testDefReqConfig)
	expectedJSON := []byte(`{"ext":{"prebid":{"aliases": {"test1": "appnexus", "test2": "rubicon", "test3": "openx"}}}}`)
	expectedAliases := map[string]string{
		"test1": "appnexus",
		"test2": "rubicon",
		"test3": "openx",
	}

	assert.JSONEq(t, string(expectedJSON), string(aliasJSON))
	assert.Equal(t, expectedAliases, defAliases)
}

func TestLoadDefaultAliasesNoInfo(t *testing.T) {
	noInfoConfig := testDefReqConfig
	noInfoConfig.AliasInfo = false
	defAliases, aliasJSON := readDefaultRequest(noInfoConfig)
	expectedJSON := []byte(`{"ext":{"prebid":{"aliases": {"test1": "appnexus", "test2": "rubicon", "test3": "openx"}}}}`)
	expectedAliases := map[string]string{}

	assert.JSONEq(t, string(expectedJSON), string(aliasJSON))
	assert.Equal(t, expectedAliases, defAliases)
}

func TestValidateDefaultAliases(t *testing.T) {
	var testCases = []struct {
		description   string
		givenAliases  map[string]string
		expectedError string
	}{
		{
			description:   "None",
			givenAliases:  map[string]string{},
			expectedError: "",
		},
		{
			description:   "Valid",
			givenAliases:  map[string]string{"aAlias": "a"},
			expectedError: "",
		},
		{
			description:   "Invalid",
			givenAliases:  map[string]string{"all": "a"},
			expectedError: "default request alias errors (1 error):\n  1: alias all is a reserved bidder name and cannot be used\n",
		},
		{
			description:   "Mixed",
			givenAliases:  map[string]string{"aAlias": "a", "all": "a"},
			expectedError: "default request alias errors (1 error):\n  1: alias all is a reserved bidder name and cannot be used\n",
		},
	}

	for _, test := range testCases {
		err := validateDefaultAliases(test.givenAliases)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}
	}
}
