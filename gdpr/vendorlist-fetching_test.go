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

func TestFetcherDynamicLoadListExists(t *testing.T) {
	// Loads the first vendor list during initialization by setting the latest vendor list version to 1.
	// All other vendor lists will be dynamically loaded.

	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]string{
			1: vendorList1,
			2: vendorList2,
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

	runTest(t, test, server)
}

func TestFetcherDynamicLoadListDoesntExist(t *testing.T) {
	// Loads the first vendor list during initialization by setting the latest vendor list version to 1.
	// All other vendor list load attempts will be done dynamically.

	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]string{
			1: vendorList1,
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

	runTest(t, test, server)
}

func TestFetcherThrottling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]string{
			1: MarshalVendorList(vendorList{
				VendorListVersion: 1,
				Vendors:           map[string]*vendor{"12": {ID: 12, Purposes: []int{1}}},
			}),
			2: MarshalVendorList(vendorList{
				VendorListVersion: 2,
				Vendors:           map[string]*vendor{"12": {ID: 12, Purposes: []int{1, 2}}},
			}),
			3: MarshalVendorList(vendorList{
				VendorListVersion: 3,
				Vendors:           map[string]*vendor{"12": {ID: 12, Purposes: []int{1, 2, 3}}},
			}),
		},
	})))
	defer server.Close()

	fetcher := newVendorListFetcher(context.Background(), testConfig(), server.Client(), testURLMaker(server))

	// Dynamically Load List 2 Successfully
	_, errList1 := fetcher(context.Background(), 2)
	assert.NoError(t, errList1)

	// Fail To Load List 3 Due To Rate Limiting
	// - The request is rate limited after dynamically list 2.
	_, errList2 := fetcher(context.Background(), 3)
	assert.EqualError(t, errList2, "gdpr vendor list version 3 does not exist, or has not been loaded yet. Try again in a few minutes")
}

func TestMalformedVendorlist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]string{
			1: "malformed",
		},
	})))
	defer server.Close()

	fetcher := newVendorListFetcher(context.Background(), testConfig(), server.Client(), testURLMaker(server))
	_, err := fetcher(context.Background(), 1)

	// Fetching should fail since vendor list could not be unmarshalled.
	assert.Error(t, err)
}

func TestServerUrlInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	invalidURLGenerator := func(uint16) string { return " http://invalid-url-has-leading-whitespace" }

	fetcher := newVendorListFetcher(context.Background(), testConfig(), server.Client(), invalidURLGenerator)
	_, err := fetcher(context.Background(), 1)

	assert.EqualError(t, err, "gdpr vendor list version 1 does not exist, or has not been loaded yet. Try again in a few minutes")
}

func TestServerUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	fetcher := newVendorListFetcher(context.Background(), testConfig(), server.Client(), testURLMaker(server))
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

var vendorList1 = MarshalVendorList(vendorList{
	VendorListVersion: 1,
	Vendors:           map[string]*vendor{"12": {ID: 12, Purposes: []int{2}}},
})

var vendorList2 = MarshalVendorList(vendorList{
	VendorListVersion: 2,
	Vendors:           map[string]*vendor{"12": {ID: 12, Purposes: []int{2, 3}}},
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

type vendorList struct {
	VendorListVersion uint16             `json:"vendorListVersion"`
	Vendors           map[string]*vendor `json:"vendors"`
}

type vendor struct {
	ID               uint16 `json:"id"`
	Purposes         []int  `json:"purposes"`
	LegIntPurposes   []int  `json:"legIntPurposes"`
	FlexiblePurposes []int  `json:"flexiblePurposes"`
	SpecialPurposes  []int  `json:"specialPurposes"`
}

func MarshalVendorList(vendorList vendorList) string {
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
	vendorListVersion uint16
}

type testExpected struct {
	errorMessage      string
	vendorListVersion uint16
	vendorID          uint16
	vendorPurposes    map[int]bool
}

func runTest(t *testing.T, test test, server *httptest.Server) {
	config := testConfig()
	fetcher := newVendorListFetcher(context.Background(), config, server.Client(), testURLMaker(server))
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
	}
}
