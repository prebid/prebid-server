package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/stretchr/testify/assert"
)

const adapterDirectory = "../adapters"

type testValidator struct{}

func TestMain(m *testing.M) {
	jsoniter.RegisterExtension(&jsonutil.RawMessageExtension{})
	os.Exit(m.Run())
}

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
	alias := map[openrtb_ext.BidderName]openrtb_ext.BidderName{openrtb_ext.BidderName("alias"): openrtb_ext.BidderName("parentAlias")}
	handler := newJsonDirectoryServer("../static/bidder-params", &testValidator{}, alias)
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/whatever", nil)
	handler(recorder, request, nil)

	var data map[string]json.RawMessage
	jsonutil.UnmarshalValid(recorder.Body.Bytes(), &data)

	// Make sure that every adapter has a json schema by the same name associated with it.
	adapterFiles, err := os.ReadDir(adapterDirectory)
	if err != nil {
		t.Fatalf("Failed to open the adapters directory: %v", err)
	}

	for _, adapterFile := range adapterFiles {
		if adapterFile.IsDir() && adapterFile.Name() != "adapterstest" {
			ensureHasKey(t, data, adapterFile.Name())
		}
	}

	ensureHasKey(t, data, "alias")
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

func TestBidderParamsCompactedOutput(t *testing.T) {
	expectedFormattedResponse := `{"appnexus":{"$schema":"http://json-schema.org/draft-04/schema#","title":"Sample schema","description":"A sample schema to test the bidder/params endpoint","type":"object","properties":{"integer_param":{"type":"integer","minimum":1,"description":"A customer id"},"string_param_1":{"type":"string","minLength":1,"description":"Text with blanks in between"},"string_param_2":{"type":"string","minLength":1,"description":"Text_with_no_blanks_in_between"}},"required":["integer_param","string_param_2"]}}`

	// Setup
	inSchemaDirectory := "bidder_params_tests"
	paramsValidator, err := openrtb_ext.NewBidderParamsValidator(inSchemaDirectory)
	assert.NoError(t, err, "Error initialing validator")

	handler := newJsonDirectoryServer(inSchemaDirectory, paramsValidator, nil)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "/bidder/params", nil)
	assert.NoError(t, err, "Error creating request")

	// Run
	handler(recorder, request, nil)

	// Assertions
	assert.Equal(t, expectedFormattedResponse, recorder.Body.String())
}
