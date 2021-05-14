package gdpr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/config"
)

func TestTCF1FetcherInitialLoad(t *testing.T) {
	// Loads two vendor lists during initialization by setting the latest vendor list version to 2.

	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 2,
		vendorLists: map[int]string{
			1: tcf1VendorList1,
			2: tcf1VendorList2,
		},
	})))
	defer server.Close()

	testCases := []test{
		{
			description: "Fallback - Vendor List 1",
			setup: testSetup{
				enableTCF1Fallback: true,
				vendorListVersion:  1,
			},
			expected: vendorListFallbackExpected,
		},
		{
			description: "Fallback - Vendor List 2",
			setup: testSetup{
				enableTCF1Fallback: true,
				vendorListVersion:  2,
			},
			expected: vendorListFallbackExpected,
		},
		{
			description: "No Fallback - Vendor List 1",
			setup: testSetup{
				enableTCF1Fallback: false,
				vendorListVersion:  1,
			},
			expected: testExpected{
				errorMessage: "gdpr vendor list version 1 does not exist, or has not been loaded yet. Try again in a few minutes",
			},
		},
		{
			description: "No Fallback - Vendor List 2",
			setup: testSetup{
				enableTCF1Fallback: false,
				vendorListVersion:  2,
			},
			expected: testExpected{
				errorMessage: "gdpr vendor list version 2 does not exist, or has not been loaded yet. Try again in a few minutes",
			},
		},
	}

	for _, test := range testCases {
		runTestTCF1(t, test, server)
	}
}

func TestTCF2FetcherDynamicLoadListExists(t *testing.T) {
	// Loads the first vendor list during initialization by setting the latest vendor list version to 1.
	// All other vendor lists will be dynamically loaded.

	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]string{
			1: tcf2VendorList1,
			2: tcf2VendorList2,
		},
	})))
	defer server.Close()

	test := test{
		description: "Dynamic Load - List Exists",
		setup: testSetup{
			vendorListVersion: 2,
		},
		expected: vendorList2Expected,
	}

	runTestTCF2(t, test, server)
}

func TestTCF2FetcherDynamicLoadListDoesntExist(t *testing.T) {
	// Loads the first vendor list during initialization by setting the latest vendor list version to 1.
	// All other vendor list load attempts will be done dynamically.

	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]string{
			1: tcf2VendorList1,
		},
	})))
	defer server.Close()

	test := test{
		description: "No Fallback - Vendor Doesn't Exist",
		setup: testSetup{
			vendorListVersion: 2,
		},
		expected: testExpected{
			errorMessage: "gdpr vendor list version 2 does not exist, or has not been loaded yet. Try again in a few minutes",
		},
	}

	runTestTCF2(t, test, server)
}

func TestTCF2FetcherThrottling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]string{
			1: tcf2MarshalVendorList(tcf2VendorList{
				VendorListVersion: 1,
				Vendors:           map[string]*tcf2Vendor{"12": {ID: 12, Purposes: []int{1}}},
			}),
			2: tcf2MarshalVendorList(tcf2VendorList{
				VendorListVersion: 2,
				Vendors:           map[string]*tcf2Vendor{"12": {ID: 12, Purposes: []int{1, 2}}},
			}),
			3: tcf2MarshalVendorList(tcf2VendorList{
				VendorListVersion: 3,
				Vendors:           map[string]*tcf2Vendor{"12": {ID: 12, Purposes: []int{1, 2, 3}}},
			}),
		},
	})))
	defer server.Close()

	fetcher := newVendorListFetcherTCF2(context.Background(), testConfig(), server.Client(), testURLMaker(server))

	// Dynamically Load List 2 Successfully
	_, errList1 := fetcher(context.Background(), 2)
	assert.NoError(t, errList1)

	// Fail To Load List 3 Due To Rate Limiting
	// - The request is rate limited after dynamically list 2.
	_, errList2 := fetcher(context.Background(), 3)
	assert.EqualError(t, errList2, "gdpr vendor list version 3 does not exist, or has not been loaded yet. Try again in a few minutes")
}

