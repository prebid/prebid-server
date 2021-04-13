package router

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestExchangeMap(t *testing.T) {
	exchanges := newExchangeMap(&config.Configuration{})
	bidderMap := openrtb_ext.BuildBidderMap()
	for bidderName := range exchanges {
		// OpenRTB doesn't support hardcoded aliases... so this test skips districtm,
		// which was the only alias in the legacy adapter map.
		if _, ok := bidderMap[bidderName]; bidderName != "districtm" && !ok {
			t.Errorf("Bidder %s exists in exchange, but is not a part of the BidderMap.", bidderName)
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

func TestLoadDataCache(t *testing.T) {
	// Test dummy
	if err := loadDataCache(&config.Configuration{
		DataCache: config.DataCache{
			Type: "dummy",
		},
	}, nil); err != nil {
		t.Errorf("data cache: dummy: %s", err)
	}
	// Test postgres error
	if err := loadDataCache(&config.Configuration{
		DataCache: config.DataCache{
			Type: "postgres",
		},
	}, nil); err == nil {
		t.Errorf("data cache: postgres: db nil should return error")
	}
	// Test file
	d, _ := ioutil.TempDir("", "pbs-filecache")
	defer os.RemoveAll(d)
	f, _ := ioutil.TempFile(d, "file")
	defer f.Close()
	if err := loadDataCache(&config.Configuration{
		DataCache: config.DataCache{
			Type:     "filecache",
			Filename: f.Name(),
		},
	}, nil); err != nil {
		t.Errorf("data cache: filecache: %s", err)
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