func TestTCF2MalformedVendorlist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]string{
			1: "malformed",
		},
	})))
	defer server.Close()

	fetcher := newVendorListFetcherTCF2(context.Background(), testConfig(), server.Client(), testURLMaker(server))
	_, err := fetcher(context.Background(), 1)

	// Fetching should fail since vendor list could not be unmarshalled.
	assert.Error(t, err)
}

func TestTCF2ServerUrlInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	invalidURLGenerator := func(uint16) string { return " http://invalid-url-has-leading-whitespace" }

	fetcher := newVendorListFetcherTCF2(context.Background(), testConfig(), server.Client(), invalidURLGenerator)
	_, err := fetcher(context.Background(), 1)

	assert.EqualError(t, err, "gdpr vendor list version 1 does not exist, or has not been loaded yet. Try again in a few minutes")
}

func TestTCF2ServerUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	fetcher := newVendorListFetcherTCF2(context.Background(), testConfig(), server.Client(), testURLMaker(server))
	_, err := fetcher(context.Background(), 1)

	assert.EqualError(t, err, "gdpr vendor list version 1 does not exist, or has not been loaded yet. Try again in a few minutes")
}

func TestVendorListURLMaker(t *testing.T) {
	testCases := []struct {
		description       string
		vendorListVersion uint16
		expectedURL       string
	}{
		{
			description:       "Latest",
			vendorListVersion: 0,
			expectedURL:       "https://vendor-list.consensu.org/v2/vendor-list.json",
		},
		{
			description:       "Specific",
			vendorListVersion: 42,
			expectedURL:       "https://vendor-list.consensu.org/v2/archives/vendor-list-v42.json",
		},
	}

	for _, test := range testCases {
		result := vendorListURLMaker(test.vendorListVersion)
		assert.Equal(t, test.expectedURL, result)
	}
}

var tcf1VendorList1 = tcf1MarshalVendorList(tcf1VendorList{
	VendorListVersion: 1,
	Vendors:           []tcf1Vendor{{ID: 12, Purposes: []int{2}}},
})

var tcf2VendorList1 = tcf2MarshalVendorList(tcf2VendorList{
	VendorListVersion: 1,
	Vendors:           map[string]*tcf2Vendor{"12": {ID: 12, Purposes: []int{2}}},
})

var tcf1VendorList2 = tcf1MarshalVendorList(tcf1VendorList{
	VendorListVersion: 2,
	Vendors:           []tcf1Vendor{{ID: 12, Purposes: []int{2, 3}}},
})

var tcf2VendorList2 = tcf2MarshalVendorList(tcf2VendorList{
	VendorListVersion: 2,
	Vendors:           map[string]*tcf2Vendor{"12": {ID: 12, Purposes: []int{2, 3}}},
})

var vendorList2Expected = testExpected{
	vendorListVersion: 2,
	vendorID:          12,
	vendorPurposes:    map[int]bool{1: false, 2: true, 3: true},
}

var vendorListFallbackExpected = testExpected{
	vendorListVersion: 215, // Values from hardcoded fallback file.
	vendorID:          12,
	vendorPurposes:    map[int]bool{1: true, 2: false, 3: true},
}

type tcf1VendorList struct {
	VendorListVersion uint16       `json:"vendorListVersion"`
	Vendors           []tcf1Vendor `json:"vendors"`
}

type tcf1Vendor struct {
	ID       uint16 `json:"id"`
	Purposes []int  `json:"purposeIds"`
}

func tcf1MarshalVendorList(vendorList tcf1VendorList) string {
	json, _ := json.Marshal(vendorList)
	return string(json)
}

type tcf2VendorList struct {
	VendorListVersion uint16                 `json:"vendorListVersion"`
	Vendors           map[string]*tcf2Vendor `json:"vendors"`
}

type tcf2Vendor struct {
	ID               uint16 `json:"id"`
	Purposes         []int  `json:"purposes"`
	LegIntPurposes   []int  `json:"legIntPurposes"`
	FlexiblePurposes []int  `json:"flexiblePurposes"`
	SpecialPurposes  []int  `json:"specialPurposes"`
}

func tcf2MarshalVendorList(vendorList tcf2VendorList) string {
	json, _ := json.Marshal(vendorList)
	return string(json)
}

type serverSettings struct {
	vendorListLatestVersion int
	vendorLists             map[int]string
}

// mockServer returns a handler which returns the given response for each global vendor list version.
// The latestVersion param can be used to mock "updates" which occur after PBS has been turned on.
// For example, if latestVersion is 3, but the responses map has data at "4", the server will return
// version "3" when asked for the latest version.
//
// This will help test lazy-fetches for versions which aren't there on app startup.
//
// If the "version" query param doesn't exist, it returns a 400.
//
// If the "version" query param points to a version which doesn't exist, it returns a 403.
// Don't ask why... that's just what the official page is doing. See https://vendor-list.consensu.org/v-9999/vendorlist.json
func mockServer(settings serverSettings) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		vendorListVersion := req.URL.Query().Get("version")
		vendorListVersionInt, err := strconv.Atoi(vendorListVersion)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Request had invalid version: " + vendorListVersion))
			return
		}
		if vendorListVersionInt == 0 {
			vendorListVersionInt = settings.vendorListLatestVersion
		}
		response, ok := settings.vendorLists[vendorListVersionInt]
		if !ok {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Version not found: " + vendorListVersion))
			return
		}
		w.Write([]byte(response))
	}
}

type test struct {
	description string
	setup       testSetup
	expected    testExpected
}

type testSetup struct {
	enableTCF1Fallback bool
	vendorListVersion  uint16
}

type testExpected struct {
	errorMessage      string
	vendorListVersion uint16
	vendorID          uint16
	vendorPurposes    map[int]bool
}

func runTestTCF1(t *testing.T, test test, server *httptest.Server) {
	config := testConfig()
	if test.setup.enableTCF1Fallback {
		config.TCF1.FallbackGVLPath = "../static/tcf1/fallback_gvl.json"
	}

	fetcher := newVendorListFetcherTCF1(config)
	vendorList, err := fetcher(context.Background(), test.setup.vendorListVersion)

	if test.expected.errorMessage != "" {
		assert.EqualError(t, err, test.expected.errorMessage, test.description+":error")
	} else {
		assert.NoError(t, err, test.description+":vendorlist")
		assert.Equal(t, test.expected.vendorListVersion, vendorList.Version(), test.description+":vendorlistid")
		vendor := vendorList.Vendor(test.expected.vendorID)
		for id, expected := range test.expected.vendorPurposes {
			result := vendor.Purpose(consentconstants.Purpose(id))
			assert.Equalf(t, expected, result, "%s:vendor-%d:purpose-%d", test.description, vendorList.Version(), id)
		}
	}
}

func runTestTCF2(t *testing.T, test test, server *httptest.Server) {
	config := testConfig()
	fetcher := newVendorListFetcherTCF2(context.Background(), config, server.Client(), testURLMaker(server))
	vendorList, err := fetcher(context.Background(), test.setup.vendorListVersion)

	if test.expected.errorMessage != "" {
		assert.EqualError(t, err, test.expected.errorMessage, test.description+":error")
	} else {
		assert.NoError(t, err, test.description+":vendorlist")
		assert.Equal(t, test.expected.vendorListVersion, vendorList.Version(), test.description+":vendorlistid")
		vendor := vendorList.Vendor(test.expected.vendorID)
		for id, expected := range test.expected.vendorPurposes {
			result := vendor.Purpose(consentconstants.Purpose(id))
			assert.Equalf(t, expected, result, "%s:vendor-%d:purpose-%d", test.description, vendorList.Version(), id)
		}
	}
}

func testURLMaker(server *httptest.Server) func(uint16) string {
	url := server.URL
	return func(vendorListVersion uint16) string {
		return url + "?version=" + strconv.Itoa(int(vendorListVersion))
	}
}

func testConfig() config.GDPR {
	return config.GDPR{
		Timeouts: config.GDPRTimeouts{
			InitVendorlistFetch:   60 * 1000,
			ActiveVendorlistFetch: 1000 * 5,
		},
		TCF1: config.TCF1{
			FetchGVL: true,
		},
	}
}
